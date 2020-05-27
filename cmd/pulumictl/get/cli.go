package get

import (
	"github.com/pulumi/pulumictl/cmd/pulumictl/get/latest_plugin"
	"github.com/spf13/cobra"

	"github.com/pulumi/pulumictl/cmd/pulumictl/get/version"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: "Get commands",
		Long:  "Commands that return information",
	}

	command.AddCommand(version.Command())
	command.AddCommand(latest_plugin.Command())

	return command
}
