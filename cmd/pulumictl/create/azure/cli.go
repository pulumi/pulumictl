package azure

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/blang/semver"
	"github.com/google/go-github/v32/github"
	"github.com/spf13/cobra"
	viperlib "github.com/spf13/viper"

	gh "github.com/pulumi/pulumictl/pkg/github"
	"github.com/pulumi/pulumictl/pkg/gitversion"
	"github.com/pulumi/pulumictl/pkg/util"
)

var (
	org         string
	repo        string
	tokenClient *http.Client
)

type Payload struct {
	Ref string `json:"ref"`
}

const eventType = "oss-sdk"

func Command() *cobra.Command {
	viper := viperlib.New()
	command := &cobra.Command{
		Use:   "oss-sdk [gitRef]",
		Short: "Publish the Azure Nextgen Provider SDK",
		Long:  `Send a repository dispatch payload to the pulumi-azure-nextgen repo that triggers the publishing of the SDK`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Grab all the configuration variables
			githubToken := viperlib.GetString("token")
			org = viper.GetString("org")

			containerRepo := "pulumi/pulumi-azure-nextgen"
			ref := args[0]

			// perform some string manipulation and validation
			containerRepoArray := strings.Split(containerRepo, "/")

			// if the string split doesn't return 2 values, it's probably not right
			if len(containerRepoArray) != 2 {
				return fmt.Errorf("unable to use container repo: format must be <org>/<repo> - value: %s", containerRepo)
			}

			_, err := semver.Parse(gitversion.StripModuleTagPrefixes(ref))

			if err != nil {
				return fmt.Errorf("must specify a valid semver ref - value: %s", ref)
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
				return fmt.Errorf("unable to create dispatch event: %w", err)
			}

			// output stuff
			fmt.Println("Submitting dispatch event to:", containerRepo)
			fmt.Println(string(payload))

			return nil

		},
	}

	command.Flags().StringP("org", "o", "pulumi", "the GitHub org that hosts the provider in the arg")

	util.NoErr(viper.BindEnv("org", "GITHUB_ORG"))
	util.NoErr(viper.BindPFlag("org", command.Flags().Lookup("org")))

	return command
}
