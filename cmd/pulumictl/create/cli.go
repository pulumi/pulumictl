package create

import (
	"github.com/pulumi/pulumictl/cmd/pulumictl/create/azure"
	"github.com/pulumi/pulumictl/cmd/pulumictl/create/chocolatey"
	pulumiCliDocsbuild "github.com/pulumi/pulumictl/cmd/pulumictl/create/cli-docs-build"
	docsbuild "github.com/pulumi/pulumictl/cmd/pulumictl/create/docs-build"
	"github.com/pulumi/pulumictl/cmd/pulumictl/create/homebrew"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "create",
		Short: "Create commands",
		Long:  "Commands that create resource or objects",
	}

	command.AddCommand(docsbuild.Command())
	command.AddCommand(pulumiCliDocsbuild.Command())
	command.AddCommand(chocolatey.Command())
	command.AddCommand(homebrew.Command())
	command.AddCommand(azure.Command())

	return command
}
