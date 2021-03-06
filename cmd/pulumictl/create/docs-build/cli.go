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
	"github.com/spf13/viper"
)

var (
	githubToken string
	org         string
	repo        string
	ref         string
	tokenClient *http.Client
)

type Payload struct {
	Repo             string `json:"repo"`
	Org              string `json:"org"'`
	Project          string `json:"project"`
	ProjectShortname string `json:"project-shortname"`
	Ref              string `json:"ref"`
}

const eventType = "tfgen-provider"

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "docs-build [provider] [tag]",
		Short: "Create a docs build",
		Long:  `Send a repository dispatch payload to the docs repo`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Grab all the configuration variables
			githubToken = viper.GetString("token")
			org = viper.GetString("org")
			ref = viper.GetString("ref")
			docsRepo := viper.GetString("docs-repo")
			project := args[0]
			ref := args[1]

			// perform some string manipulation and validation
			// this manipulation will allow us to handle providers
			// such as pulumi-equinix-metal and pulumi-azure-nextgen
			parts := strings.Split(project, "-")
			parts = append(parts[:0], parts[1:]...)
			shortName := strings.Join(parts, "-")
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
				Repo:             fmt.Sprintf("%s/%s", org, project),
				Org:              org,
				Project:          project,
				ProjectShortname: shortName,
				Ref:              ref,
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
