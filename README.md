# buildtrigger
Tiny API that triggers Jenkins builds from GitHub Webhook

## CLI
The following CLI commands are available:
```
buildtrigger start --config=PATH_TO_CONFIG_FILE
```

## Building
Ensure you have your
[workspace directory](https://golang.org/doc/code.html#Workspaces) created.
Change directory to $GOPATH/github.com/nicholasgasior/buildtrigger and run
the following commands:

```
make tools
make
```

Binary files will be build in `$GOPATH/bin/linux` and `$GOPATH/bin/darwin`
directories.

## Configuration
Look at `config-sample.json` to see how the configuration file look like.

Keys:
* `port` - port on which API should listen for connections;
* `jenkins_url` - Jenkins URL;
* `jenkins_user` - user to access Jenkins;
* `jenkins_token` - password to access Jenkins.

## Running
Execute the binary, eg.

```
./buildtrigger start --config=PATH_TO_CONFIG_FILE
```

## Development
Follow the steps mentioned in `Building` section. Additionally, there are
commands that might be useful:

* `make fmt` will use `gofmt` to reformat the code;
* `make fmtcheck` will use `gofmt` to check the code intending.

## TODO
The following features are planned:
* list of Jenkins URL's that are triggered as a list in configuration file
instead of being hardcoded;
* each Jenkins URL to have number of retries, required response HTTP status etc.
