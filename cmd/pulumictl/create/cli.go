package create

import (
	"github.com/pulumi/pulumictl/cmd/pulumictl/create/containers"
	docsbuild "github.com/pulumi/pulumictl/cmd/pulumictl/create/docs-build"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "create",
		Short: "Create commands",
		Long:  "Commands that create resource or objects",
	}

	command.AddCommand(docsbuild.Command())
	command.AddCommand(containers.Command())

	return command
}
