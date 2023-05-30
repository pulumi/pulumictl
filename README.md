# `pulumictl` - A Swiss Army Knife for Pulumi Development

`pulumictl` is a utility CLI to support publishing Pulumi packages (providers, policy packs, etc.) This tool provides utility functions to replace shell scripts. If you are looking to author programs in Pulumi or find the Pulumi CLI & engine, visit [Pulumi docs](https://www.pulumi.com/docs) or [github.com/pulumi/pulumi](https://github.com/pulumi/pulumi) respectively.

## Usage

```
$ pulumictl --help
A swiss army knife for Pulumi development

Usage:
  pulumictl [command]

Available Commands:
  completion      Generate the autocompletion script for the specified shell
  convert-version Convert versions
  copyright       Check copyright notices
  cover           Manipulate coverage profiles
  create          Create commands
  dispatch        Send a command dispatch event with a ref
  download-binary Downloads a version of a specific binary
  generate        Runs code generator over a schema
  get             Get commands
  help            Help about any command
  version         Get the current version
  winget-deploy   Create a WinGet Deployment

Flags:
  -D, --debug          enable debug logging
  -h, --help           help for pulumictl
  -t, --token string   a github token to use for making API calls to GitHub.

Use "pulumictl [command] --help" for more information about a command.
```

## Installation

Add the Pulumi homebrew tap and install:

```bash
brew tap pulumi/tap
brew install pulumictl
```

Or download the binary directly from Github releases and place it in your `$PATH`
