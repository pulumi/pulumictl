package latest_plugin //nolint:revive // backwards compatibility

import (
	"fmt"
	"strings"

	gh "github.com/pulumi/pulumictl/pkg/github"

	"github.com/pulumi/pulumictl/pkg/pluginversion"
	"github.com/spf13/cobra"
	viperlib "github.com/spf13/viper"
)

var (
	version     string
	exists      bool
	tagsToCheck []string
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "latest-plugin [provider]",
		Short: "Get the latest available plugin",
		Long:  "Get the last update plugin version",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {

			org, _ := cmd.Flags().GetString("org")
			numOfTagsToCheck, _ := cmd.Flags().GetInt("num-tags")
			project := args[0]
			githubToken := viperlib.GetString("token")

			// create a github client and token
			ctx, client := gh.CreateGithubClient(githubToken)

			tags, _, err := client.Repositories.ListTags(ctx, org, project, nil)
			if err != nil {
				return fmt.Errorf("unable to list tags from GitHub: %w", err)
			}

			for t := 0; t < numOfTagsToCheck; t++ {
				tagsToCheck = append(tagsToCheck, *tags[t].Name)
			}

			result, err := pluginversion.CheckPluginTags(project, tagsToCheck)
			if err != nil {
				return fmt.Errorf("unable to get plugin tags: %w", err)
			}

			fmt.Println(strings.TrimPrefix(result, "v"))

			return nil
		},
	}

	command.Flags().StringP("org", "o", "pulumi", "the GitHub organization where the plugin lives.")
	command.Flags().IntP("num-tags", "n", 3, "The number of tags back from the latest to check for plugin versions.")

	return command
}
