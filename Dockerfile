FROM golang:alpine AS builder
LABEL maintainer="Nicholas Gasior <nicholas@laatu.org>"

RUN apk add --update git bash openssh make

WORKDIR /go/src/github.com/nicholasgasior/buildtrigger
COPY . .
RUN make tools
RUN make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /bin
COPY --from=builder /go/bin/linux/buildtrigger .

ENTRYPOINT ["/bin/buildtrigger"]
