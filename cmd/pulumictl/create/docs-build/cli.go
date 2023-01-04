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
	category    string
	displayName string
	component   bool
	schemaPath  string
	publisher   string
	tokenClient *http.Client
)

type Payload struct {
	Repo             string `json:"repo"`
	Org              string `json:"org"'`
	Project          string `json:"project"`
	ProjectShortname string `json:"project-shortname"`
	Ref              string `json:"ref"`
	Category         string `json:"category"`
	DisplayName      string `json:"display-name"`
	Component        bool   `json:"is-component"`
	SchemaPath       string `json:"schema-path"`
	Publisher        string `json:"publisher"`
}

const eventType = "resource-provider"

func Command() *cobra.Command {
	viper := viperlib.New()
	command := &cobra.Command{
		Use:   "docs-build [provider] [tag]",
		Short: "Create a docs build",
		Long:  `Send a repository dispatch payload to the docs repo`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Grab all the configuration variables
			githubToken := viperlib.GetString("token")
			org = viper.GetString("org")
			docsRepo := "pulumi/registry"
			project := args[0]
			ref := args[1]
			category = viper.GetString("category")
			displayName = viper.GetString("display-name")
			component = viper.GetBool("is-component")
			schemaPath = viper.GetString("schema-path")
			publisher = viper.GetString("publisher")

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
				Category:         category,
				DisplayName:      displayName,
				Component:        component,
				SchemaPath:       schemaPath,
				Publisher:        publisher,
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
	command.Flags().StringP("category", "c", "", "the category of the provider/component")
	command.Flags().String("display-name", "", "the display name of the provider/component")
	command.Flags().Bool("is-component", false, "is this a component?")
	command.Flags().String("schema-path", "", "the path (relative to repo root) to the schema.yaml/json file")
	command.Flags().String("publisher", "", "the name of the provider/component publisher")

	viper.BindEnv("org", "GITHUB_ORG")
	viper.BindEnv("category", "PROVIDER_CATEGORY")
	viper.BindEnv("display-name", "PROVIDER_DISPLAY_NAME")
	viper.BindEnv("is-component", "PROVIDER_IS_COMPONENT")
	viper.BindEnv("schema-path", "PROVIDER_SCHEMA_PATH")
	viper.BindEnv("publisher", "PROVIDER_PUBLISHER_NAME")
	viper.BindPFlag("org", command.Flags().Lookup("org"))
	viper.BindPFlag("category", command.Flags().Lookup("category"))
	viper.BindPFlag("display-name", command.Flags().Lookup("display-name"))
	viper.BindPFlag("is-component", command.Flags().Lookup("is-component"))
	viper.BindPFlag("schema-path", command.Flags().Lookup("schema-path"))
	viper.BindPFlag("publisher", command.Flags().Lookup("publisher"))

	return command
}
