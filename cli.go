package main

import (
	"fmt"
	gocli "github.com/gasiordev/go-cli"
	"os"
)

type CLI struct {
	gocli *gocli.CLI
	app   *App
}

func NewCLI() *CLI {
	cli := &CLI{}
	return cli
}

func (cli *CLI) Init(app *App) {
	cli.app = app

	cli.gocli = gocli.NewCLI("github-webhookd", "Tiny API to receive GitHub Webhooks and trigger Jenkins jobs", "Nicholas Gasior <nicholas@gasior.dev>")

	cmdStart := cli.gocli.AddCmd("start", "Starts API", cli.startHandler)
	cmdStart.AddFlag("config", "Config file", gocli.CLIFlagTypePathFile|gocli.CLIFlagMustExist|gocli.CLIFlagRequired)

	_ = cli.gocli.AddCmd("version", "Prints version", cli.versionHandler)

	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		os.Args = []string{"App", "version"}
	}
}

func (cli *CLI) Run(app *App) {
	cli.Init(app)
	os.Exit(cli.gocli.Run(os.Stdout, os.Stderr))
}

func (cli *CLI) startHandler(c *gocli.CLI) int {
	cli.app.Init(c.Flag("config"))
	return cli.app.Start()
}

func (cli *CLI) versionHandler(c *gocli.CLI) int {
	fmt.Fprintf(os.Stdout, VERSION+"\n")
	return 0
}
