package convertVersion //nolint:revive // backwards compatibility

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumictl/pkg/gitversion"
	"github.com/pulumi/pulumictl/pkg/util"
	"github.com/spf13/cobra"
	viperlib "github.com/spf13/viper"
)

var (
	language string
	version  string
)

func Command() *cobra.Command {
	viper := viperlib.New()

	command := &cobra.Command{
		Use:   "convert-version",
		Short: "Convert versions",
		Long:  "Convert a generic version into a language specific version",
		Args:  cobra.MaximumNArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.ParseFlags(args); err != nil {
				return err
			}

			language = viper.GetString("language")
			version = viper.GetString("version")

			versions, err := gitversion.GetLanguageOptionsFromVersion(version)

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

	command.Flags().StringVarP(&language,
		"language", "l", "",
		"the platform for which the version should be output.")
	command.Flags().StringVarP(&version,
		"version", "v", "",
		"the generic version to convert (e.g. 3.0.0). Must be valid semver.")

	util.NoErr(viper.BindEnv("language", "PULUMI_LANGUAGE"))
	util.NoErr(viper.BindPFlag("language", command.Flags().Lookup("language")))

	util.NoErr(viper.BindEnv("version", "VERSION"))
	util.NoErr(viper.BindPFlag("version", command.Flags().Lookup("version")))

	return command
}
