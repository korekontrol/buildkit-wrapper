package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containerd/console"
	"github.com/moby/buildkit/client"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/moby/buildkit/util/appdefaults"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/sync/errgroup"
)

func main() {
	app := cli.NewApp()
	app.Name = "buildkit-build"
	app.Usage = "Wrapper for buildkit for building containers in a way similar to docker"
	app.UsageText = `buildkit-build [OPTIONS] PATH | URL | -`
	app.Description = `
build container using BuildKit, based on Dockerfile.

This command mimics behavior of "docker build" command so that you can easily get started with BuildKit.
This command is NOT the replacement of "docker build", and should NOT be used for building production images.
It supports only limited set of options to docker.

The resulting image is loaded to Docker.
`
	dockerIncompatibleFlags := []cli.Flag{
		cli.StringFlag{
			Name:   "buildkit-addr",
			Usage:  "buildkit daemon address",
			EnvVar: "BUILDKIT_HOST",
			Value:  appdefaults.Address,
		},
		cli.BoolFlag{
			Name:   "clientside-frontend",
			Usage:  "run dockerfile frontend client side, rather than builtin to buildkitd",
			EnvVar: "BUILDKIT_CLIENTSIDE_FRONTEND",
		},
		cli.StringFlag{
			Name:   "local-cache-import",
			Usage:  "import cache from local directory",
			EnvVar: "BUILDKIT_LOCAL_CACHE_IMPORT",
		},
		cli.StringFlag{
			Name:   "local-cache-export",
			Usage:  "export cache to local directory",
			EnvVar: "BUILDKIT_LOCAL_CACHE_EXPORT",
		},
		cli.StringFlag{
			Name: "progress",
			Usage: "Set type of progress output",
		},
	}
	app.Flags = append([]cli.Flag{
		cli.StringSliceFlag{
			Name:  "build-arg",
			Usage: "Set build-time variables",
		},
		cli.StringFlag{
			Name:  "file, f",
			Usage: "Name of the Dockerfile (Default is 'PATH/Dockerfile')",
		},
		cli.StringFlag{
			Name:  "tag, t",
			Usage: "Name and optionally a tag in the 'name:tag' format",
		},
		cli.StringFlag{
			Name:  "target",
			Usage: "Set the target build stage to build.",
		},
		cli.BoolFlag{
			Name:  "no-cache",
			Usage: "Do not use cache when building the image",
		},
		cli.StringSliceFlag{
			Name:  "label",
			Usage: "Set build labels",
		},
	}, dockerIncompatibleFlags...)
	app.Action = action
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func action(clicontext *cli.Context) error {
	ctx := appcontext.Context()

	if tag := clicontext.String("tag"); tag == "" {
		return errors.New("tag is not specified")
	}
	c, err := client.New(ctx, clicontext.String("buildkit-addr"), client.WithFailFast())
	if err != nil {
		return err
	}
	pipeR, pipeW := io.Pipe()
	solveOpt, err := newSolveOpt(clicontext, pipeW)
	if err != nil {
		return err
	}
	ch := make(chan *client.SolveStatus)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var err error
		if clicontext.Bool("clientside-frontend") {
			_, err = c.Build(ctx, *solveOpt, "", dockerfile.Build, ch)
		} else {
			_, err = c.Solve(ctx, nil, *solveOpt, ch)
		}
		return err
	})
	eg.Go(func() error {
		var c console.Console
		progressOpt := clicontext.String("progress")

		switch progressOpt {
		case "auto", "tty":
			cf, err := console.ConsoleFromFile(os.Stderr)
			if err != nil && progressOpt == "tty" {
				return err
			}
			c = cf
		case "plain":
		default:
			return errors.Errorf("invalid progress value : %s", progressOpt)
		}

		return progressui.DisplaySolveStatus(context.TODO(), "", c, os.Stdout, ch)
	})
	eg.Go(func() error {
		if err := loadDockerTar(pipeR); err != nil {
			return err
		}
		return pipeR.Close()
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	logrus.Infof("Loaded the image %q to Docker.", clicontext.String("tag"))
	return nil
}

func newSolveOpt(clicontext *cli.Context, w io.WriteCloser) (*client.SolveOpt, error) {
	buildCtx := clicontext.Args().First()
	if buildCtx == "" {
		return nil, errors.New("please specify build context (e.g. \".\" for the current directory)")
	} else if buildCtx == "-" {
		return nil, errors.New("stdin not supported yet")
	}

	file := clicontext.String("file")
	if file == "" {
		file = filepath.Join(buildCtx, "Dockerfile")
	}
	localDirs := map[string]string{
		"context":    buildCtx,
		"dockerfile": filepath.Dir(file),
	}

	frontend := "dockerfile.v0" // TODO: use gateway
	if clicontext.Bool("clientside-frontend") {
		frontend = ""
	}
	frontendAttrs := map[string]string{
		"filename": filepath.Base(file),
	}
	if target := clicontext.String("target"); target != "" {
		frontendAttrs["target"] = target
	}
	if clicontext.Bool("no-cache") {
		frontendAttrs["no-cache"] = ""
	}
	for _, buildArg := range clicontext.StringSlice("build-arg") {
		kv := strings.SplitN(buildArg, "=", 2)
		if len(kv) != 2 {
			return nil, errors.Errorf("invalid build-arg value %s", buildArg)
		}
		frontendAttrs["build-arg:"+kv[0]] = kv[1]
	}
	for _, label := range clicontext.StringSlice("label") {
		kv := strings.SplitN(label, "=", 2)
		if len(kv) != 2 {
			return nil, errors.Errorf("invalid label value %s", label)
		}
		frontendAttrs["label:"+kv[0]] = kv[1]
	}

	var cacheImports []client.CacheOptionsEntry
	if localCacheImport := clicontext.String("local-cache-import"); localCacheImport != "" {
		if _, err := os.Stat(localCacheImport + "/index.json"); os.IsNotExist(err) {
			logrus.Warn("Local cache import specified, but cache contents not found. Not importing local cache.")
		} else {
			cacheImports = append(cacheImports, client.CacheOptionsEntry{
				Type: "local",
				Attrs: map[string]string{
					"src": localCacheImport,
				},
			})
		}
	}

	var cacheExports []client.CacheOptionsEntry
	if localCacheExport := clicontext.String("local-cache-export"); localCacheExport != "" {
		cacheExports = append(cacheExports, client.CacheOptionsEntry{
			Type: "local",
			Attrs: map[string]string {
				"dest": localCacheExport,
			},
		})
	}

	return &client.SolveOpt{
		Exports: []client.ExportEntry{
			{
				// Export docker image to stdout, will be imported by docker with pipe " | docker load"
				Type: "docker",
				Attrs: map[string]string{
					"name": clicontext.String("tag"),
				},
				Output: w,
			},
			//{
			//	// Containerd image store
			//	Type: "image",
			//	Attrs: map[string]string{
			//		"name": clicontext.String("tag"),
			//	},
			//},

		},
		LocalDirs:     localDirs,
		Frontend:      frontend,
		FrontendAttrs: frontendAttrs,
		CacheImports:  cacheImports,
		CacheExports:  cacheExports,
	}, nil
}

func loadDockerTar(r io.Reader) error {
	// no need to use moby/moby/client here
	cmd := exec.Command("docker", "load")
	cmd.Stdin = r
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
