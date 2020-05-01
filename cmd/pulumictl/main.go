package main

import (
	"fmt"

	"github.com/pulumi/pulumictl/pkg/gitversion"
)

func main() {
	version, err := gitversion.VersionAtCommitForRepo("/Users/James/Code/pulumi/pulumi-aws", "")
	if err != nil {
		panic(err)
	}

	fmt.Println(version)
}
