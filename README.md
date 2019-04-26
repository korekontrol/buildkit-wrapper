## BuildKit wrapper

### Usage:

##### Start buildkit container:

```
# Start buildkit container
docker run -d --rm --privileged \
  -p 1234:1234 --name buildkit moby/buildkit:latest \
  --addr tcp://0.0.0.0:1234 --oci-worker-platform linux/amd64
```
To run buildkit with `containerd` - use the following command to start `buildkit` container instead:
```
docker run -d --rm --privileged \
  -v /var/lib/buildkit:/var/lib/buildkit \
  -v /var/lib/containerd:/var/lib/containerd \
  -v /run/containerd:/run/containerd \
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

##### Configure buildkit endpoint
```
export BUILDKIT_HOST=docker://buildkit
```

##### Build a container
```
./buildkit-build -t <image_name> (...)
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
