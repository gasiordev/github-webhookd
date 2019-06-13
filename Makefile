VERSION?=$$(cat version.go | grep VERSION | cut -d"=" -f2 | sed 's/"//g' | sed 's/ //g')
GOFMT_FILES?=$$(find . -name '*.go')
PROJECT_BIN?=buildtrigger
PROJECT_SRC?=github.com/nicholasgasior/buildtrigger

default: build

tools:
	GO111MODULE=off go get -u github.com/nicholasgasior/go-cli
	GO111MODULE=off go get -u github.com/gorilla/mux

guard-%:
	@ if [ "${${*}}" = "" ]; then \
		echo "Environment variable $* not set"; \
		exit 1; \
	fi

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@ gofmt_files=$$(gofmt -l $(GOFMT_FILES)); \
	if [[ -n $${gofmt_files} ]]; then \
		echo "The following files fail gofmt:"; \
		echo "$${gofmt_files}"; \
		echo "Run \`make fmt\` to fix this."; \
		exit 1; \
	fi

build: guard-GOPATH
	mkdir -p $$GOPATH/bin/linux
	mkdir -p $$GOPATH/bin/darwin
	GOOS=linux GOARCH=amd64 go build -v -o $$GOPATH/bin/linux/${PROJECT_BIN} $$GOPATH/src/${PROJECT_SRC}/*.go
	GOOS=darwin GOARCH=amd64 go build -v -o $$GOPATH/bin/darwin/${PROJECT_BIN} $$GOPATH/src/${PROJECT_SRC}/*.go

release: build
	mkdir -p $$GOPATH/releases
	tar -cvzf $$GOPATH/releases/${PROJECT_BIN}-${VERSION}-linux-amd64.tar.gz -C $$GOPATH/bin/linux ${PROJECT_BIN}
	tar -cvzf $$GOPATH/releases/${PROJECT_BIN}-${VERSION}-darwin-amd64.tar.gz -C $$GOPATH/bin/darwin ${PROJECT_BIN}

.NOTPARALLEL:

.PHONY: tools fmt build
