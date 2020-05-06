package main

import (
	"github.com/davecgh/go-spew/spew"

	"github.com/pulumi/pulumictl/pkg/gitversion"
)

func main() {
	// version, err := gitversion.versionAtCommitForRepo("/Users/James/Code/pulumi/pulumi-aws", "a5b8388061a1cdad34c399e3699f05621b0e0464")
	version, err := gitversion.GetLanguageVersions("/Users/James/Code/pulumi/pulumi-aws", "HEAD")
	if err != nil {
		panic(err)
	}

	spew.Config.DisableMethods = true
	spew.Dump(version)
}
