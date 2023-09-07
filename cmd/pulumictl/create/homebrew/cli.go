package homebrew

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/blang/semver"
	"github.com/google/go-github/v32/github"
	gh "github.com/pulumi/pulumictl/pkg/github"
	"github.com/pulumi/pulumictl/pkg/gitversion"
	"github.com/pulumi/pulumictl/pkg/util"
	"github.com/spf13/cobra"
	viperlib "github.com/spf13/viper"
)

var (
	org         string
	repo        string
	commitsha   string
	tokenClient *http.Client
)

type Payload struct {
	Ref       string `json:"ref"`
	CommitSha string `json:"commitSha"`
}

const eventType = "homebrew-bump"

func Command() *cobra.Command {
	viper := viperlib.New()

	command := &cobra.Command{
		Use:   "homebrew-bump [tag]",
		Short: "Create a Homebrew deployment",
		Long: "Send a repository dispatch payload to the pulumi repo that triggers" +
			" the deployment of a homebrew formulae bump",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Grab all the configuration variables
			githubToken := viperlib.GetString("token")
			org = viper.GetString("org")
			homebrewRepo := "pulumi/pulumi"
			ref := args[0]
			commitsha := args[1]

			// perform some string manipulation and validation
			containerRepoArray := strings.Split(homebrewRepo, "/")

			// if the string split doesn't return 2 values, it's probably not right
			if len(containerRepoArray) != 2 {
				return fmt.Errorf("unable to use container repo: format must be <org>/<repo> - value: %s", homebrewRepo)
			}

			_, err := semver.Parse(gitversion.StripModuleTagPrefixes(ref))

			if err != nil {
				return fmt.Errorf("must specify a valid semver ref - value: %s", ref)
			}

			// create a github client and token
			ctx, client := gh.CreateGithubClient(githubToken)

			// create the JSON payload
			jsonPayload, err := json.Marshal(Payload{
				Ref:       ref,
				CommitSha: commitsha,
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
			fmt.Println("Submitting dispatch event to:", homebrewRepo)
			fmt.Println(string(payload))

			return nil

		},
	}

	command.Flags().StringP("org", "o", "pulumi", "the GitHub org that hosts the provider in the arg")

	util.NoErr(viper.BindEnv("org", "GITHUB_ORG"))
	util.NoErr(viper.BindPFlag("org", command.Flags().Lookup("org")))

	return command
}
