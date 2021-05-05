package gosrcs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func findRepoRoot(dir string) (string, error) {
	root := dir
	for {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			return root, nil
		}
		if root == "/" {
			return "", errors.New("no .git directory found")
		}
		root = filepath.Dir(root)
	}
}

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "gosrcs",
		Short: "List source files for a directory containing Go packages",
		Long:  "List all of the source files in a repository that influence the Go packages under a given directory",
		Args:  cobra.MinimumNArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			repoRoot, err := findRepoRoot(wd)
			if err != nil {
				return fmt.Errorf("finding repository root: %w", err)
			}

			sources, err := findSources(repoRoot, wd, args...)
			if err != nil {
				return err
			}
			for i, s := range sources {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Print(s)
			}
			return nil
		},
	}

	return command
}
