# Docker image contains source code and binaries.
FROM golang:alpine
LABEL maintainer="Nicholas Gasior <nicholas@laatu.org>"

RUN apk add --update git bash openssh make

WORKDIR $GOPATH/src/github.com/nicholasgasior/buildtrigger
COPY . .
RUN make tools
RUN make build

WORKDIR $GOPATH
ENTRYPOINT ["bin/linux/buildtrigger"]
