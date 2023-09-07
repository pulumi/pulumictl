package main

import (
	"fmt"
	"os"

	download_binary "github.com/pulumi/pulumictl/cmd/pulumictl/download-binary"

	"github.com/spf13/cobra"
	viperlib "github.com/spf13/viper"

	convert_version "github.com/pulumi/pulumictl/cmd/pulumictl/convert-version"
	"github.com/pulumi/pulumictl/cmd/pulumictl/copyright"
	"github.com/pulumi/pulumictl/cmd/pulumictl/cover"
	"github.com/pulumi/pulumictl/cmd/pulumictl/create"
	"github.com/pulumi/pulumictl/cmd/pulumictl/create/winget"
	"github.com/pulumi/pulumictl/cmd/pulumictl/dispatch"
	"github.com/pulumi/pulumictl/cmd/pulumictl/generate"
	"github.com/pulumi/pulumictl/cmd/pulumictl/get"
	"github.com/pulumi/pulumictl/cmd/pulumictl/version"
	"github.com/pulumi/pulumictl/pkg/contract"
	"github.com/pulumi/pulumictl/pkg/util"
)

var (
	githubToken string
	debug       bool
)

func configureCLI() *cobra.Command {
	// Using the shared global Viper instance for the top-level command. Sub-commands should use viperlib.New().
	viper := viperlib.GetViper()

	rootCommand := &cobra.Command{
		Use:  "pulumictl",
		Long: "A swiss army knife for Pulumi development",
	}

	rootCommand.AddCommand(get.Command())
	rootCommand.AddCommand(create.Command())
	rootCommand.AddCommand(version.Command())
	rootCommand.AddCommand(dispatch.Command())
	rootCommand.AddCommand(copyright.Command())
	rootCommand.AddCommand(generate.Command())
	rootCommand.AddCommand(cover.Command())
	rootCommand.AddCommand(winget.Command())
	rootCommand.AddCommand(download_binary.Command())
	rootCommand.AddCommand(convert_version.Command())

	rootCommand.PersistentFlags().StringVarP(&githubToken,
		"token", "t", "", "a github token to use for making API calls to GitHub.")
	rootCommand.PersistentFlags().BoolVarP(&debug, "debug", "D", false, "enable debug logging")
	util.NoErr(viper.BindEnv("debug", "PULUMICTL_DEBUG"))
	util.NoErr(viper.BindEnv("token", "GITHUB_TOKEN"))
	util.NoErr(viper.BindPFlag("debug", rootCommand.PersistentFlags().Lookup("debug")))

	return rootCommand
}

func main() {
	rootCommand := configureCLI()

	if err := rootCommand.Execute(); err != nil {
		contract.IgnoreIoError(fmt.Fprintf(os.Stderr, "%s", err))
		os.Exit(1)
	}
}
