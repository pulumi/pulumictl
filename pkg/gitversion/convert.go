package gitversion

import (
	"fmt"
	"regexp"
	"strings"
)

func GetLanguageOptionsFromVersion(version string) (*LanguageVersions, error) {
	// Strip leading "v" if present
	normalised := version
	if strings.HasPrefix(version, "v") {
		normalised = version[1:]
	}

	parts := strings.SplitN(normalised, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("version did not contain exactly 3 parts as expected: %q", version)
	}

	major := parts[0]
	minor := parts[1]
	patch := parts[2]

	pythonPatch, err := convertPatchToPython(patch)
	if err != nil {
		return nil, err
	}

	genericVersion := fmt.Sprintf("%s.%s.%s", major, minor, patch)
	pythonVersion := fmt.Sprintf("%s.%s.%s", major, minor, pythonPatch)
	jsVersion := fmt.Sprintf("v%s", genericVersion)
	dotnetVersion := genericVersion

	return &LanguageVersions{
		SemVer:     genericVersion,
		Python:     pythonVersion,
		JavaScript: jsVersion,
		DotNet:     dotnetVersion,
	}, nil
}

func convertPatchToPython(patch string) (string, error) {
	re := regexp.MustCompile(`^(\d+)(.*)$`)
	matches := re.FindStringSubmatch(patch)
	if len(matches) != 3 {
		return patch, nil
	}
	version := matches[1]
	pre := matches[2]
	pythonPre, err := getPythonPreVersion(pre)
	return version + pythonPre, err
}

func getPythonPreVersion(preVersion string) (string, error) {
	if preVersion == "" {
		return preVersion, nil
	}

	prefix, remaining := getPythonPrePrefix(preVersion)

	isDirty := strings.Contains(preVersion, "dirty")
	if isDirty {
		remaining = strings.Replace(remaining, "dirty", "", 1)
	}

	// Find the hash and remove to make sure we don't pick up the hash as a number
	hashRe := regexp.MustCompile(`\+[0-9a-f]{8}\b`)
	hashMatches := hashRe.FindAllString(remaining, 5)
	if len(hashMatches) > 0 {
		// Use the last match in case the build number is also exactly 8 digits long.
		shortHash := hashMatches[len(hashMatches)-1]
		remaining = strings.Replace(remaining, shortHash, "", 1)
	}
	// Find a number in the middle of non-words (- or .)
	numRe := regexp.MustCompile(`\W(\d+)(\W|$)`)
	nums := numRe.FindStringSubmatch(remaining)
	num := ""
	// Our match group is in the 2nd array entry.
	if len(nums) == 3 {
		num = nums[1]
	}

	// Python uses PEP440, but Pypi has some curiosities.
	pythonPreVersion := ""

	// PEP440 (https://peps.python.org/pep-0440/) says pre-release parts MUST have a number in them,
	// but we want to support tags like `v1.0.0-alpha`. If no number is present add `0` to keep PEP440
	// happy.
	var pythonPreSuffix string
	if num == "" {
		pythonPreSuffix = "0"
	} else {
		// Trim the initial "."
		pythonPreSuffix = num
	}

	if prefix != "" {
		pythonPreVersion = fmt.Sprintf("%s%s", prefix, pythonPreSuffix)
	}

	// Detect if the git worktree is dirty, and add `dirty` to the version if it is
	if isDirty {
		pythonPreVersion = fmt.Sprintf("%s+dirty", pythonPreVersion)
	}

	return pythonPreVersion, nil
}

func getPythonPrePrefix(preVersion string) (string, string) {
	if strings.HasPrefix(preVersion, "-dev") {
		return "d", preVersion[4:]
	}
	if strings.HasPrefix(preVersion, "-alpha") {
		return "a", preVersion[6:]
	}
	if strings.HasPrefix(preVersion, "-beta") {
		return "b", preVersion[5:]
	}
	if strings.HasPrefix(preVersion, "-rc") {
		return "rc", preVersion[3:]
	}
	return "", ""
}
