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
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/require"
)

var testSignature = &object.Signature{
	Name:  "Test User",
	Email: "test@localhost",
}

func testRepoCreate(t *testing.T) *git.Repository {
	workDir := memfs.New()

	repo, err := git.Init(memory.NewStorage(), workDir)
	require.NoError(t, err)

	return repo
}

func testRepo(
	setup func(*testing.T, *git.Repository),
	test func(*testing.T, *git.Repository),
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("local", func(t *testing.T) {
			repo := testRepoCreate(t)
			setup(t, repo)
			test(t, repo)
		})

		t.Run("remote", func(t *testing.T) {
			remoteDir := t.TempDir()
			remote := testRepoFSCreate(t, remoteDir)
			url := "file://" + remoteDir + "/.git/"
			t.Logf("\nTempDir is %q\n=>\nURL is %q", remoteDir, url)

			setup(t, remote)

			headRef, err := remote.Head()
			require.NoError(t, err)

			t.Logf("Remote HEAD: %s", headRef.Hash())

			localDir := memfs.New()
			local, err := git.Clone(memory.NewStorage(), localDir, &git.CloneOptions{
				URL:        url,
				RemoteName: "origin",
				Depth:      1,
				Tags:       git.TagFollowing,
			})
			require.NoError(t, err)

			test(t, local)
		})
	}
}

func testRepoFSCreate(t *testing.T, baseDir string) *git.Repository {
	gitDir := osfs.New(filepath.Join(baseDir, ".git"))
	repo, err := git.Init(filesystem.NewStorageWithOptions(gitDir,
		cache.NewObjectLRUDefault(), filesystem.Options{
			ExclusiveAccess: true,
		}), osfs.New(baseDir))
	require.NoError(t, err)
	return repo
}

func testRepoSingleCommitPastRelease(t *testing.T, repo *git.Repository) {
	workTree, err := repo.Worktree()
	require.NoError(t, err)

	// alpha tag
	workDir := workTree.Filesystem
	err = writeFile(workDir, "hello-world.alpha", "Hello World")
	require.NoError(t, err)

	_, err = workTree.Add("hello-world.alpha")
	require.NoError(t, err)
	alphaCommitHash, err := workTree.Commit("Initial Commit!", &git.CommitOptions{Author: testSignature})
	require.NoError(t, err)

	_, err = repo.CreateTag("v1.0.0-alpha.1", alphaCommitHash, nil)
	require.NoError(t, err)

	// v1.0.0 tag
	err = writeFile(workDir, "hello-world", "Hello World")
	require.NoError(t, err)

	_, err = workTree.Add("hello-world")
	require.NoError(t, err)
	tagCommitHash, err := workTree.Commit("Initial Commit!", &git.CommitOptions{Author: testSignature})
	require.NoError(t, err)
	_, err = repo.CreateTag("v1.0.0", tagCommitHash, nil)
	require.NoError(t, err)

	// commit after tag
	err = writeFile(workDir, "hello-world2", "Hello World 2")
	require.NoError(t, err)
	_, err = workTree.Add("hello-world2")
	require.NoError(t, err)
	postReleaseCommit, err := workTree.Commit("Subsequent Commit!", &git.CommitOptions{Author: testSignature})
	require.NoError(t, err)

	// make the commit after tag a new beta tag
	_, err = repo.CreateTag("v2.0.0-beta.1", postReleaseCommit, nil)
	require.NoError(t, err)
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
