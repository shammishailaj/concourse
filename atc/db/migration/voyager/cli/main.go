package main

import (
	"os"

	command "github.com/concourse/concourse/atc/db/migration/voyager/cli/command"
	flags "github.com/jessevdk/go-flags"
)

func main() {
	cmd := &command.MigrationCommand{}

	parser := flags.NewParser(cmd, flags.Default)
	parser.Command.Find("generate")
	_, err := parser.Parse()
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}
