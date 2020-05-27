package latest_plugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/pulumi/pulumictl/pkg/pluginversion"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
)

var (
	version     string
	exists      bool
	tagsToCheck []string
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

			client := github.NewClient(nil)
			tags, _, err := client.Repositories.ListTags(context.Background(), org, project, nil)
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

	command.Flags().StringP("org", "o", "pulumi", "the GitHub organization where the plugin lives")
	command.Flags().IntP("num-tags", "n", 3, "The number of tags back from the latest to check for plugin versions")

	return command
}
