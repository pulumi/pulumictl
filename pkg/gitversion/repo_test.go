package gitversion

import (
	"bufio"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/require"
)

var testSignature = &object.Signature{
	Name:  "Test User",
	Email: "test@localhost",
}

func testRepoCreate() (*git.Repository, error) {
	workDir := memfs.New()

	repo, err := git.Init(memory.NewStorage(), workDir)

	if err != nil {
		return nil, fmt.Errorf("git init: %w", err)
	}

	return repo, nil
}

func testRepoFSCreate(baseDir string) (*git.Repository, error) {
	gitDir := osfs.New(filepath.Join(baseDir, ".git"))
	return git.Init(filesystem.NewStorageWithOptions(gitDir, nil, filesystem.Options{
		ExclusiveAccess: true,
	}), osfs.New(baseDir))
}

func testRepoSingleCommitPastRelease(repo *git.Repository) (*git.Repository, error) {
	workTree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	// alpha tag
	workDir := workTree.Filesystem
	if err := writeFile(workDir, "hello-world.alpha", "Hello World"); err != nil {
		return nil, fmt.Errorf("writeFile: %w", err)
	}
	if _, err := workTree.Add("hello-world.alpha"); err != nil {
		return nil, fmt.Errorf("worktree-add: %w", err)
	}
	alphaCommitHash, err := workTree.Commit("Initial Commit!", &git.CommitOptions{Author: testSignature})
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	if _, err := repo.CreateTag("v1.0.0-alpha.1", alphaCommitHash, nil); err != nil {
		return nil, fmt.Errorf("tag: %w", err)
	}

	// v1.0.0 tag
	if err := writeFile(workDir, "hello-world", "Hello World"); err != nil {
		return nil, fmt.Errorf("writeFile: %w", err)
	}
	if _, err := workTree.Add("hello-world"); err != nil {
		return nil, fmt.Errorf("worktree-add: %w", err)
	}
	tagCommitHash, err := workTree.Commit("Initial Commit!", &git.CommitOptions{Author: testSignature})
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	if _, err := repo.CreateTag("v1.0.0", tagCommitHash, nil); err != nil {
		return nil, fmt.Errorf("tag: %w", err)
	}

	// commit after tag
	if err := writeFile(workDir, "hello-world2", "Hello World 2"); err != nil {
		return nil, fmt.Errorf("writeFile: %w", err)
	}
	if _, err := workTree.Add("hello-world2"); err != nil {
		return nil, fmt.Errorf("worktree-add: %w", err)
	}
	postReleaseCommit, err := workTree.Commit("Subsequent Commit!", &git.CommitOptions{Author: testSignature})
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// make the commit after tag a new beta tag
	if _, err := repo.CreateTag("v2.0.0-beta.1", postReleaseCommit, nil); err != nil {
		return nil, fmt.Errorf("tag: %w", err)
	}

	return repo, nil
}

func addFile(t *testing.T, workTree *git.Worktree, name, content string) {
	err := writeFile(workTree.Filesystem, name, content)
	require.NoError(t, err, "writeFile")

	_, err = workTree.Add(name)
	require.NoError(t, err, "worktree-add")
}

func testRepoSingleCommit(repo *git.Repository) (plumbing.Hash, error) {
	workTree, err := repo.Worktree()
	if err != nil {
		return plumbing.Hash{}, fmt.Errorf("worktree: %w", err)
	}

	workDir := workTree.Filesystem
	if err := writeFile(workDir, "hello-world", "Hello World"); err != nil {
		return plumbing.Hash{}, fmt.Errorf("writeFile: %w", err)
	}
	if _, err := workTree.Add("hello-world"); err != nil {
		return plumbing.Hash{}, fmt.Errorf("worktree-add: %w", err)
	}

	commit, err := workTree.Commit("Initial Commit!", &git.CommitOptions{Author: testSignature})
	if err != nil {
		return plumbing.Hash{}, fmt.Errorf("commit: %w", err)
	}

	return commit, nil
}

func testRepoWithTags(repo *git.Repository, tagSequence []string) (*git.Repository, error) {
	workTree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	addFile := func(name, content string) error {
		if err := writeFile(workTree.Filesystem, name, content); err != nil {
			return fmt.Errorf("writeFile: %w", err)
		}
		if _, err := workTree.Add(name); err != nil {
			return fmt.Errorf("worktree-add: %w", err)
		}
		return nil
	}

	commitAndTag := func(tag string) error {
		tagCommitHash, err := workTree.Commit(tag, &git.CommitOptions{Author: testSignature})
		if err != nil {
			return fmt.Errorf("commit: %w", err)
		}
		if _, err := repo.CreateTag(tag, tagCommitHash, nil); err != nil {
			return fmt.Errorf("tag: %w", err)
		}
		return nil
	}

	for i, tag := range tagSequence {
		if err := addFile(fmt.Sprintf("%d.txt", i), tag); err != nil {
			return nil, err
		}
		if err := commitAndTag(tag); err != nil {
			return nil, err
		}
	}

	return repo, nil
}

func writeFile(workDir billy.Filesystem, fileName string, content string) error {
	file, err := workDir.Create(fileName)
	if err != nil {
		return fmt.Errorf("open in-memory file: %w", err)
	}
	defer func() {
		_ = file.Close
	}()

	writer := bufio.NewWriter(file)
	if _, err = writer.WriteString(content); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}

	return nil
}
