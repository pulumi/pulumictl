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

package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	dotnetgen "github.com/pulumi/pulumi/pkg/v3/codegen/dotnet"
	gogen "github.com/pulumi/pulumi/pkg/v3/codegen/go"
	nodejsgen "github.com/pulumi/pulumi/pkg/v3/codegen/nodejs"
	pythongen "github.com/pulumi/pulumi/pkg/v3/codegen/python"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/cmdutil"
)

const (
	tool   string = "pulumictl"
	stdout string = "stdout"
	nodejs string = "nodejs"
	python string = "python"
	golang string = "go"
	dotnet string = "dotnet"
)

func Command() *cobra.Command {
	var language string
	var inputFile string
	var outputFile string
	short := "Runs code generator over a schema"
	cmd := &cobra.Command{
		Use:   "generate",
		Short: short,
		Long: short + "\n" +
			"\nThis is a simple wrapper around a call to `GeneratePackage` in the appropriate language.\n" +
			"This is intended to make it easier to observe changes for code generation.\n" +
			"This is not intended to produce production code.\n",
		Run: cmdutil.RunFunc(func(cmd *cobra.Command, args []string) error {
			var schema *schema.Package
			var err error
			schema, err = readSchema(inputFile)
			if err != nil {
				return err
			}
			pkg, err := getPackage(language, schema)
			if err != nil {
				return err
			}
			if outputFile == stdout {
				displayPackage(pkg)
			} else {
				return writePackage(outputFile, pkg)
			}
			return nil
		}),
	}
	cmd.PersistentFlags().StringVarP(&language,
		"language", "l", "all", "Language to emit in")
	cmd.PersistentFlags().StringVarP(&inputFile,
		"input", "i", "schema.json", "The schema from which to generate code")
	cmd.PersistentFlags().StringVarP(&outputFile,
		"output", "o", stdout, "The output directory to emit generated code."+
			" 'stdout' indicates that generate code should be printed.")
	return cmd
}

func readSchema(schemaPath string) (*schema.Package, error) {
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}
	var pkgSpec schema.PackageSpec
	if err = json.Unmarshal(schemaBytes, &pkgSpec); err != nil {
		return nil, err
	}

	return schema.ImportSpec(pkgSpec, nil)
}

func getPackage(language string, pkg *schema.Package) (map[string][]byte, error) {
	resources := map[string][]byte{}
	var err error
	switch language {
	case nodejs:
		resources, err = nodejsgen.GeneratePackage(tool, pkg, map[string][]byte{}, nil, false)
	case python:
		resources, err = pythongen.GeneratePackage(tool, pkg, map[string][]byte{})
	case golang:
		resources, err = gogen.GeneratePackage(tool, pkg, nil)
	case dotnet:
		resources, err = dotnetgen.GeneratePackage(tool, pkg, map[string][]byte{}, nil)
	case "all":
		var tmp map[string][]byte
		tmp, err = nodejsgen.GeneratePackage(tool, pkg, map[string][]byte{}, nil, false)
		if err != nil {
			return nil, err
		}
		for fName, source := range tmp {
			resources[path.Join(nodejs, fName)] = source
		}
		tmp, err = pythongen.GeneratePackage(tool, pkg, map[string][]byte{})
		if err != nil {
			return nil, err
		}
		for fName, source := range tmp {
			resources[path.Join(python, fName)] = source
		}
		tmp, err = gogen.GeneratePackage(tool, pkg, nil)
		if err != nil {
			return nil, err
		}
		for fName, source := range tmp {
			resources[path.Join(golang, fName)] = source
		}
		tmp, err = dotnetgen.GeneratePackage(tool, pkg, map[string][]byte{}, nil)
		if err != nil {
			return nil, err
		}
		for fName, source := range tmp {
			resources[path.Join(dotnet, fName)] = source
		}
	}

	return resources, err
}

func displayPackage(pkg map[string][]byte) {
	fmt.Printf("Printing %d files\n", len(pkg))
	for name, source := range pkg {
		fmt.Printf(";;; File %s\n", name)
		fmt.Printf("%s\n\n;;; End %s\n", string(source), name)
	}
}

func writePackage(dir string, pkg map[string][]byte) error {
	err := os.Mkdir(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	for name, source := range pkg {
		// We do this because name might contain directories
		fullPath := path.Join(dir, name)
		err = os.MkdirAll(path.Dir(fullPath), 0755)
		if err != nil && !os.IsExist(err) {
			return err
		}
		err = os.WriteFile(fullPath, source, 0644) //nolint:gosec
		if err != nil {
			return err
		}
	}
	return nil
}
