package docsbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v32/github"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
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

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "docs-build",
		Short: "Create a docs build",
		Long:  `Send a repository dispatch webhook to the docs repo`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Grab all the configuration variables
			githubToken = viper.GetString("token")
			org = viper.GetString("org")
			ref = viper.GetString("ref")
			docsRepo := viper.GetString("docs-repo")
			eventType := viper.GetString("event-type")

			// perform some string manipulation
			project := args[0]
			shortName := strings.Split(project, "-")[1]
			docsRepoArray := strings.Split(docsRepo, "/")

			// if the string split doesn't return 2 values, it's probably not right
			if len(docsRepoArray) != 2 {
				return fmt.Errorf("unable to use docs repo: format must be <org>/<repo> -currently: %s", docsRepo)
			}

			// create a github client and token
			tokenClient = nil
			ctx := context.Background()
			if githubToken != "" {
				ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
				tokenClient = oauth2.NewClient(ctx, ts)
			}
			client := github.NewClient(tokenClient)

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
				return fmt.Errorf("unable to create dispatch event: %w", err)
			}

			// output stuff
			fmt.Println("Submitting dispatch event to:", docsRepo)
			fmt.Println(string(payload))

			return nil

		},
	}

	command.Flags().StringP("org", "o", "pulumi", "the GitHub organization to send to the dispatch")
	command.Flags().StringP("ref", "r", "master", "the git reference to checkout from the provider")
	command.Flags().StringP("docs-repo", "d", "pulumi/docs", "the docs repository to send the payload")
	command.Flags().StringP("event-type", "e", "tfgen-provider", "the event type to send to the dispatch")
	viper.BindEnv("org", "GITHUB_ORG")
	viper.BindEnv("ref", "GITHUB_PROVIDER_REF")
	viper.BindEnv("docs-repo", "GITHUB_DOCS_REPO")
	viper.BindEnv("event-type", "GITHUB_EVENT_TYPE")
	viper.BindPFlag("ref", command.Flags().Lookup("ref"))
	viper.BindPFlag("org", command.Flags().Lookup("org"))
	viper.BindPFlag("docs-repo", command.Flags().Lookup("docs-repo"))
	viper.BindPFlag("event-type", command.Flags().Lookup("event-type"))

	return command
}
