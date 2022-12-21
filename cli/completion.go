package cli

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

// taken from https://github.com/urfave/cli/blob/master/autocomplete/zsh_autocomplete
var zsh_autocomplete = `
#compdef tctl

_cli_zsh_autocomplete() {

  local -a opts
  local cur
  cur=${words[-1]}
  if [[ "$cur" == "-"* ]]; then
    opts=("${(@f)$(_CLI_ZSH_AUTOCOMPLETE_HACK=1 ${words[@]:0:#words[@]-1} ${cur} --generate-bash-completion)}")
  else
    opts=("${(@f)$(_CLI_ZSH_AUTOCOMPLETE_HACK=1 ${words[@]:0:#words[@]-1} --generate-bash-completion)}")
  fi

  if [[ "${opts[1]}" != "" ]]; then
    _describe 'values' opts
  else
    _files
  fi

  return
}
compdef _cli_zsh_autocomplete tctl
`

func newCompletionCommand() *cli.Command {
	return &cli.Command{
		Name:        "completion",
		Usage:       "Output shell completion code for the specified shell",
		Subcommands: newCompletionSubCommands(),
	}
}

// NewCmdCompletion creates the available `completion` commands
func newCompletionSubCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:      "zsh",
			Usage:       "zsh completion output",
			Action: func(c *cli.Context) error {
				return runCompletion(c, zsh_autocomplete)
			},
		},
	}
}


func runCompletion(c *cli.Context, shell string) error {
	fmt.Fprintln(os.Stdout, shell)
	return nil
}