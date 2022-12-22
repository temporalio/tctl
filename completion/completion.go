package completion

import (
	"fmt"
	"os"
)

// taken from https://github.com/urfave/cli/blob/master/autocomplete/zsh_autocomplete
var zsh_script = `
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

var bash_script = `
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
`

var (
	schellScriptMap = map[string]func() {
		"bash": func() { fmt.Fprintln(os.Stdout, bash_script) },
		"zsh":  func() { fmt.Fprintln(os.Stdout, zsh_script) },
	}
)

type Shell string

const (
	BASH Shell = "bash"
	ZSH  Shell = "zsh"
)
type shellConfig struct {
  Name Shell
  Usage string
}

type commandConfig struct {
  Name string
  Usage string
  Shells []shellConfig
}

func (s shellConfig) Print() {
  schellScriptMap[string(s.Name)]()
}

var CommandConfig commandConfig = commandConfig{
  Name: "completion", 
  Usage: "Output shell completion code for the specified shell (zsh, bash)",
  Shells: []shellConfig {
    {
      Name: ZSH,
      Usage: "zsh completion output",
    },
    {
      Name: BASH,
      Usage: "bash completion output",
    },
  },
}
