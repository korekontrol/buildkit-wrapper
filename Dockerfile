FROM golang:1.12 AS gobuild-base
WORKDIR /go/src/github.com/moby/buildkit
COPY . .
RUN go build ./examples/build-using-dockerfile
RUN ls -la && false

FROM scratch AS result
COPY --from=gobuild-base build-using-dockerfile /build
