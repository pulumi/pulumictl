package gitversion

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var testSignature = &object.Signature{
	Name:  "Test User",
	Email: "test@localhost",
}

func testRepoCreate() (*git.Repository, error) {

	//workDir := memfs.New()
	workDir, err := ioutil.TempDir(os.TempDir(), "test-repo")
	if err != nil {
		return nil, fmt.Errorf("Error creating tempdir: #{err}")
	}

	fs := osfs.New(workDir)
	st := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())

	repo, err := git.Init(st, fs)

	if err != nil {
		return nil, fmt.Errorf("git init: %w", err)
	}

	return repo, nil
}

func testRepoSingleCommitPastRelease() (*git.Repository, error) {
	repo, err := testRepoCreate()
	if err != nil {
		return nil, fmt.Errorf("repo create: %w", err)
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	workDir := workTree.Filesystem
	if err := writeFile(workDir, "hello-world", "Hello World"); err != nil {
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

	if err := writeFile(workDir, "hello-world2", "Hello World 2"); err != nil {
		return nil, fmt.Errorf("writeFile: %w", err)
	}
	if _, err := workTree.Add("hello-world2"); err != nil {
		return nil, fmt.Errorf("worktree-add: %w", err)
	}
	if _, err := workTree.Commit("Subsequent Commit!", &git.CommitOptions{Author: testSignature}); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return repo, nil
}

func testRepoSingleCommit() (*git.Repository, error) {
	repo, err := testRepoCreate()
	if err != nil {
		return nil, fmt.Errorf("repo create: %w", err)
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	workDir := workTree.Filesystem
	if err := writeFile(workDir, "hello-world", "Hello World"); err != nil {
		return nil, fmt.Errorf("writeFile: %w", err)
	}
	if _, err := workTree.Add("hello-world"); err != nil {
		return nil, fmt.Errorf("worktree-add: %w", err)
	}

	_, err = workTree.Commit("Initial Commit!", &git.CommitOptions{Author: testSignature})
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
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
	if _, err = writer.WriteString("Hello World"); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
