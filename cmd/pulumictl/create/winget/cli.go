package winget

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v32/github"
	gh "github.com/pulumi/pulumictl/pkg/github"
	"github.com/spf13/cobra"
	viperlib "github.com/spf13/viper"
)

var (
	tokenClient *http.Client
)

const eventType = "winget-deploy"

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "winget-deploy",
		Short: "Create a WinGet Deployment",
		Long: "Send a repository dispatch payload to the pulumi-winget " +
			"repo that triggers the deployment of a winget package",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {

			// Grab all the configuration variables
			githubToken := viperlib.GetString("token")
			containerRepo := "pulumi/pulumi-winget"
			// perform some string manipulation and validation
			containerRepoArray := strings.Split(containerRepo, "/")

			// if the string split doesn't return 2 values, it's probably not right
			if len(containerRepoArray) != 2 {
				return fmt.Errorf("unable to use container repo:"+
					" format must be <org>/<repo> - value:"+
					" %s\n", containerRepo)
			}

			// create a github client and token
			ctx, client := gh.CreateGithubClient(githubToken)

			// create an JSON payload
			// pulumi-winget doesn't require a particular payload
			emptyJSONPayload := "{}"
			payload := json.RawMessage(emptyJSONPayload)

			// create the repository dispatch event
			_, _, err := client.Repositories.Dispatch(ctx,
				containerRepoArray[0],
				containerRepoArray[1],
				github.DispatchRequestOptions{
					EventType:     eventType,
					ClientPayload: &payload,
				})

			if err != nil {
				return fmt.Errorf("unable to create dispatch event: %w", err)
			}

			fmt.Println("Submitting dispatch event to:", containerRepo)
			return nil
		},
	}
	return command
}
