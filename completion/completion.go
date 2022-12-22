package completion

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
)

// taken from https://github.com/urfave/cli/blob/master/autocomplete/zsh_autocomplete
var zsh = `
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

var bash = `
#! /bin/bash

_cli_bash_autocomplete() {
  if [[ "${COMP_WORDS[0]}" != "source" ]]; then
    local cur opts base
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    if [[ "$cur" == "-"* ]]; then
      opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} ${cur} --generate-bash-completion )
    else
      opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-bash-completion )
    fi
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
  fi
}

complete -o bashdefault -o default -o nospace -F _cli_bash_autocomplete tctl
unset tctl
`

type Shell string
const (
	BASH Shell = "bash"
	ZSH  Shell = "zsh"
)

func Print(shell Shell) error {
	if shell == BASH {
		fmt.Fprintln(os.Stdout, bash)
	} else if shell == ZSH {
		fmt.Fprintln(os.Stdout, zsh)
	} else {
    return errors.Errorf("unsupported shell type %q for completion", shell)
  }
	
	return nil
}
