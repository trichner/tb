package tags

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/manifoldco/promptui/list"

	"github.com/Masterminds/semver/v3"
	"github.com/manifoldco/promptui"
)

const tagPrefix = "refs/tags/"

type cli struct{}

func Exec(ctx context.Context, args []string) {
	// kong expects only actual arguments and not the program itself
	args = args[1:]

	var flags cli

	k, err := kong.New(&flags)
	if err != nil {
		log.Fatalf("cannot parse arguments: %v", err)
	}
	_, err = k.Parse(args)
	if err != nil {
		log.Fatalf("cannot parse arguments: %v", err)
	}

	if err := run(&flags); err != nil {
		slog.Error("failed to tag", "err", err)
		os.Exit(1)
	}
}

func run(opts *cli) error {
	repoPath := "."

	versionTags, err := remoteVersionTags(repoPath)
	if err != nil {
		return fmt.Errorf("cannot list remote tags for current repository: %w", err)
	}

	slog.Info("found remote tag groups", "count", len(versionTags))

	if len(versionTags) == 0 {
		versionTags[""] = []*versionTag{{version: semver.MustParse("0.0.0")}}
	}

	selectedGroup, err := selectGroup(versionTags)
	if err != nil {
		return fmt.Errorf("cannot select version group: %w", err)
	}

	versions := versionTags[selectedGroup]
	if len(versions) == 0 {
		versions = []*versionTag{{version: semver.MustParse("0.0.0")}}
	}

	slog.Info("found previous versions", "count", len(versions))

	lastVersion := versions[len(versions)-1]

	oldTag := lastVersion.version.Original()
	if selectedGroup != "" {
		oldTag = selectedGroup + "/" + oldTag
	}

	slog.Info("latest version", "version", oldTag)

	//TBD: automatically bump version?
	newVersion := lastVersion.version

	newTag := "v" + newVersion.String()
	if selectedGroup != "" {
		newTag = selectedGroup + "/" + newTag
	}

	newTag, err = editTag(newTag)
	if err != nil {
		return err
	}

	// IMPROVEMENT: ask for edit?
	slog.Info("next version", "version", newTag)

	tagged, err := tagIfDesired(repoPath, lastVersion.version.Original(), newTag)
	if err != nil {
		return err
	}

	if tagged {
		if err = pushIfDesired(repoPath, newTag); err != nil {
			return err
		}
	}

	return err
}

func editTag(newTag string) (string, error) {
	p := promptui.Prompt{
		Label:     "new tag",
		Default:   newTag,
		AllowEdit: true,
		Validate: func(s string) error {
			_, _, err := parseVersionWithGroup(s)
			return err
		},
	}
	return p.Run()
}

func selectGroup(groupedVersions map[string][]*versionTag) (string, error) {
	if len(groupedVersions) == 0 {
		return "", fmt.Errorf("no previous versions found")
	}

	if len(groupedVersions) == 1 {
		for k := range groupedVersions {
			return k, nil
		}
	}

	var groups []string
	for g := range groupedVersions {
		groups = append(groups, g)
	}
	sort.Strings(groups)

	labels := make([]string, len(groups))
	copy(labels, groups)
	for i := range groups {
		if groups[i] == "" {
			labels[i] = "-"
			break
		}
	}

	p := promptui.Select{
		Label:             "select tag group",
		Items:             labels,
		CursorPos:         0,
		Size:              min(len(groups), 16),
		Searcher:          caseInsensitivePrefixSearcher(labels),
		StartInSearchMode: true,
	}

	sel, _, err := p.Run()
	if err != nil {
		return "", err
	}
	return groups[sel], nil
}

func caseInsensitivePrefixSearcher(values []string) list.Searcher {
	return func(input string, index int) bool {
		return strings.HasPrefix(
			strings.TrimSpace(strings.ToLower(values[index])),
			strings.TrimSpace(strings.ToLower(input)))
	}
}

func proceedWithCurrentBranch(clonePath string) (bool, error) {
	b, err := currentBranch(clonePath)
	if err != nil {
		return false, err
	}

	if b == "main" {
		return true, nil
	}

	return allowNonMainTag()
}

func allowNonMainTag() (bool, error) {
	p := promptui.Select{
		Label:     fmt.Sprintf("current branch is not 'main', tag anyways?"),
		Items:     []string{"yes, I'm brave 🔥", "no, I'm reasonable"},
		CursorPos: 1,
	}

	sel, _, err := p.Run()
	if err != nil {
		return false, err
	}
	return sel == 0, nil
}

func tagIfDesired(clonePath string, latestVersion, nextVersion string) (bool, error) {
	y, err := shouldTag(latestVersion, nextVersion)
	if err != nil {
		return false, err
	}
	if !y {
		slog.Info("skip tagging")
		return false, nil
	}

	proceed, err := proceedWithCurrentBranch(clonePath)
	if !proceed || err != nil {
		return false, err
	}

	slog.Info("tagging next version", "version", nextVersion)
	_, err = execGitTag(clonePath, nextVersion)

	return err == nil, err
}

func shouldTag(latestVersion, nextVersion string) (bool, error) {
	p := promptui.Select{
		Label: fmt.Sprintf("latest version is '%s', do you want to tag '%s'?", latestVersion, nextVersion),
		Items: []string{"yes 🏷️", "no"},
	}

	sel, _, err := p.Run()
	if err != nil {
		return false, err
	}
	return sel == 0, nil
}

func pushIfDesired(clonePath string, tag string) error {
	y, err := shouldPush(tag)
	if err != nil {
		return err
	}
	if !y {
		slog.Info("skip pushing")
		return nil
	}

	_, err = execGitPushTag(clonePath, tag)
	return err
}

func shouldPush(tag string) (bool, error) {
	p := promptui.Select{
		Label: fmt.Sprintf("do you want to push the '%s' tag?", tag),
		Items: []string{"yes 🚀", "no"},
	}

	sel, _, err := p.Run()
	if err != nil {
		return false, err
	}
	return sel == 0, nil
}

type versionTag struct {
	id      string
	version *semver.Version
}

func remoteVersionTags(clonePath string) (map[string][]*versionTag, error) {
	// list all tags
	tags, err := listRemoteTags(clonePath)
	if err != nil {
		return nil, fmt.Errorf("cannot list remote tags for current repository: %w", err)
	}

	slog.Info("remote tags listed", "count", len(tags))

	versions := make(map[string][]*versionTag)

	for _, t := range tags {
		group, version, err := parseVersionWithGroup(t.Name)
		if err == nil {
			versions[group] = append(versions[group], &versionTag{
				id:      t.ID,
				version: version,
			})
		} else {
			slog.Warn("tag is not a valid semver version", "tag", t.Name)
		}
	}

	for g, v := range versions {
		sort.Sort(byVersion(v))
		versions[g] = v
	}

	return versions, nil
}

func parseVersionWithGroup(name string) (group string, version *semver.Version, err error) {
	pattern := regexp.MustCompile("^((.+)/)?v[0-9]+(\\.[0-9]+){0,2}.*$")
	parsed := pattern.FindStringSubmatch(name)
	if parsed == nil {
		return "", nil, fmt.Errorf("tag '%s' does not match pattern", name)
	}

	prefix := parsed[1]
	group = parsed[2]

	v := strings.TrimPrefix(name, prefix)

	version, err = semver.NewVersion(v)
	return group, version, err
}

type byVersion []*versionTag

func (a byVersion) Len() int           { return len(a) }
func (a byVersion) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byVersion) Less(i, j int) bool { return a[i].version.LessThan(a[j].version) }

type ref struct {
	ID, Name string
}

func listRemoteTags(clonePath string) ([]*ref, error) {
	var tags []*ref

	refs, err := listRemoteRefs(clonePath)
	if err != nil {
		return nil, err
	}

	for _, r := range refs {
		if strings.HasPrefix(r.Name, tagPrefix) {
			tags = append(tags, &ref{ID: r.ID, Name: strings.TrimPrefix(r.Name, tagPrefix)})
		}
	}

	return tags, nil
}

func listRemoteRefs(clonePath string) ([]*ref, error) {
	s, err := execGitLsRemoteTags(clonePath)
	if err != nil {
		return nil, fmt.Errorf("cannot list remote tags: %w", err)
	}

	return parseRefs(s)
}

func parseRefs(raw string) ([]*ref, error) {
	var refs []*ref
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		parsed, err := parseRefLine(line)
		if err != nil {
			return nil, err
		}
		refs = append(refs, parsed)
	}

	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return refs, nil
}

// 709b403093309e014853cbecd4731143c694163d        refs/heads/release/v4.49
var refPattern = regexp.MustCompile("^([a-f0-9]{40})[ \t]+([^ ]+)$")

func parseRefLine(line string) (*ref, error) {
	matches := refPattern.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("failed to parse ref line: %q", line)
	}
	return &ref{
		ID:   matches[1],
		Name: matches[2],
	}, nil
}

func currentBranch(clonePath string) (string, error) {
	ret, err := execute(clonePath, "git", "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("cannot determine current branch: %w", err)
	}
	ret = strings.TrimSpace(ret)
	return ret, nil
}

func execGitLsRemoteTags(clonePath string) (string, error) {
	return execute(clonePath, "git", "ls-remote", "--tags", "--refs")
}

func execGitPushTag(clonePath, tag string) (string, error) {
	return execute(clonePath, "git", "push", "origin", tagPrefix+tag)
}

func execGitTag(clonePath, tag string) (string, error) {
	return execute(clonePath, "git", "tag", tag)
}

func execute(workingDir string, prog string, args ...string) (string, error) {
	cmd := exec.Command(prog, args...)
	cmd.Dir = workingDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
