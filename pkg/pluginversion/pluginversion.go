package pluginversion

import (
	"fmt"
	"net/http"
	"strings"
)

var (
	err error
)

func CheckPluginTags(project string, versions []string) (string, error) {

	for i, version := range versions {
		success, err := CheckPluginExists(project, version)
		if success {
			return version, nil
		}
		// If we're at the end of out loop, we should bail and throw an error
		if i == len(versions) {
			return "", err
		}
	}

	return "No plugins found", err
}

func CheckPluginExists(project string, version string) (bool, error) {

	resource := strings.Split(project, "-")
	if len(resource) != 2 {
		return false, err
	}
	pluginUrl := fmt.Sprintf("https://api.pulumi.com/releases/plugins/pulumi-resource-%s-%s-darwin-amd64.tar.gz", resource[1], version)

	// FIXME: would be nice if we could use `HEAD` here
	resp, err := http.Get(pluginUrl)

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, err

}
