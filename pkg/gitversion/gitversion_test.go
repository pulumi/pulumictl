package gitversion

import (
	"io/ioutil"
	"os"
	"strings"
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

		hasMostRecent, mostRecent, err := mostRecentTag(repo, headRef.Hash(), false, nil)
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

		hasMostRecent, mostRecent, err := mostRecentTag(repo, headRef.Hash(), false, nil)
		require.NoError(t, err)
		require.False(t, hasMostRecent)
		require.Nil(t, mostRecent)
	})

	t.Run("Repo with mod-prefixed tags", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0-alpha.1",
			"mod/v0.0.1-alpha.1",
			"v1.0.0",
			"mod/v0.0.1",
			"mod/v0.0.2-beta.1",
			"v2.0.0-beta.1",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		headRef, err := repo.Head()
		require.NoError(t, err)
		require.NotEmpty(t, headRef)

		hasModPrefix := func(tag string) bool {
			return strings.HasPrefix(tag, "mod/")
		}

		noPrefix := func(tag string) bool {
			return !strings.Contains(tag, "/")
		}

		isMostRecent := func(expected string, preRelease bool, tagFilter func(string) bool) {
			hasMostRecent, mostRecent, err := mostRecentTag(repo,
				headRef.Hash(), preRelease, tagFilter)
			require.NoError(t, err)
			require.True(t, hasMostRecent)
			require.Equal(t, expected, mostRecent.Name().String())
		}

		isMostRecent("refs/tags/mod/v0.0.1", false, hasModPrefix)
		isMostRecent("refs/tags/mod/v0.0.2-beta.1", true, hasModPrefix)
		isMostRecent("refs/tags/v1.0.0", false, noPrefix)
		isMostRecent("refs/tags/v2.0.0-beta.1", true, noPrefix)
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
		isExact, exact, err := isExactTag(repo, headRef.Hash(), false, nil)
		require.NoError(t, err)
		require.Nil(t, exact)
		require.False(t, isExact)
	})

	t.Run("With exact tag - prerelease", func(t *testing.T) {
		exactRef, err := repo.Tag("v1.0.0-alpha.1")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash(), false, nil)
		require.NoError(t, err)
		require.NotNil(t, exact)
		require.True(t, isExact)
	})

	t.Run("With exact tag", func(t *testing.T) {
		exactRef, err := repo.Tag("v1.0.0")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash(), false, nil)
		require.NoError(t, err)
		require.NotNil(t, exact)
		require.True(t, isExact)
	})

	t.Run("Don't skip the beta tag as it's a pre-release", func(t *testing.T) {
		exactRef, err := repo.Tag("v2.0.0-beta.1")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash(), true, nil)
		require.NoError(t, err)
		require.NotNil(t, exact)
		require.True(t, isExact)
	})

	t.Run("Skip the beta as it's a normal release", func(t *testing.T) {
		exactRef, err := repo.Tag("v2.0.0-beta.1")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash(), false, nil)
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
