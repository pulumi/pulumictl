package gitversion

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

func VersionAtCommitForRepo(workingDirPath string, commitish plumbing.Revision) (string, error) {
	// Open repository
	repo, err := git.PlainOpen(workingDirPath)
	if err != nil {
		return "", fmt.Errorf("error opening repository: %w", err)
	}

	// Resolve the `commitish` we were given into a reference
	revision, err := repo.ResolveRevision(commitish)
	if err != nil {
		return "", fmt.Errorf("error resolving commitish to reference: %w", err)
	}
	commit, err := repo.CommitObject(*revision)
	if err != nil {
		return "", fmt.Errorf("error resolving reference: %w", err)
	}

	// First check whether we had a commit with an exact tag to start with
	isExact, exactMatch, err := isExactTag(repo, commit.Hash)
	if err != nil {
		return "", fmt.Errorf("isExactTag: %w", err)
	}
	if isExact {
		fmt.Println(exactMatch.String())
	}

	return "", nil
}

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
