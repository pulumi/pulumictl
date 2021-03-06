package chocolatey

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
	app         string
)

type Payload struct {
	Ref string `json:"ref"`
}

const eventType = "choco-deploy"

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "choco-deploy [tag]",
		Short: "Create a Chocolatey Deployment",
		Long:  `Send a repository dispatch payload to the pulumi-chocolatey repo that triggers the deployment of a chocolatey package`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Grab all the configuration variables
			githubToken = viper.GetString("token")
			org = viper.GetString("org")
			ref = viper.GetString("ref")
			containerRepo := "pulumi/pulumi-chocolatey"
			ref := args[0]

			app = viper.GetString("app")

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

			var eventTriggerType string
			if app != "" {
				eventTriggerType = fmt.Sprintf("choco-deploy-%s", app)
			} else {
				eventTriggerType = eventType
			}

			// create the repository dispatch event
			_, _, err = client.Repositories.Dispatch(ctx,
				containerRepoArray[0],
				containerRepoArray[1],
				github.DispatchRequestOptions{
					EventType:     eventTriggerType,
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
	command.Flags().StringP("app", "a", "", "The name of the chocolatey application to deploy")

	viper.BindEnv("org", "GITHUB_ORG")
	viper.BindPFlag("org", command.Flags().Lookup("org"))
	viper.BindPFlag("app", command.Flags().Lookup("app"))

	return command
}
