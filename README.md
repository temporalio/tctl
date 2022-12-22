[![build](https://github.com/temporalio/tctl/actions/workflows/test.yml/badge.svg)](https://github.com/temporalio/tctl/actions/workflows/test.yml)

The Temporal CLI is a command-line tool you can use to perform various tasks on a Temporal Server. It can perform namespace operations such as register, update, and describe as well as Workflow operations like start Workflow, show Workflow history, and signal Workflow.

Documentation for the Temporal command line interface is located at our [main site](https://docs.temporal.io/docs/system-tools/tctl).

## Quick Start

Run `make` from the project root. You should see an executable file called `tctl`. Try a few example commands to
get started:  
`./tctl` for help on top level commands and global options  
`./tctl namespace` for help on namespace operations  
`./tctl workflow` for help on workflow operations  
`./tctl task-queue` for help on tasklist operations  
(`./tctl help`, `./tctl help [namespace|workflow]` will also print help messages)

**Note:** Make sure you have a Temporal server running before using the CLI.

### Trying out the new `tctl next` with updated UX

**Note** Switching to `tctl next` is not recommended on production environments.

The package contains both `tctl v1` and the updated `tctl next`. Version `next` brings updated UX, new commands and flags semantics, new features ([see details](https://github.com/temporalio/proposals/tree/master/cli)). Please expect more of upcoming changes in `tctl next`

By default, executing tctl commands will execute commands from tctl v1. In order to switch to experimental `tctl next` run

```
tctl config set version next
```

This will create a configuration file (`~/.config/temporalio/tctl.yaml`) and set tctl to `next`.

To switch back to the stable v1, run

```
tctl config set version current
```

## Auto-completion

Running `tctl completion SHELL` will output the related completion SHELL code. See the following
sections for more details for each specific shell / OS and how to enable it.

### zsh auto-completion

Add the following to your `~/.zshrc` file:

```sh
source <(tctl completion zsh)
```

or from your terminal run:

```sh
echo 'source <(tctl completion zsh)' >> ~/.zshrc
```

Then run `source ~/.zshrc`.

### Bash auto-completion (linux)

Bash auto-completion relies on [bash-completion](https://github.com/scop/bash-completion#installation). Make sure
you follow the instruction [here](https://github.com/scop/bash-completion#installation) and install the software or
use a package manager to install it like `apt-get install bash-completion` or `yum install bash-completion`, etc. For example
on alpine linux:

-   apk update
-   apk add bash-completion
-   source /etc/profile.d/bash_completion.sh

Verify that bash-completion is installed by running `type _init_completion` add the following to your `.bashrc`
file to enable completion for tctl

```
echo 'source <(tctl completion bash)' >>~/.bashrc
source ~/.bashrc
```

### Bash auto-completion (macos)

For macos you can install it via brew `brew install bash-completion@2` and add the following line to
your `~/.bashrc`:

```sh
[[ -r "/usr/local/etc/profile.d/bash_completion.sh" ]] && . "/usr/local/etc/profile.d/bash_completion.sh"
```

Verify that bash-completion is installed by running `type _init_completion` and add the following to your `.bashrc`
file to enable completion for tctl

```
echo 'source <(tctl completion bash)' >> ~/.bashrc
source ~/.bashrc
```

## License

MIT License, please see [LICENSE](https://github.com/temporalio/tctl/blob/master/LICENSE) for details.
