package gitversion

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

func VersionAtCommitForRepo(workingDirPath string, commitish string) (string, error) {
	// Open repository
	repo, err := git.PlainOpen(workingDirPath)
	if err != nil {
		return "", fmt.Errorf("error opening repository: %w", err)
	}

	// Resolve the `commitish` we were given into a reference
	headRef, err := repo.Reference("HEAD", true)
	if err != nil {
		return "", fmt.Errorf("error resolving commitish to reference: %w", err)
	}

	isExact, err := isExactTag(repo, headRef.Hash())
	if err != nil {
		return "", err
	}

	fmt.Println(isExact)

	return "", nil
}

func isExactTag(repo *git.Repository, hash plumbing.Hash) (bool, error) {
	tags, err := repo.Tags()
	if err != nil {
		return false, fmt.Errorf("error listing tags: %w", err)
	}

	isExact := false
	if err := tags.ForEach(func(ref *plumbing.Reference) error {
		if ref.Hash() == hash {
			isExact = true
			return storer.ErrStop
		}

		return nil
	}); err != nil {
		return false, fmt.Errorf("error iterating on tags: %w", err)
	}

	return isExact, nil
}
