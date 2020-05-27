package latest_plugin

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"github.com/pulumi/pulumictl/pkg/pluginversion"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

var (
	version     string
	exists      bool
	tagsToCheck []string
	githubToken string
	tokenClient *http.Client
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "latest-plugin",
		Short: "Get the latest available plugin",
		Long:  "Get the last update plugin version",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {

			org, _ := cmd.Flags().GetString("org")
			numOfTagsToCheck, _ := cmd.Flags().GetInt("num-tags")
			project := args[0]
			githubToken = viper.GetString("token")
			tokenClient = nil

			ctx := context.Background()

			// Check if we have a github token, and set a client if we do
			if githubToken != "" {
				ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
				tokenClient = oauth2.NewClient(ctx, ts)
			}

			client := github.NewClient(tokenClient)
			tags, _, err := client.Repositories.ListTags(ctx, org, project, nil)
			if err != nil {
				return err
			}

			for t := 0; t < numOfTagsToCheck; t++ {
				tagsToCheck = append(tagsToCheck, *tags[t].Name)
			}

			result, err := pluginversion.CheckPluginTags(project, tagsToCheck)
			if err != nil {
				return err
			}

			fmt.Println(strings.TrimPrefix(result, "v"))

			return nil
		},
	}

	command.Flags().StringP("org", "o", "pulumi", "the GitHub organization where the plugin lives.")
	command.Flags().IntP("num-tags", "n", 3, "The number of tags back from the latest to check for plugin versions.")
	command.Flags().StringVarP(&githubToken, "token", "t", "", "a github token to use for making API calls.")

	viper.BindEnv("token", "GITHUB_TOKEN")
	viper.BindPFlag("token", command.Flags().Lookup("token"))

	return command
}
