package gitversion

import (
	"fmt"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/blang/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// LanguageVersions contains a generic semantic version and Python-specific version number.
type LanguageVersions struct {
	SemVer     string
	Python     string
	JavaScript string
	DotNet     string
}

// GetLanguageVersions calculates the generic and Python-specific version numbers for the
// given `commitish` based on the most recent tag, the status of the work tree with respect
// to dirty files, and a timestamp.
func GetLanguageVersions(workingDirPath string, commitish plumbing.Revision, omitCommitHash bool,
	releasePrefix string, isPrerelease bool) (*LanguageVersions, error) {
	versionComponents, err := versionAtCommitForRepo(workingDirPath, commitish, releasePrefix, isPrerelease)
	if err != nil {
		return nil, fmt.Errorf("getting language versions: %w", err)
	}

	// For most platforms we use major.minor.patch-prerelease_tag.timestamp
	genericVersion := semver.Version{}
	genericVersion.Major = versionComponents.Semver.Major
	genericVersion.Minor = versionComponents.Semver.Minor
	genericVersion.Patch = versionComponents.Semver.Patch
	if len(versionComponents.Semver.Pre) == 1 {
		genericVersion.Pre = []semver.PRVersion{
			versionComponents.Semver.Pre[0],
			{VersionStr: strconv.FormatInt(versionComponents.Timestamp.UTC().Unix(), 10)},
		}
	}
	if len(versionComponents.Semver.Pre) > 1 {
		genericVersion.Pre = []semver.PRVersion{
			versionComponents.Semver.Pre[0],
			versionComponents.Semver.Pre[1],
		}
	}

	// Check the shorthash
	var shortHash string
	if omitCommitHash || isPrerelease {
		shortHash = ""
	} else {
		shortHash = fmt.Sprintf("+%s", versionComponents.ShortHash)
	}

	// a standard semantic version
	preVersion := ""
	// Python uses PEP440, but Pypi has some curiosities.
	pythonPreVersion := ""
	if len(genericVersion.Pre) != 0 {
		var preSuffix int64

		if !versionComponents.IsExact {
			preSuffix = versionComponents.Timestamp.UTC().Unix()
		} else {
			preSuffix = int64(versionComponents.Semver.Pre[1].VersionNum)
		}

		switch genericVersion.Pre[0].VersionStr {
		case "dev":
			pythonPreVersion = fmt.Sprintf("dev%d", preSuffix)
			preVersion = fmt.Sprintf("-dev.%d%s", preSuffix, shortHash)
		case "alpha":
			pythonPreVersion = fmt.Sprintf("a%d", preSuffix)
			preVersion = fmt.Sprintf("-alpha.%d%s", preSuffix, shortHash)
		case "beta":
			pythonPreVersion = fmt.Sprintf("b%d", preSuffix)
			preVersion = fmt.Sprintf("-beta.%d%s", preSuffix, shortHash)
		case "rc":
			pythonPreVersion = fmt.Sprintf("rc%d", preSuffix)
			preVersion = fmt.Sprintf("-rc.%d%s", preSuffix, shortHash)
		default:
			return nil, fmt.Errorf("prerelease string %q not valid semver string", genericVersion.Pre[0].VersionStr)
		}
	}

	// Detect if the git worktree is dirty, and add `dirty` to the version if it is
	if versionComponents.Dirty {
		if versionComponents.IsExact {
			preVersion = fmt.Sprintf("%s+dirty", preVersion)
			pythonPreVersion = fmt.Sprintf("%s+dirty", pythonPreVersion)
		} else {
			preVersion = fmt.Sprintf("%s.dirty", preVersion)
			pythonPreVersion = fmt.Sprintf("%s+dirty", pythonPreVersion)
		}
	}

	// a base version with the pre release info
	baseVersion := fmt.Sprintf("%d.%d.%d", genericVersion.Major, genericVersion.Minor, genericVersion.Patch)

	// calculate versions for all languages
	version := fmt.Sprintf("%s%s", baseVersion, preVersion)
	pythonVersion := fmt.Sprintf("%s%s", baseVersion, pythonPreVersion)
	jsVersion := fmt.Sprintf("v%s", version)
	dotnetVersion := version

	return &LanguageVersions{
		SemVer:     version,
		Python:     pythonVersion,
		JavaScript: jsVersion,
		DotNet:     dotnetVersion,
	}, nil
}

// versionComponents groups the various parameters which impact version calculation
type versionComponents struct {
	Semver    semver.Version
	Dirty     bool
	ShortHash string
	Timestamp time.Time
	IsExact   bool
}

// versionAtCommitForRepo determines the version components on which the language-specific variants
// are calculated from.
func versionAtCommitForRepo(workingDirPath string, commitish plumbing.Revision, releasePrefix string,
	isPrerelease bool) (*versionComponents, error) {
	// Open repository
	repo, err := git.PlainOpenWithOptions(workingDirPath, &git.PlainOpenOptions{EnableDotGitCommonDir: true})
	if err != nil {
		return nil, fmt.Errorf("error opening repository: %w", err)
	}

	revision, err := repo.ResolveRevision(commitish)
	if err != nil {
		return nil, fmt.Errorf("error resolving commitish to reference: %w", err)
	}

	commit, err := repo.CommitObject(*revision)
	if err != nil {
		return nil, fmt.Errorf("error getting commit for revision: %w", err)
	}

	baseVersion, isExact, err := determineBaseVersion(repo, revision, isPrerelease)
	if err != nil {
		return nil, fmt.Errorf("error determining base versionComponents: %w", err)
	}

	version, err := semver.Parse(baseVersion)
	if err != nil {
		return nil, fmt.Errorf("error parsing base versionComponents %q: %w", baseVersion, err)
	}
	if !isExact {
		if version.Major == 0 {
			version.Patch += 1
		} else {
			version.Minor += 1
			version.Patch = 0
		}
		version.Pre = []semver.PRVersion{
			{VersionStr: "alpha"},
		}
	}

	if releasePrefix != "" {
		newVersion, err := semver.Parse(releasePrefix)
		if err != nil {
			return nil, fmt.Errorf("error parsing releasePrefix override %q: %w", releasePrefix, err)
		}

		version.Major = newVersion.Major
		version.Minor = newVersion.Minor
		version.Patch = newVersion.Patch
	}

	isDirty, err := workTreeIsDirty(repo)
	if err != nil {
		return nil, err
	}

	return &versionComponents{
		Semver:    version,
		Dirty:     isDirty,
		ShortHash: revision.String()[:8],
		Timestamp: commit.Committer.When,
		IsExact:   isExact,
	}, nil
}

// determineBaseVersion returns an appropriate semantic versionComponents by the following process:
//
// - If `commitish` has a tag exactly associated with it, the versionComponents component of the tag
//   is returned.
// - If `commitish` does not have an exact tag associated, the versionComponents component of the most
//   recent exact tag is returned.
// - Otherwise, "v0.0.0" is returned
//
// The second return value is true if an exact tag match was made.
func determineBaseVersion(repo *git.Repository, revision *plumbing.Hash, isPrerelease bool) (string, bool, error) {
	// Resolve the `commitish` we were given into a reference
	commit, err := repo.CommitObject(*revision)
	if err != nil {
		return "", false, fmt.Errorf("error resolving reference: %w", err)
	}

	// First check whether we had a commit with an exact tag to start with
	isExact, exactMatch, err := isExactTag(repo, commit.Hash, isPrerelease)
	if err != nil {
		return "", false, fmt.Errorf("isExactTag: %w", err)
	}
	if isExact {
		return StripModuleTagPrefixes(exactMatch.Name().Short()), true, nil
	}

	// If not, find the most recent tag
	hasRecent, recentMatch, err := mostRecentTag(repo, commit.Hash, isPrerelease)
	if err != nil {
		return "", false, fmt.Errorf("mostRecentTag: %w", err)
	}
	if hasRecent {
		return StripModuleTagPrefixes(recentMatch.Name().Short()), false, nil
	}

	// Fallback if we don't have anything
	return "0.0.0", false, nil
}

// stripModuleTagPrefixes returns the last component of a path. This is used to
// resolve the tag format used in pulumi repos of "module/versionComponents" to a simple
// versionComponents.
func StripModuleTagPrefixes(tag string) string {
	_, versionComponent := path.Split(tag)
	return strings.TrimPrefix(versionComponent, "v")
}

// isExactTag returns true if the given hash has a tag associated with it. If
// true is returned, the second return value is a reference representing the tag.
func isExactTag(repo *git.Repository, hash plumbing.Hash, isPrerease bool) (bool, *plumbing.Reference, error) {
	tags, err := repo.Tags()
	if err != nil {
		return false, nil, fmt.Errorf("error listing tags: %w", err)
	}

	var exactTag *plumbing.Reference = nil
	if err := tags.ForEach(func(ref *plumbing.Reference) error {
		// if we are marking the release as a pre-release, then we want to take into account
		// the beta and rc versions
		// if we are in a normal release cycle then we want to skip these
		// we want to ignore the beta and rc tags - they are the next major version so we
		// don't want to use these in our calculations of the current release variant
		if !isPrerease && strings.Contains(ref.Name().String(), "beta") ||
			!isPrerease && strings.Contains(ref.Name().String(), "rc") {
			return nil
		}
		if ref.Hash() == hash {
			exactTag = ref
			return storer.ErrStop
		}

		return nil
	}); err != nil {
		return false, nil, fmt.Errorf("error iterating on tags: %w", err)
	}

	return exactTag != nil, exactTag, nil
}

// mostRecentTag returns a reference to the most recent tag in which the given commit reference is included.
// The first return value is true if there is a tag matching these criteria, and false if not. If the
// first return is true, the second value contains a reference to the appropriate tag.
func mostRecentTag(repo *git.Repository, ref plumbing.Hash, isPrerelease bool) (bool, *plumbing.Reference, error) {
	commit, err := repo.CommitObject(ref)
	if err != nil {
		return false, nil, fmt.Errorf("no commit for ref %q: %w", ref, err)
	}

	var mostRecentTag *plumbing.Reference
	walker := object.NewCommitPreorderIter(commit, nil, nil)
	err = walker.ForEach(func(commit *object.Commit) error {
		isExact, exact, err := isExactTag(repo, commit.Hash, isPrerelease)
		if err != nil {
			return err
		}

		if !isExact {
			return nil
		}

		mostRecentTag = exact
		return storer.ErrStop
	})

	return mostRecentTag != nil, mostRecentTag, err
}

// workTreeIsDirty returns whether the worktree associated with the given repository
// has local modifications.
func workTreeIsDirty(repo *git.Repository) (bool, error) {
	debug := viper.GetBool("debug")
	workTree, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("looking up worktree: %w", err)
	}

	if _, ok := repo.Storer.(*filesystem.Storage); !ok {
		status, err := workTree.Status()
		if err != nil {
			return false, fmt.Errorf("error getting git worktree status: %w", err)
		}
		if debug {
			fmt.Println(status)
		}
		return !status.IsClean(), err
	}

	// we need to refresh the index before we try and check diff-files
	// diff-files doesn't check the contents changing, it only checks
	// the stat changes so if the file was "touched" in anyway, then diff-files
	// *could* show it has changed but a git status then git diff-files wouldn't
	// because git status causes a reindex
	c := exec.Command("git", "update-index", "-q", "--refresh")
	c.Dir = workTree.Filesystem.Root()
	output, err := c.Output()
	if err != nil {
		if debug {
			fmt.Println(err)
			fmt.Println("Error updating git index - forcing isDirty")
		}
		return true, nil
	}
	if debug {
		fmt.Println(string(output))
	}

	// Fast-path if the underlying filesystem is on disk since Status is really slow
	// on larger repositories.
	c = exec.Command("git", "diff-files", "--name-status", "--ignore-space-at-eol")
	c.Dir = workTree.Filesystem.Root()
	output, err = c.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if debug {
				fmt.Println(string(ee.Stderr))
			}
			return true, nil
		}
		return false, err
	}
	if debug {
		fmt.Println(string(output))
	}
	return len(output) > 0, nil
}
