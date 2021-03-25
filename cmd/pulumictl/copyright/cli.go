package copyright

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {

	command := &cobra.Command{
		Use:   "copyright",
		Short: "Check copyright notices",
		Long:  "Checks that all source files have copyright notices in them",
		Args:  cobra.MaximumNArgs(0),

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.ParseFlags(args); err != nil {
				return err
			}

			repo, err := cmd.Flags().GetString("repo")
			if err != nil {
				return err
			}

			fixup, err := cmd.Flags().GetBool("fixup")
			if err != nil {
				return err
			}

			lines, err := cmd.Flags().GetInt("lines")
			if err != nil {
				return err
			}

			parallelism, err := cmd.Flags().GetInt("parallelism")
			if err != nil {
				return err
			}

			c := newChecker(repo, lines, parallelism)

			if fixup {
				return c.executeFixup()
			} else {
				return c.executeCheck()
			}

			return nil
		},
	}

	command.Flags().StringP("repo", "r", "", "path to repository, defaults to current working directory")
	command.Flags().Bool("fixup", false, "edit files to comply")
	command.Flags().Int("parallelism", 8, "parallelism level to use")
	command.Flags().Int("lines", 20, "max head lines to scan in each file")

	return command
}

type checker struct {
	repo                 string
	sourceFilePattern    *regexp.Regexp
	copyrightLinePattern *regexp.Regexp
	copyrightNotice      string
	headLineLimit        int
	parallelism          int
}

func newChecker(repo string, headLineLimit int, parallelism int) *checker {
	srcP := regexp.MustCompile(`[.](py|ts|cs|go)$`)
	copyP := regexp.MustCompile(`Copyright (20..-20..|20..), Pulumi Corporation`)
	copy := fmt.Sprintf("Copyright %d, Pulumi Corporation.  All rights reserved.",
		time.Now().Year())
	return &checker{
		copyrightNotice:      copy,
		repo:                 repo,
		sourceFilePattern:    srcP,
		copyrightLinePattern: copyP,
		headLineLimit:        headLineLimit,
		parallelism:          parallelism,
	}
}

func (c *checker) executeCheck() error {
	files, err := c.findUnlabelled()
	if err != nil {
		return err
	}

	if len(files) > 0 {
		fmt.Fprintf(os.Stderr, `Error: found %d source files missing a Copyright notice
Please add a notice or use pulumictl copyright --fixup.
Example notice:
    %s
Files missing a Copyright notice:
`, len(files), c.copyrightNotice)

		for _, f := range files {
			fmt.Printf("%s\n", f)
		}

		return fmt.Errorf("Found %d source files missing a Copyright notice", len(files))
	} else {
		return nil
	}
}

func (c *checker) executeFixup() error {
	files, err := c.findUnlabelled()
	if err != nil {
		return err
	}

	for _, f := range files {
		err := c.fixupFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *checker) fixupFile(filename string) error {
	commentPrefix := commentPrefixByFilename(filename)

	fileStat, err := os.Stat(filename)
	if err != nil {
		return err
	}

	oldBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileStat.Mode())
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)

	preamble := fmt.Sprintf("%s %s\n\n", commentPrefix, c.copyrightNotice)
	if _, err := w.WriteString(preamble); err != nil {
		return err
	}

	if _, err := w.Write(oldBytes); err != nil {
		return err
	}

	if err = w.Flush(); err != nil {
		return err
	}

	return nil
}

func commentPrefixByFilename(filename string) string {
	if strings.HasSuffix(filename, ".py") {
		return "#"
	} else {
		return "//"
	}
}

func (c *checker) findAllFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	if c.repo != "" {
		cmd.Dir = c.repo
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := splitLines(string(out))
	var files []string
	for _, line := range lines {
		files = append(files, filepath.Join(c.repo, line))
	}
	return files, nil
}

func (c *checker) findUnlabelled() ([]string, error) {
	fs, err := c.findAllFiles()
	if err != nil {
		return nil, err
	}

	var sourceFilesToCheck []interface{}
	for _, f := range fs {
		if c.isSourceFile(f) {
			sourceFilesToCheck = append(sourceFilesToCheck, f)
		}
	}

	check := func(input interface{}) interface{} {
		filename := input.(string)
		unlabelled, err := c.isUnlabelled(filename)
		if err != nil {
			return err
		}
		if unlabelled {
			return filename
		}
		return nil
	}

	checkResults := parMap(c.parallelism, sourceFilesToCheck, check)

	var out []string
	for _, result := range checkResults {
		if result != nil {
			switch v := result.(type) {
			case error:
				return nil, v
			case string:
				out = append(out, v)
			}
		}
	}

	return out, nil
}

func (c *checker) isSourceFile(filename string) bool {
	return c.sourceFilePattern.MatchString(filename)
}

func (c *checker) isUnlabelled(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return true, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var lineNo int = 0

	for scanner.Scan() {
		line := scanner.Text()

		if c.isCopyrightNotice(line) {
			return false, nil
		}

		lineNo++

		if lineNo >= c.headLineLimit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return true, err
	}

	return true, nil
}

func (c *checker) isCopyrightNotice(line string) bool {
	return c.copyrightLinePattern.MatchString(line)
}

func splitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

func parMap(numWorkers int, inputs []interface{}, mapper func(input interface{}) interface{}) []interface{} {
	jobs := make(chan interface{})
	results := make(chan interface{})

	for w := 0; w < numWorkers; w++ {
		go (func() {
			for input := range jobs {
				results <- mapper(input)
			}
		})()
	}

	go (func() {
		for _, input := range inputs {
			jobs <- input
		}
		close(jobs)
	})()

	var out []interface{}
	for range inputs {
		out = append(out, <-results)
	}

	return out
}
