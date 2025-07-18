package tags

import (
	_ "embed"
	"sort"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/trichner/tb/pkg/assert"
)

func TestParseRef(t *testing.T) {
	raw := "a278d1424a011185748c4690c99c10a6d0676435 refs/remotes/origin/feature/spicedb-binding"

	got, err := parseRefLine(raw)

	assert.NoError(t, err)
	assert.Equal(t, got.ID, "a278d1424a011185748c4690c99c10a6d0676435")
	assert.Equal(t, got.Name, "refs/remotes/origin/feature/spicedb-binding")
}

//go:embed fixture_refs.txt
var rawRefs string

func TestParseRefs(t *testing.T) {
	got, err := parseRefs(rawRefs)

	assert.NoError(t, err)
	assert.Equal(t, len(got), 51)
}

func TestSortVersions(t *testing.T) {
	versions := []*versionTag{
		{"c0ff33", semver.MustParse("v1.0.1-rc.1")},
		{"c0ff33", semver.MustParse("v7.0.1")},
		{"c0ff33", semver.MustParse("v0.0.1-rc.1")},
		{"c0ff33", semver.MustParse("v7.0.1-rc.1")},
		{"c0ff33", semver.MustParse("v7.0.1-dev.99")},
	}

	sort.Sort(byVersion(versions))

	assert.Equal(t, versions[0].version.Original(), "v0.0.1-rc.1")
	assert.Equal(t, versions[4].version.Original(), "v7.0.1")
	assert.Equal(t, versions[4].version.Major(), 7)
}

func Test_parseVersionWithGroup(t *testing.T) {
	s := "ocs-backend/v1.0.1-rc.1"
	got, got1, err := parseVersionWithGroup(s)
	assert.NoError(t, err)
	assert.Equal(t, got, "ocs-backend")
	assert.Equal(t, got1, semver.MustParse("v1.0.1-rc.1"))
}

func Test_parseVersionWithGroup2(t *testing.T) {
	s := "v1.0.1-rc.1"
	got, got1, err := parseVersionWithGroup(s)
	assert.NoError(t, err)
	assert.Equal(t, got, "")
	assert.True(t, got1.Equal(semver.MustParse("v1.0.1-rc.1")))
}
