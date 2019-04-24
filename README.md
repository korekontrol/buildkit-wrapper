## BuildKit wrapper
Usage (Linux amd64 only):

```
docker create --name buildkit-wrapper korekontrol/buildkit-wrapper
docker cp buildkit-wrapper:/build .
./build ...
```
