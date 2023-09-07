package version

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pulumi/pulumictl/pkg/gitversion"
	"github.com/pulumi/pulumictl/pkg/util"
	"github.com/spf13/cobra"
	viperlib "github.com/spf13/viper"
)

var (
	language       string
	versionPrefix  string
	omitCommitHash bool
	isPreRelease   bool
	tagPattern     string
)

func Command() *cobra.Command {
	viper := viperlib.New()
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

			workingDir, _ := cmd.Flags().GetString("repo")
			if workingDir == "" {
				var err error
				workingDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("error obtaining working directory: %w", err)
				}
			}

			language = viper.GetString("language")
			versionPrefix = viper.GetString("version-prefix")
			isPreRelease = viper.GetBool("is-prerelease")
			tagPattern = viper.GetString("tag-pattern")

			var tagFilter func(string) bool
			if tagPattern != "" {
				re, err := regexp.Compile(tagPattern)
				if err != nil {
					return fmt.Errorf("tag-pattern not a valid regexp: %w", err)
				}
				tagFilter = func(tag string) bool {
					return re.MatchString(tag)
				}
			}

			repo, err := git.PlainOpenWithOptions(workingDir, &git.PlainOpenOptions{
				DetectDotGit:          true,
				EnableDotGitCommonDir: true})
			if err != nil {
				return fmt.Errorf("error opening repository: %w", err)
			}

			versions, err := gitversion.GetLanguageVersionsWithOptions(gitversion.LanguageVersionsOptions{
				Repo:           repo,
				Commitish:      plumbing.Revision(commitish),
				OmitCommitHash: omitCommitHash,
				ReleasePrefix:  versionPrefix,
				IsPreRelease:   isPreRelease,
				TagFilter:      tagFilter,
			})

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
	command.Flags().StringVar(&versionPrefix,
		"version-prefix", "", "the version prefix (e.g. 3.0.0). Must be valid semver.")
	command.Flags().BoolVarP(&omitCommitHash,
		"omit-commit-hash", "o", false, "whether to include or omit the commit hash in the version")
	command.Flags().BoolVar(&isPreRelease, "is-prerelease", false, "whether this is a pre-release version")
	command.Flags().StringVar(&tagPattern, "tag-pattern", "", "regex pattern to filter tags with (e.g. ^sdk/)")

	viper.SetDefault("language", "generic")
	util.NoErr(viper.BindEnv("language", "PULUMI_LANGUAGE"))
	util.NoErr(viper.BindPFlag("language", command.Flags().Lookup("language")))

	util.NoErr(viper.BindEnv("version-prefix", "VERSION_PREFIX"))
	util.NoErr(viper.BindPFlag("version-prefix", command.Flags().Lookup("version-prefix")))

	util.NoErr(viper.BindEnv("is-prerelease", "IS_PRERELEASE"))
	util.NoErr(viper.BindPFlag("is-prerelease", command.Flags().Lookup("is-prerelease")))

	util.NoErr(viper.BindEnv("tag-pattern", "TAG_PATTERN"))
	util.NoErr(viper.BindPFlag("tag-pattern", command.Flags().Lookup("tag-pattern")))

	return command
}
