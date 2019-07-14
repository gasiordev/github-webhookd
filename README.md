# github-webhookd
Tiny API that triggers Jenkins builds from GitHub Webhook

## CLI
The following CLI commands are available:
```
github-webhookd start --config=PATH_TO_CONFIG_FILE
```

## Building
Ensure you have your
[workspace directory](https://golang.org/doc/code.html#Workspaces) created.
Change directory to $GOPATH/github.com/gen64/github-webhookd and run
the following commands:

```
make tools
make
```

Binary files will be build in `$GOPATH/bin/linux` and `$GOPATH/bin/darwin`
directories.

## Configuration
Look at `config-sample.json` to see how the configuration file look like. It has
changed since previous version a lot.

Now, all Jenkins details are now described in `jenkins` section. It contains 4
keys: `user`, `token`, `base_url` and `endpoints`. First two are obvious,
`base_url` is prefix for your endpoints.
`endpoints` is an array that contains objects as the following example:
```
{
  "id": "multibranch_pipeline_scan",
  "path": "/job/{{.repository}}_multibranch/build",
  "retry": {
    "delay": "10",
    "count": "5"
  },
  "success": {
    "http_status": "200"
  }
}
```
Keys of `retry` and `success` are optional. First one determines what is the
maximum number application should retry posting to and endpoint and what should
be the delay between retries. The `success` with `http_status` defined expected
HTTP Status Code (eg. 200 or 201). If different then request is considered a
failure (and will be retries if set to do so).

In `path`, any occurrence of `{{.repository}}` and `{{.branch}}` will be
replaced with repository and branch names.

In above example, application will make a `POST` request to
`base_url`+`path`.

## Running
Execute the binary, eg.

```
./github-webhookd start --config=PATH_TO_CONFIG_FILE
```

## Development
Follow the steps mentioned in `Building` section. Additionally, there are
commands that might be useful:

* `make fmt` will use `gofmt` to reformat the code;
* `make fmtcheck` will use `gofmt` to check the code intending.
