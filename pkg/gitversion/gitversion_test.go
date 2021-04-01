package gitversion

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripModuleTagPrefixes(t *testing.T) {
	require.Equal(t, "0.0.0", StripModuleTagPrefixes("v0.0.0"))
	require.Equal(t, "2.1.0", StripModuleTagPrefixes("sdk/v2.1.0"))
	require.Equal(t, "2.1.0", StripModuleTagPrefixes("sdk/nodejs/v2.1.0"))
}

func TestMostRecentTag(t *testing.T) {
	t.Run("Repo with commit after tag", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		repo, err = testRepoSingleCommitPastRelease(repo)
		require.NoError(t, err)
		require.NotNil(t, repo)

		headRef, err := repo.Head()
		require.NoError(t, err)
		require.NotEmpty(t, headRef)

		hasMostRecent, mostRecent, err := mostRecentTag(repo, headRef.Hash())
		require.NoError(t, err)
		require.True(t, hasMostRecent)
		require.NotNil(t, mostRecent)
		require.Equal(t, "refs/tags/v1.0.0", mostRecent.Name().String())
	})

	t.Run("Repo with no tags", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		repo, err = testRepoSingleCommit(repo)
		require.NoError(t, err)
		require.NotNil(t, repo)

		headRef, err := repo.Head()
		require.NoError(t, err)
		require.NotEmpty(t, headRef)

		hasMostRecent, mostRecent, err := mostRecentTag(repo, headRef.Hash())
		require.NoError(t, err)
		require.False(t, hasMostRecent)
		require.Nil(t, mostRecent)
	})

}

func TestIsExactTag(t *testing.T) {
	repo, err := testRepoCreate()
	require.NoError(t, err)
	repo, err = testRepoSingleCommitPastRelease(repo)
	require.NoError(t, err)
	require.NotNil(t, repo)

	headRef, err := repo.Head()
	require.NoError(t, err)
	require.NotEmpty(t, headRef)

	t.Run("Not an exact tag", func(t *testing.T) {
		isExact, exact, err := isExactTag(repo, headRef.Hash())
		require.NoError(t, err)
		require.Nil(t, exact)
		require.False(t, isExact)
	})

	t.Run("With exact tag - prerelease", func(t *testing.T) {
		exactRef, err := repo.Tag("v1.0.0-alpha.1")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash())
		require.NoError(t, err)
		require.NotNil(t, exact)
		require.True(t, isExact)
	})

	t.Run("With exact tag", func(t *testing.T) {
		exactRef, err := repo.Tag("v1.0.0")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash())
		require.NoError(t, err)
		require.NotNil(t, exact)
		require.True(t, isExact)
	})

	t.Run("Skip the beta tag", func(t *testing.T) {
		exactRef, err := repo.Tag("v2.0.0-beta.1")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash())
		require.NoError(t, err)
		require.Nil(t, exact)
		require.False(t, isExact)
	})
}

func TestIsWorktreeDirty(t *testing.T) {
	dir, err := ioutil.TempDir("", "worktree")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	repo, err := testRepoFSCreate(dir)
	require.NoError(t, err)
	repo, err = testRepoSingleCommit(repo)
	require.NoError(t, err)
	require.NotNil(t, repo)

	t.Run("Working tree is clean", func(t *testing.T) {
		clean, err := workTreeIsDirty(repo)
		require.NoError(t, err)
		require.False(t, clean)
	})

	// Add a file but don't commit it
	worktree, err := repo.Worktree()
	if err != nil {
		t.Errorf("worktree: %w", err)
	}

	workDir := worktree.Filesystem

	// Write a file but don't commit it
	if err := writeFile(workDir, "hello-world", "Hello World 2"); err != nil {
		t.Errorf("writeFile: %w", err)
	}

	t.Run("Working tree is dirty", func(t *testing.T) {
		dirty, err := workTreeIsDirty(repo)
		require.NoError(t, err)
		require.True(t, dirty)
	})
}
