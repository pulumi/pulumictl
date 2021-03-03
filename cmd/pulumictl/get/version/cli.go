package version

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pulumi/pulumictl/pkg/gitversion"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	language string
	versionPrefix string
	omitCommitHash bool
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "version",
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
			versionPrefix = viper.GetString("version-prefix")

			versions, err := gitversion.GetLanguageVersions(repo, plumbing.Revision(commitish),
				omitCommitHash, versionPrefix)
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
	command.Flags().StringVar(&versionPrefix, "version-prefix", "", "the version prefix (e.g. 3.0.0). Must be valid semver.")
	command.Flags().BoolVarP(&omitCommitHash, "omit-commit-hash", "o", false, "whether to include or omit the commit hash in the version")

	viper.SetDefault("language", "generic")
	viper.BindEnv("language", "PULUMI_LANGUAGE")
	viper.BindPFlag("language", command.Flags().Lookup("language"))

	viper.BindEnv("version-prefix", "VERSION_PREFIX")
	viper.BindPFlag("version-prefix", command.Flags().Lookup("version-prefix"))

	return command
}
