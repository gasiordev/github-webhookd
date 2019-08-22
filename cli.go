package main

import (
	"fmt"
	"github.com/gen64/go-cli"
	"os"
)

func getCLIStartTriggerHandler(trig *BuildTrigger) func(*cli.CLI) int {
	fn := func(c *cli.CLI) int {
		trig.Init(c.Flag("config"))
		return trig.Start()
	}

	return fn
}

func getCLIVersionHandler(trig *BuildTrigger) func(*cli.CLI) int {
	fn := func(c *cli.CLI) int {
		fmt.Fprintf(os.Stdout, VERSION+"\n")
		return 0
	}
	return fn
}

func NewBuildTriggerCLI(trig *BuildTrigger) *cli.CLI {
	BuildTriggerCLI := cli.NewCLI("github-webhookd", "Tiny API to receive GitHub Webhooks and trigger Jenkins jobs", "Mikolaj Gasior <mg@gen64.pl>")

	cmdStart := BuildTriggerCLI.AddCmd("start", "Starts API", getCLIStartTriggerHandler(trig))
	cmdStart.AddFlag("config", "Config file", cli.CLIFlagTypePathFile|cli.CLIFlagMustExist|cli.CLIFlagRequired)

	_ = BuildTriggerCLI.AddCmd("version", "Prints version", getCLIVersionHandler(trig))

	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		os.Args = []string{"BuildTrigger", "version"}
	}
	return BuildTriggerCLI
}
