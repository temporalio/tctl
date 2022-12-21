package cli_curr

import (
	"github.com/temporalio/tctl/completion"
	"github.com/urfave/cli"
)


func newCompletionCommand() cli.Command {
	return cli.Command{
		Name:    "completion",
		Usage:       "Output shell completion code for the specified shell (zsh, bash)",
		Subcommands: []cli.Command{
			{
				Name:      "zsh",
				Usage:       "zsh completion output",
				Action: func(c *cli.Context) error {
					return completion.Print(completion.ZSH)
				},
			},
			{
				Name:      "bash",
				Usage:       "bash completion output",
				Action: func(c *cli.Context) error {
					return completion.Print(completion.BASH)
				},
			},
		},
	}
}