package docsbuild

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/blang/semver"

	"github.com/google/go-github/v32/github"
	gh "github.com/pulumi/pulumictl/pkg/github"
	"github.com/pulumi/pulumictl/pkg/gitversion"
	"github.com/spf13/cobra"
	viperlib "github.com/spf13/viper"
)

var (
	org         string
	repo        string
	tokenClient *http.Client
)

type Payload struct {
	Ref string `json:"ref"`
}

const eventType = "pulumi-cli"

func Command() *cobra.Command {
	viper := viperlib.New()
	command := &cobra.Command{
		Use:   "cli-docs-build [tag]",
		Short: "Create a docs build",
		Long:  `Send a repository dispatch payload to the docs repo`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Grab all the configuration variables
			githubToken := viperlib.GetString("token")
			org = viper.GetString("org")
			docsRepo := viper.GetString("docs-repo")
			ref := args[0]

			// perform some string manipulation and validation
			docsRepoArray := strings.Split(docsRepo, "/")

			// if the string split doesn't return 2 values, it's probably not right
			if len(docsRepoArray) != 2 {
				return fmt.Errorf("unable to use docs repo: format must be <org>/<repo> - value: %s\n", docsRepo)
			}

			_, err := semver.Parse(gitversion.StripModuleTagPrefixes(ref))

			if err != nil {
				return fmt.Errorf("must specify a valid semver ref - value: %s\n", ref)
			}

			// create a github client and token
			ctx, client := gh.CreateGithubClient(githubToken)

			// create the JSON payload
			jsonPayload, err := json.Marshal(Payload{
				Ref: ref,
			})

			if err != nil {
				return err
			}
			payload := json.RawMessage(jsonPayload)

			// create the repository dispatch event
			_, _, err = client.Repositories.Dispatch(ctx,
				docsRepoArray[0],
				docsRepoArray[1],
				github.DispatchRequestOptions{
					EventType:     eventType,
					ClientPayload: &payload,
				})

			if err != nil {
				return fmt.Errorf("unable to create dispatch event: %w\n", err)
			}

			// output stuff
			fmt.Println("Submitting dispatch event to:", docsRepo)
			fmt.Println(string(payload))

			return nil

		},
	}

	command.Flags().StringP("org", "o", "pulumi", "the GitHub org that hosts the provider in the arg")
	command.Flags().StringP("docs-repo", "d", "pulumi/docs", "the docs repository to send in the payload")

	viper.BindEnv("org", "GITHUB_ORG")
	viper.BindEnv("docs-repo", "GITHUB_DOCS_REPO")
	viper.BindPFlag("org", command.Flags().Lookup("org"))
	viper.BindPFlag("docs-repo", command.Flags().Lookup("docs-repo"))

	return command
}
