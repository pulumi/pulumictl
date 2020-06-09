package main

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/pulumi/pulumictl/cmd/pulumictl/create"

	"github.com/spf13/cobra"

	"github.com/pulumi/pulumictl/cmd/pulumictl/get"
	"github.com/pulumi/pulumictl/cmd/pulumictl/version"
	"github.com/pulumi/pulumictl/pkg/contract"
)

var (
	githubToken string
)

func configureCLI() *cobra.Command {
	rootCommand := &cobra.Command{
		Use:  "pulumictl",
		Long: "A swiss army knife for Pulumi development",
	}

	rootCommand.AddCommand(get.Command())
	rootCommand.AddCommand(create.Command())
	rootCommand.AddCommand(version.Command())

	rootCommand.PersistentFlags().StringVarP(&githubToken, "token", "t", "", "a github token to use for making API calls to GitHub.")
	viper.BindEnv("token", "GITHUB_TOKEN")
	viper.BindPFlag("token", rootCommand.PersistentFlags().Lookup("token"))

	return rootCommand
}

func main() {
	rootCommand := configureCLI()

	if err := rootCommand.Execute(); err != nil {
		contract.IgnoreIoError(fmt.Fprintf(os.Stderr, "%s", err))
		os.Exit(1)
	}
}
