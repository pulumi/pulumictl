package gitversion

import (
	"bufio"
	"fmt"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/require"
)

func TestIsExactTag(t *testing.T) {
	repo, err := makeTagTestRepo()
	require.NoError(t, err)
	require.NotNil(t, repo)

	headRef, err := repo.Head()
	require.NoError(t, err)
	require.NotEmpty(t, headRef)

	isExact, err := isExactTag(repo, headRef.Hash())
	require.NoError(t, err)
	require.False(t, isExact)

	exactRef, err := repo.Tag("v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, exactRef)

	isExact, err = isExactTag(repo, exactRef.Hash())
	require.NoError(t, err)
	require.True(t, isExact)
}

func makeTagTestRepo() (*git.Repository, error) {
	workDir := memfs.New()

	testSignature := &object.Signature{
		Name:  "Test User",
		Email: "test@localhost",
	}

	repo, err := git.Init(memory.NewStorage(), workDir)
	if err != nil {
		return nil, fmt.Errorf("git init: %w", err)
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	writeFile := func(fileName string, content string) error {
		file, err := workDir.Create(fileName)
		if err != nil {
			return fmt.Errorf("open in-memory file: %w", err)
		}
		defer func() {
			_ = file.Close
		}()

		writer := bufio.NewWriter(file)
		if _, err = writer.WriteString("Hello World"); err != nil {
			return fmt.Errorf("write: %w", err)
		}

		return nil
	}

	if err := writeFile("hello-world", "Hello World"); err != nil {
		return nil, fmt.Errorf("writeFile: %w", err)
	}
	if _, err := workTree.Add("hello-world"); err != nil {
		return nil, fmt.Errorf("worktree-add: %w", err)
	}

	commitHash, err := workTree.Commit("Initial Commit!", &git.CommitOptions{Author: testSignature})
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	if _, err := repo.CreateTag("v1.0.0", commitHash, nil); err != nil {
		return nil, fmt.Errorf("tag: %w", err)
	}

	if err := writeFile("hello-world2", "Hello World 2"); err != nil {
		return nil, fmt.Errorf("writeFile: %w", err)
	}
	if _, err := workTree.Add("hello-world2"); err != nil {
		return nil, fmt.Errorf("worktree-add: %w", err)
	}
	if _, err := workTree.Commit("Initial Commit!", &git.CommitOptions{Author: testSignature}); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return repo, nil
}
