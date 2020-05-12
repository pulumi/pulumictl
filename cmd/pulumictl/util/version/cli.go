package version

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"

	"github.com/pulumi/pulumictl/pkg/gitversion"
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

			platform, _ := cmd.Flags().GetString("platform")

			versions, err := gitversion.GetLanguageVersions(repo, plumbing.Revision(commitish))
			if err != nil {
				return fmt.Errorf("error calculating version: %w", err)
			}

			switch strings.ToLower(platform) {
			case "generic":
				fmt.Println(versions.SemVer)
			case "python":
				fmt.Println(versions.Python)
			default:
				return fmt.Errorf("invalid platform %q - valid platforms are %q and %q", platform, "python", "generic")
			}

			return nil
		},
	}

	command.Flags().StringP("repo", "r", "", "path to repository, defaults to current working directory")
	command.Flags().StringP("platform", "p", "generic", "the platform for which the version should be output. `python` or `generic` are valid")

	return command
}
