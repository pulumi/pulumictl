package version

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pulumi/pulumictl/pkg/gitversion"
)

var (
	language string
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "get-version",
		Short: "Calculate versions",
		Long:  "Calculate a package version from repository tags and state",
		Args:  cobra.MaximumNArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.ParseFlags(args); err != nil {
				return err
			}

			commitish := "HEAD"
			if len(args) == 1 {
				commitish = args[0]
			}

			repo, _ := cmd.Flags().GetString("repo")
			if repo == "" {
				workingDir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("error obtaining working directory: %w", err)
				}

				repo = workingDir
			}

			language = viper.GetString("language")

			versions, err := gitversion.GetLanguageVersions(repo, plumbing.Revision(commitish))
			if err != nil {
				return fmt.Errorf("error calculating version: %w", err)
			}

			// FIXME: We could get the values here from the struct fields?
			switch strings.ToLower(language) {
			case "generic":
				fmt.Println(versions.SemVer)
			case "python":
				fmt.Println(versions.Python)
			case "javascript":
				fmt.Println(versions.JavaScript)
			case "dotnet":
				fmt.Println(versions.DotNet)
			default:
				return fmt.Errorf("invalid language %q ", language)
			}

			return nil
		},
	}

	command.Flags().StringP("repo", "r", "", "path to repository, defaults to current working directory")
	command.Flags().StringVarP(&language, "language", "p", "", "the platform for which the version should be output.")
	viper.SetDefault("language", "generic")
	viper.BindEnv("language", "PULUMI_LANGUAGE")
	viper.BindPFlag("language", command.Flags().Lookup("language"))

	return command
}
