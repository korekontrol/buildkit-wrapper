## BuildKit wrapper
This wrapper uses internal buildkit API and enables more customised build process than offered by `docker build`.
It is using `containerd` for storing outputs (images). Such approach allows importing and exporting contents of build cache layers,
which can be useful on distributed build systems.


### Usage:

##### Start buildkit container:

```
docker run -d --rm --privileged \
  -v /run/buildkit:/run/buildkit
  -v /var/lib/buildkit:/var/lib/buildkit \
  -v /run/containerd:/run/containerd \
  -v /var/lib/containerd:/var/lib/containerd \
  -v /tmp:/tmp \
  --name buildkit moby/buildkit:latest \
  --oci-worker=false --containerd-worker=true
```

##### Download and extract buildkit-build
The binary version works on Linux/AMD64 only.
```
docker create --name buildkit-wrapper korekontrol/buildkit-wrapper
docker cp buildkit-wrapper:/buildkit-build .
docker rm buildkit-wrapper
```

##### Build an image
Setup buildkit endpoint configuration:
```
export BUILDKIT_HOST=tcp://0.0.0.0:1234
```
For import or export cache, add environmental variables:

```
export BUILDKIT_LOCAL_CACHE_IMPORT=/tmp/docker-layers
export BUILDKIT_LOCAL_CACHE_EXPORT=/tmp/docker-layers
```
Run the build:

```
./buildkit-build -t <image_name> (...)
```
This process is executed using standalone buildkit, running in a container. It is not the same as
building container using buildkit bundled in docker.
Note that resulting container will be persisted in local containerd image repository,
so it will not be available to docker. If you want to use it in docker, you need to export/import image.


##### Export containerd image into docker
```
ctr --namespace buildkit images export - <image_name> \
  | docker image import - <image_name>
```


### Syntax

The syntax of `./buildkit-build` *should* be the same as syntax of `docker build`.
Currently, only limited set of `docker build` options is supported:

|  Option  |  Description  |
|---|---|
| `--file <filename>`| (required) Path to Dockerfile | 
| `--tag <value>` | (required) Resulting image name/tag |
| `--build-arg <name>=<value>`  | Add build argument value |
| `--label <name>=<value>` | Add label value |
| `--target <value>` | Target name (inside Dockerfile) |
| `--progress <value>`  | `plain`, `tty` or `auto` |


## Credits

Adapted by [Marek Obuchowicz](https://github.com/marek-obuchowicz) from [KoreKontrol](https://www.korekontrol.eu/).
Repository is a fork of [moby/buildkit](https://github.com/moby/buildkit) and is based on examples provided there.

## License
[APACHE](LICENSE)
