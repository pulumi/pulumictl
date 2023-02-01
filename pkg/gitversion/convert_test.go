package gitversion

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type versionTest struct {
	desc   string
	semver string
	python string
}

func TestConversions(t *testing.T) {
	inputs := []versionTest{
		{
			semver: "0.0.0",
			python: "0.0.0",
		},
		{
			desc:   "Repo with no tags",
			semver: "0.0.1-alpha.0+68804cfa",
			python: "0.0.1a0",
		},
		{
			desc:   "Repo with exact tag",
			semver: "1.0.0",
			python: "1.0.0",
		},
		{
			desc:   "Repo with with commit after tag",
			semver: "1.1.0-alpha.0+9fa804e8",
			python: "1.1.0a0",
		},
		{
			desc:   "Repo with with commit after tag and dirty",
			semver: "1.1.0-alpha.0+9fa804e8.dirty",
			python: "1.1.0a0+dirty",
		},
		{
			desc:   "Repo with no tags and dirty",
			semver: "0.0.1-alpha.0+68804cfa.dirty",
			python: "0.0.1a0+dirty",
		},
		{
			desc:   "Repo with alpha tag and dirty",
			semver: "1.0.0-alpha.1+e624a7d7.dirty",
			python: "1.0.0a1+dirty",
		},
		{
			desc:   "Repo with exact tag and dirty",
			semver: "1.0.0+dirty",
			python: "1.0.0+dirty",
		},
		{
			desc:   "Repo with un-dotted alpha tag",
			semver: "1.0.0-alpha+26d1c29c",
			python: "1.0.0a0",
		},
		{
			desc:   "Repo with un-dotted alpha tag marked for pre-release",
			semver: "1.0.0-alpha",
			python: "1.0.0a0",
		},
		{
			desc:   "Repo with dotted alpha tag marked for pre-release",
			semver: "1.0.0-alpha.1",
			python: "1.0.0a1",
		},
		{
			desc:   "Repo with beta tag and dirty",
			semver: "1.0.0-beta.1+e624a7d7.dirty",
			python: "1.0.0b1+dirty",
		},
		{
			desc:   "Repo with rc tag and dirty",
			semver: "1.0.0-rc.1+e624a7d7.dirty",
			python: "1.0.0rc1+dirty",
		},
		{
			desc:   "Repo with dev tag and dirty",
			semver: "1.0.0-dev.1+e624a7d7.dirty",
			python: "1.0.0d1+dirty",
		},
		{
			desc:   "Master prerelease",
			semver: "1.93.1-alpha.1675198718+c586f7b1",
			python: "1.93.1a1675198718",
		},
		{
			desc:   "Master prerelease dirty",
			semver: "1.93.1-alpha.1675198718+c586f7b1.dirty",
			python: "1.93.1a1675198718+dirty",
		},
	}
	for _, v := range inputs {
		javascript := "v" + v.semver
		desc := v.desc
		if desc == "" {
			desc = "test"
		}
		t.Run(fmt.Sprintf("%s %s", desc, v.semver), func(t *testing.T) {
			v1, err := GetLanguageOptionsFromVersion(v.semver)
			require.NoError(t, err)
			require.Equal(t, v.semver, v1.SemVer)
			require.Equal(t, v.semver, v1.DotNet)
			require.Equal(t, javascript, v1.JavaScript)
			require.Equal(t, v.python, v1.Python)

			// Can round-trip from javascript too (with leading "v")
			v2, err := GetLanguageOptionsFromVersion(javascript)
			require.NoError(t, err)
			require.Equal(t, v.semver, v2.SemVer)
			require.Equal(t, v.semver, v2.DotNet)
			require.Equal(t, javascript, v2.JavaScript)
			require.Equal(t, v.python, v2.Python)
		})
	}
}
