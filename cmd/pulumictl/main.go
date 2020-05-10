package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/pulumi/pulumictl/cmd/pulumictl/util"
	"github.com/pulumi/pulumictl/cmd/pulumictl/version"
	"github.com/pulumi/pulumictl/pkg/contract"
)

func configureCLI() *cobra.Command {
	rootCommand := &cobra.Command{
		Use:  "pulumictl",
		Long: "A swiss army knife for Pulumi development",
	}

	rootCommand.AddCommand(util.Command())
	rootCommand.AddCommand(version.Command())

	return rootCommand
}

func main() {
	rootCommand := configureCLI()

	if err := rootCommand.Execute(); err != nil {
		contract.IgnoreIoError(fmt.Fprintf(os.Stderr, "%s", err))
		os.Exit(1)
	}
}
