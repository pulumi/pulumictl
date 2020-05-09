package util

import (
	"github.com/spf13/cobra"

	"github.com/pulumi/pulumictl/cmd/pulumictl/util/version"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "util",
		Short: "Utility commands",
		Long:  "Utility commands for use in external scripts",
	}

	command.AddCommand(version.Command())

	return command
}
