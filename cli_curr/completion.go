package cli_curr

import (
	"github.com/temporalio/tctl/completion"
	"github.com/urfave/cli"
)


func newCompletionCommand() cli.Command {
	var subCommands = []cli.Command{}
	for _, shell := range completion.CommandConfig.Shells {
		cmd := cli.Command{
			Name: string(shell.Name),
			Usage: shell.Usage,
			Action: func(c *cli.Context) error {
				shell.Print()
				return nil
			},
		
		}
		subCommands = append(subCommands, cmd)
	}

	return cli.Command{
		Name: completion.CommandConfig.Name,
		Usage: completion.CommandConfig.Usage,
		Subcommands: subCommands,
	}
}