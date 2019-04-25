## BuildKit wrapper
Usage (Linux amd64 only):

```
docker run -d --privileged -p 1234:1234 --name buildkit moby/buildkit:latest --addr tcp://0.0.0.0:1234 --oci-worker-platform linux/amd64
export BUILDKIT_HOST=tcp://0.0.0.0:1234
docker create --name buildkit-wrapper korekontrol/buildkit-wrapper
docker cp buildkit-wrapper:/buildkit-build .
./buildkit-build -t <image_name> (...)
```

The syntax of `./build` should be the same as syntax of `docker build`. Currently only limited set of `docker build` options
is supported.


## Credits

Adapted by [Marek Obuchowicz](https://github.com/marek-obuchowicz) from [KoreKontrol](https://www.korekontrol.eu/).
Repository is a fork of moby/buildkit and is based on examples provided there.

## License
[APACHE](LICENSE)
