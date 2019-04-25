FROM golang:1.12 AS gobuild-base
WORKDIR /go/src/github.com/moby/buildkit
COPY . .
ENV CGO_ENABLED 0
ENV GOOS linux
RUN go build -a -ldflags '-extldflags "-static"' ./examples/build-using-dockerfile

FROM scratch AS result
COPY --from=gobuild-base /go/src/github.com/moby/buildkit/build-using-dockerfile /build
CMD /build
