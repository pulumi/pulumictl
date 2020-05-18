package gitversion

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripModuleTagPrefixes(t *testing.T) {
	require.Equal(t, "0.0.0", stripModuleTagPrefixes("v0.0.0"))
	require.Equal(t, "2.1.0", stripModuleTagPrefixes("sdk/v2.1.0"))
	require.Equal(t, "2.1.0", stripModuleTagPrefixes("sdk/nodejs/v2.1.0"))
}

func TestMostRecentTag(t *testing.T) {
	t.Run("Repo with commit after tag", func(t *testing.T) {
		repo, err := testRepoSingleCommitPastRelease()
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
		repo, err := testRepoSingleCommit()
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
	repo, err := testRepoSingleCommitPastRelease()
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

	t.Run("With exact tag", func(t *testing.T) {
		exactRef, err := repo.Tag("v1.0.0")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash())
		require.NoError(t, err)
		require.NotNil(t, exact)
		require.True(t, isExact)
	})
}

func TestIsWorktreeDirty(t *testing.T) {
	repo, err := testRepoSingleCommit()
	require.NoError(t, err)
	require.NotNil(t, repo)

	// Add a file but don't commit it
	worktree, err := repo.Worktree()
	if err != nil {
		t.Errorf("worktree: %w", err)
	}
	t.Log(worktree.Status())

	t.Run("Working tree is clean", func(t *testing.T) {
		clean, err := workTreeIsDirty(repo)
		require.NoError(t, err)
		require.False(t, clean)
	})

	workDir := worktree.Filesystem

	// Write a file but don't commit it
	if err := writeFile(workDir, "foo", "bar"); err != nil {
		t.Errorf("writeFile: %w", err)
	}

	t.Run("Working tree is dirty", func(t *testing.T) {
		dirty, err := workTreeIsDirty(repo)
		require.NoError(t, err)
		require.True(t, dirty)
	})
}
