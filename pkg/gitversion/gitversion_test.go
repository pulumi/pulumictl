package gitversion

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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
		head, err := testRepoSingleCommit(repo)
		require.NoError(t, err)
		require.NotEmpty(t, head)

		hasMostRecent, mostRecent, err := mostRecentTag(repo, head, false, nil)
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
	head, err := testRepoSingleCommit(repo)
	require.NoError(t, err)
	require.NotEmpty(t, head)

	t.Run("Working tree is clean", func(t *testing.T) {
		clean, err := workTreeIsDirty(repo)
		require.NoError(t, err)
		require.False(t, clean)
	})

	// Add a file but don't commit it
	worktree, err := repo.Worktree()
	if err != nil {
		t.Errorf("worktree: %s", err)
	}

	workDir := worktree.Filesystem

	// Write a file but don't commit it
	if err := writeFile(workDir, "hello-world", "Hello World 2"); err != nil {
		t.Errorf("writeFile: %s", err)
	}

	t.Run("Working tree is dirty", func(t *testing.T) {
		dirty, err := workTreeIsDirty(repo)
		require.NoError(t, err)
		require.True(t, dirty)
	})
}

func TestGetVersion(t *testing.T) {

	t.Run("Repo with no tags", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		_, err = testRepoSingleCommit(repo)
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "0.0.1-alpha.0+68804cfa", version.SemVer)
		require.Equal(t, "0.0.1-alpha.0+68804cfa", version.DotNet)
		require.Equal(t, "v0.0.1-alpha.0+68804cfa", version.JavaScript)
		require.Equal(t, "0.0.1a0", version.Python)
	})

	t.Run("Repo with exact tag", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0", version.SemVer)
		require.Equal(t, "1.0.0", version.DotNet)
		require.Equal(t, "v1.0.0", version.JavaScript)
		require.Equal(t, "1.0.0", version.Python)
	})

	t.Run("Repo with with commit after tag", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		workTree, err := repo.Worktree()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		// add another commit
		addFile(t, workTree, "hello.txt", "Hello world")
		_, err = workTree.Commit("Next commit", &git.CommitOptions{Author: testSignature})
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.1.0-alpha.0+9fa804e8", version.SemVer)
		require.Equal(t, "1.1.0-alpha.0+9fa804e8", version.DotNet)
		require.Equal(t, "v1.1.0-alpha.0+9fa804e8", version.JavaScript)
		require.Equal(t, "1.1.0a0", version.Python)
	})

	t.Run("Repo with with commit after tag and dirty", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		workTree, err := repo.Worktree()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		// add another commit
		addFile(t, workTree, "hello.txt", "Hello world")
		_, err = workTree.Commit("Next commit", &git.CommitOptions{Author: testSignature})
		require.NoError(t, err)

		// Write a file but don't commit it
		workDir := workTree.Filesystem
		err = writeFile(workDir, "hello-world", "Hello World 2")
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.1.0-alpha.0+9fa804e8.dirty", version.SemVer)
		require.Equal(t, "1.1.0-alpha.0+9fa804e8.dirty", version.DotNet)
		require.Equal(t, "v1.1.0-alpha.0+9fa804e8.dirty", version.JavaScript)
		require.Equal(t, "1.1.0a0+dirty", version.Python)
	})

	t.Run("Repo with no tags and dirty", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		workTree, err := repo.Worktree()
		require.NoError(t, err)
		_, err = testRepoSingleCommit(repo)
		require.NoError(t, err)

		// Write a file but don't commit it
		workDir := workTree.Filesystem
		err = writeFile(workDir, "hello-world", "Hello World 2")
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "0.0.1-alpha.0+68804cfa.dirty", version.SemVer)
		require.Equal(t, "0.0.1-alpha.0+68804cfa.dirty", version.DotNet)
		require.Equal(t, "v0.0.1-alpha.0+68804cfa.dirty", version.JavaScript)
		require.Equal(t, "0.0.1a0+dirty", version.Python)
	})

	t.Run("Repo with alpha tag and dirty", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		workTree, err := repo.Worktree()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0-alpha.1",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		// Write a file but don't commit it
		workDir := workTree.Filesystem
		err = writeFile(workDir, "hello-world", "Hello World 2")
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0-alpha.1+e624a7d7.dirty", version.SemVer)
		require.Equal(t, "1.0.0-alpha.1+e624a7d7.dirty", version.DotNet)
		require.Equal(t, "v1.0.0-alpha.1+e624a7d7.dirty", version.JavaScript)
		require.Equal(t, "1.0.0a1+dirty", version.Python)
	})

	t.Run("Repo with annotated tag", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)

		workTree, err := repo.Worktree()
		require.NoError(t, err)

		err = writeFile(workTree.Filesystem, "test.txt", "Hello world")
		require.NoError(t, err)
		_, err = workTree.Add("test.txt")
		require.NoError(t, err)

		tagCommitHash, err := workTree.Commit("Commit for v1.0.0", &git.CommitOptions{Author: testSignature})
		require.NoError(t, err)

		_, err = repo.CreateTag("v1.0.0", tagCommitHash, &git.CreateTagOptions{
			Message: "version 1",
			Tagger: &object.Signature{
				Name:  "test",
				Email: "test@example.com",
			},
		})
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0", version.SemVer)
		require.Equal(t, "1.0.0", version.DotNet)
		require.Equal(t, "v1.0.0", version.JavaScript)
		require.Equal(t, "1.0.0", version.Python)
	})

	t.Run("Repo with exact tag and dirty", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		workTree, err := repo.Worktree()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		// Write a file but don't commit it
		workDir := workTree.Filesystem
		err = writeFile(workDir, "hello-world", "Hello World 2")
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0+dirty", version.SemVer)
		require.Equal(t, "1.0.0+dirty", version.DotNet)
		require.Equal(t, "v1.0.0+dirty", version.JavaScript)
		require.Equal(t, "1.0.0+dirty", version.Python)
	})

	t.Run("Repo with un-dotted alpha tag", func(t *testing.T) {
		// Regression test for https://github.com/pulumi/pulumictl/issues/50

		repo, err := testRepoCreate()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0-alpha",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0-alpha+26d1c29c", version.SemVer)
		require.Equal(t, "1.0.0-alpha+26d1c29c", version.DotNet)
		require.Equal(t, "v1.0.0-alpha+26d1c29c", version.JavaScript)
		require.Equal(t, "1.0.0a0", version.Python)
	})

	t.Run("Repo with un-dotted alpha tag marked for pre-release", func(t *testing.T) {
		// Regression test for https://gi thub.com/pulumi/pulumictl/issues/50

		repo, err := testRepoCreate()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0-alpha",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			IsPreRelease: true,
			Repo:         repo,
			Commitish:    plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0-alpha", version.SemVer)
		require.Equal(t, "1.0.0-alpha", version.DotNet)
		require.Equal(t, "v1.0.0-alpha", version.JavaScript)
		require.Equal(t, "1.0.0a0", version.Python)
	})

	t.Run("Repo with dotted alpha tag marked for pre-release", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0-alpha.1",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			IsPreRelease: true,
			Repo:         repo,
			Commitish:    plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0-alpha.1", version.SemVer)
		require.Equal(t, "1.0.0-alpha.1", version.DotNet)
		require.Equal(t, "v1.0.0-alpha.1", version.JavaScript)
		require.Equal(t, "1.0.0a1", version.Python)
	})

	t.Run("Repo with build info tag", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0+1",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0+1", version.SemVer)
		require.Equal(t, "1.0.0+1", version.DotNet)
		require.Equal(t, "v1.0.0+1", version.JavaScript)
		require.Equal(t, "1.0.0+1", version.Python)
	})

	t.Run("Repo with complicated build info tag", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0+1abc.345.whoop",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		opts := LanguageVersionsOptions{
			Repo:      repo,
			Commitish: plumbing.Revision("HEAD"),
		}
		version, err := GetLanguageVersionsWithOptions(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0+1abc.345.whoop", version.SemVer)
		require.Equal(t, "1.0.0+1abc.345.whoop", version.DotNet)
		require.Equal(t, "v1.0.0+1abc.345.whoop", version.JavaScript)
		require.Equal(t, "1.0.0+1abc.345.whoop", version.Python)
	})
}
