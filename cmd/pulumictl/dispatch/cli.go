package dispatch

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/blang/semver"

	"github.com/google/go-github/v32/github"
	gh "github.com/pulumi/pulumictl/pkg/github"
	"github.com/pulumi/pulumictl/pkg/gitversion"
	"github.com/pulumi/pulumictl/pkg/util"
	"github.com/spf13/cobra"
	viperlib "github.com/spf13/viper"
)

type Payload struct {
	Ref string `json:"ref"`
}

func Command() *cobra.Command {
	viper := viperlib.New()

	command := &cobra.Command{
		Use:   "dispatch <ref> | <key>=<value>...",
		Short: "Send a command dispatch event with a ref",
		Long:  `Send a repository dispatch payload to a given repo`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Grab all the configuration variables
			githubToken := viperlib.GetString("token")
			repo := viper.GetString("repo")
			command := viper.GetString("command")

			payloadMap := make(map[string]string)
			if len(args) == 1 && !strings.Contains(args[0], "=") {
				payloadMap["ref"] = args[0]

				_, err := semver.Parse(gitversion.StripModuleTagPrefixes(payloadMap["ref"]))
				if err != nil {
					return fmt.Errorf("must specify a valid semver ref - value: %s", payloadMap["ref"])
				}
			} else {
				for _, arg := range args {
					parts := strings.SplitN(arg, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid argument: %s", arg)
					}
					payloadMap[parts[0]] = parts[1]
				}
			}

			// perform some string manipulation and validation
			repoArray := strings.Split(repo, "/")

			// if the string split doesn't return 2 values, it's probably not right
			if len(repoArray) != 2 {
				return fmt.Errorf("unable to use repo: format must be <org>/<repo> - value: %s", repo)
			}

			// create a github client and token
			ctx, client := gh.CreateGithubClient(githubToken)

			// create the JSON payload
			jsonPayload, err := json.Marshal(payloadMap)
			if err != nil {
				return err
			}
			payload := json.RawMessage(jsonPayload)

			// create the repository dispatch event
			_, _, err = client.Repositories.Dispatch(ctx,
				repoArray[0],
				repoArray[1],
				github.DispatchRequestOptions{
					EventType:     command,
					ClientPayload: &payload,
				})

			if err != nil {
				return fmt.Errorf("unable to create dispatch event: %w", err)
			}

			// output stuff
			fmt.Println("Submitting dispatch event to:", repo)
			fmt.Println(string(payload))

			return nil
		},
	}

	command.Flags().StringP("repo", "r", "", "the repository to send in the payload")
	command.Flags().StringP("command", "c", "", "The repository dispatch command to trigger")

	util.NoErr(viper.BindPFlag("repo", command.Flags().Lookup("repo")))
	util.NoErr(viper.BindPFlag("command", command.Flags().Lookup("command")))

	return command
}
