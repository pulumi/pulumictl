package containers

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
	Ref string `json:"ref"`
}

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "containers [tag]",
		Short: "Create a containers build",
		Long:  `Send a repository dispatch payload to the pulumi repo that triggers the creation of sdk based images`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Grab all the configuration variables
			githubToken = viper.GetString("token")
			org = viper.GetString("org")
			ref = viper.GetString("ref")
			containerRepo := viper.GetString("container-repo")
			eventType := viper.GetString("event-type")
			ref := args[0]

			// perform some string manipulation and validation
			containerRepoArray := strings.Split(containerRepo, "/")

			// if the string split doesn't return 2 values, it's probably not right
			if len(containerRepoArray) != 2 {
				return fmt.Errorf("unable to use container repo: format must be <org>/<repo> - value: %s\n", containerRepo)
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
				containerRepoArray[0],
				containerRepoArray[1],
				github.DispatchRequestOptions{
					EventType:     eventType,
					ClientPayload: &payload,
				})

			if err != nil {
				return fmt.Errorf("unable to create dispatch event: %w\n", err)
			}

			// output stuff
			fmt.Println("Submitting dispatch event to:", containerRepo)
			fmt.Println(string(payload))

			return nil

		},
	}

	command.Flags().StringP("org", "o", "pulumi", "the GitHub org that hosts the provider in the arg")
	command.Flags().StringP("container-repo", "d", "pulumi/pulumi", "the pulumi repository to send in the payload")
	command.Flags().StringP("event-type", "e", "docker-build", "the event type to send to the dispatch")
	viper.BindEnv("org", "GITHUB_ORG")
	viper.BindEnv("container-repo", "GITHUB_PULUMI_REPO")
	viper.BindEnv("event-type", "GITHUB_EVENT_TYPE")
	viper.BindPFlag("org", command.Flags().Lookup("org"))
	viper.BindPFlag("container-repo", command.Flags().Lookup("container-repo"))
	viper.BindPFlag("event-type", command.Flags().Lookup("event-type"))

	return command
}
