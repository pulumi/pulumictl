// Copyright 2016-2021, Pulumi Corporation.
//
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package download_binary //nolint:revive // backwards compatibility

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pulumi/pulumi/sdk/v3/go/common/util/cmdutil"
	"github.com/pulumi/pulumictl/pkg/util"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	var repoSlug string
	var name string
	var version string
	var host string
	short := "Downloads a version of a specific binary"
	cmd := &cobra.Command{
		Use:     "download-binary",
		Short:   short,
		Example: "pulumictl download-binary -n my-binary -r pulumi/pulumi-mybinary -v v0.0.1",
		Long: short + "\n" +
			"\nThis will download a version of binary to a specific location.",
		Run: cmdutil.RunFunc(func(cmd *cobra.Command, args []string) error {
			filename := fmt.Sprintf("%s-%s-%s-%s.tar.gz", name, version, runtime.GOOS, runtime.GOARCH)
			downloadURL := fmt.Sprintf("%s/%s/releases/download/%s/%s", host, repoSlug, version, filename)

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("unable to detect current working directory: %w", err)
			}
			srcFile := fmt.Sprintf("%s/bin/%s", cwd, filename)

			err = downloadBinary(downloadURL, srcFile)
			if err != nil {
				return err
			}

			err = decompressBinary(srcFile, cwd)
			if err != nil {
				return err
			}

			return os.Remove(srcFile)
		}),
	}
	cmd.PersistentFlags().StringVarP(&repoSlug,
		"repo-slug", "r", "", "the repo slug for the repository where the binary lives")
	cmd.PersistentFlags().StringVarP(&name,
		"name", "n", "", "the name of name to download e.g. pulumi-language-java")
	cmd.PersistentFlags().StringVarP(&version,
		"version", "v", "", "the version of the binary e.g. v0.4.0")
	cmd.PersistentFlags().StringVar(&host,
		"host", "https://github.com", "The host of the repo to download from. "+
			"Defaults to https://github.com")

	util.Ignore(cmd.MarkFlagRequired("repo-slug"))
	util.Ignore(cmd.MarkFlagRequired("name"))
	util.Ignore(cmd.MarkFlagRequired("version"))

	return cmd
}

func downloadBinary(url, destFile string) error {
	response, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("unable to retrieve URL: %q, received non 200 response code: %v", url, response.StatusCode)
	}

	out, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer out.Close()

	// Writer the body to file
	_, err = io.Copy(out, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func decompressBinary(srcFile, cwd string) error {
	reader, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer archive.Close()

	tarReader := tar.NewReader(archive)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		path := filepath.Join(cwd, "bin", header.Name) //nolint:gosec
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		_, err = io.Copy(file, tarReader) //nolint:gosec
		if err != nil {
			file.Close()
			return err
		}
		file.Close()
	}

	return nil
}
