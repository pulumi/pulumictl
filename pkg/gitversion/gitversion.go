package gitversion

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

type LanguageVersions struct {
	SemVer string
	Python string
}

func GetLanguageVersions(workingDirPath string, commitish plumbing.Revision) (*LanguageVersions, error) {
	versionComponents, err := versionAtCommitForRepo(workingDirPath, commitish)
	if err != nil {
		return nil, err
	}

	// For most platforms we use major.minor.patch-prerelease_tag.timestamp
	genericVersion := semver.Version{}
	genericVersion.Major = versionComponents.Semver.Major
	genericVersion.Minor = versionComponents.Semver.Minor
	genericVersion.Patch = versionComponents.Semver.Patch
	genericVersion.Pre = versionComponents.npmPreVersion()

	// Python uses PEP440, but Pypi has some curiosities.
	pythonPreVersion := ""
	if len(genericVersion.Pre) != 0 {
		switch genericVersion.Pre[0].VersionStr {
		case "dev":
			pythonPreVersion = fmt.Sprintf("dev%d", versionComponents.Timestamp.UTC().Unix())
		case "alpha":
			pythonPreVersion = fmt.Sprintf("a%d", versionComponents.Timestamp.UTC().Unix())
		case "beta":
			pythonPreVersion = fmt.Sprintf("b%d", versionComponents.Timestamp.UTC().Unix())
		case "rc":
			pythonPreVersion = fmt.Sprintf("rc%d", versionComponents.Timestamp.UTC().Unix())
		default:
			return nil, fmt.Errorf("prerelease string %q incompatible with Pypi", genericVersion.Pre[0].VersionStr)
		}
	}
	pythonVersion := fmt.Sprintf("%d.%d.%d%s", genericVersion.Major, genericVersion.Minor, genericVersion.Patch, pythonPreVersion)

	return &LanguageVersions{
		SemVer: genericVersion.String(),
		Python: pythonVersion,
	}, nil
}

type versionComponents struct {
	Semver    semver.Version
	Dirty     bool
	ShortHash string
	Timestamp time.Time
}

func (v *versionComponents) npmPreVersion() []semver.PRVersion {
	if len(v.Semver.Pre) == 0 {
		return nil
	}

	return []semver.PRVersion{
		v.Semver.Pre[0],
		{
			VersionStr: strconv.FormatInt(v.Timestamp.UTC().Unix(), 10),
		},
	}
}

func versionAtCommitForRepo(workingDirPath string, commitish plumbing.Revision) (*versionComponents, error) {
	// Open repository
	repo, err := git.PlainOpen(workingDirPath)
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

	baseVersion, isExact, err := determineBaseVersion(repo, revision)
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

	isDirty, err := workTreeIsDirty(repo)
	if err != nil {
		return nil, err
	}

	return &versionComponents{
		Semver:    version,
		Dirty:     isDirty,
		ShortHash: revision.String()[:10],
		Timestamp: commit.Committer.When,
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
func determineBaseVersion(repo *git.Repository, revision *plumbing.Hash) (string, bool, error) {
	// Resolve the `commitish` we were given into a reference
	commit, err := repo.CommitObject(*revision)
	if err != nil {
		return "", false, fmt.Errorf("error resolving reference: %w", err)
	}

	// First check whether we had a commit with an exact tag to start with
	isExact, exactMatch, err := isExactTag(repo, commit.Hash)
	if err != nil {
		return "", false, fmt.Errorf("isExactTag: %w", err)
	}
	if isExact {
		return stripModuleTagPrefixes(exactMatch.Name().Short()), true, nil
	}

	// If not, find the most recent tag
	hasRecent, recentMatch, err := mostRecentTag(repo, commit.Hash)
	if err != nil {
		return "", false, fmt.Errorf("mostRecentTag: %w", err)
	}
	if hasRecent {
		return stripModuleTagPrefixes(recentMatch.Name().Short()), false, nil
	}

	// Fallback if we don't have anything
	return "0.0.0", false, nil
}

// stripModuleTagPrefixes returns the last component of a path. This is used to
// resolve the tag format used in pulumi repos of "module/versionComponents" to a simple
// versionComponents.
func stripModuleTagPrefixes(tag string) string {
	_, versionComponent := path.Split(tag)
	return strings.TrimPrefix(versionComponent, "v")
}

// isExactTag returns true if the given hash has a tag associated with it. If
// true is returned, the second return value is a reference representing the tag.
func isExactTag(repo *git.Repository, hash plumbing.Hash) (bool, *plumbing.Reference, error) {
	tags, err := repo.Tags()
	if err != nil {
		return false, nil, fmt.Errorf("error listing tags: %w", err)
	}

	var exactTag *plumbing.Reference = nil
	if err := tags.ForEach(func(ref *plumbing.Reference) error {
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
func mostRecentTag(repo *git.Repository, ref plumbing.Hash) (bool, *plumbing.Reference, error) {
	commit, err := repo.CommitObject(ref)
	if err != nil {
		return false, nil, fmt.Errorf("no commit for ref %q: %w", ref, err)
	}

	var mostRecentTag *plumbing.Reference
	walker := object.NewCommitPreorderIter(commit, nil, nil)
	err = walker.ForEach(func(commit *object.Commit) error {
		isExact, exact, err := isExactTag(repo, commit.Hash)
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
	worktree, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("error getting git worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("error getting git worktree status: %w", err)
	}

	return !status.IsClean(), nil
}
