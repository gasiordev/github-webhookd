FROM golang:alpine AS builder
LABEL maintainer="Mikolaj Gasior <miko@gen64.pl>"

RUN apk add --update git bash openssh make

WORKDIR /go/src/github.com/gen64/github-webhookd
COPY . .
RUN make tools
RUN make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /bin
COPY --from=builder /go/bin/linux/github-webhookd .

ENTRYPOINT ["/bin/github-webhookd"]
