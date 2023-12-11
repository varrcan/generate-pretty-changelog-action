package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/varrcan/generate-pretty-changelog/pkg/context"
	"github.com/varrcan/generate-pretty-changelog/pkg/git"
)

// errInvalidSortDirection happens when the sort order is invalid.
var errInvalidSortDirection = errors.New("invalid sort direction")

const li = "* "

const (
	useGit    = "git"
	useGitHub = "github"
)

// generate changelog
func generate(ctx *context.Context) error {
	if err := checkSortDirection(ctx.Config.Changelog.Sort); err != nil {
		return err
	}

	entries, err := buildChangelog(ctx)
	if err != nil {
		return err
	}

	changes, err := formatChangelog(ctx, entries)
	if err != nil {
		return err
	}
	changelogElements := []string{changes}

	ctx.ReleaseNotes = strings.Join(changelogElements, "\n\n")
	if !strings.HasSuffix(ctx.ReleaseNotes, "\n") {
		ctx.ReleaseNotes += "\n"
	}

	return os.WriteFile("CHANGELOG.md", []byte(ctx.ReleaseNotes), 0o644) //nolint: gosec
}

type changelogGroup struct {
	title   string
	entries []string
	order   int
}

func title(s string, level int) string {
	if s == "" {
		return ""
	}
	return fmt.Sprintf("%s %s", strings.Repeat("#", level), s)
}

func newLineFor() string {
	return "\n"
}

func abbrevEntry(s string, abbr int) string {
	switch abbr {
	case 0:
		return s
	case -1:
		_, rest, _ := strings.Cut(s, " ")
		return rest
	default:
		commit, rest, _ := strings.Cut(s, " ")
		if abbr > len(commit) {
			return s
		}
		return fmt.Sprintf("%s %s", commit[:abbr], rest)
	}
}

func abbrev(entries []string, abbr int) []string {
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, abbrevEntry(entry, abbr))
	}
	return result
}

func formatChangelog(ctx *context.Context, entries []string) (string, error) {
	entries = abbrev(entries, ctx.Config.Changelog.Abbrev)

	result := []string{title("Changelog", 2)}
	if len(ctx.Config.Changelog.Groups) == 0 {
		return strings.Join(append(result, filterAndPrefixItems(entries)...), newLineFor()), nil
	}

	var groups []changelogGroup
	for _, group := range ctx.Config.Changelog.Groups {
		item := changelogGroup{
			title: title(group.Title, 3),
			order: group.Order,
		}
		if group.Regexp == "" {
			// If no regexp is provided, we purge all strikethrough entries and add remaining entries to the list
			item.entries = filterAndPrefixItems(entries)
			// clear array
			entries = nil
		} else {
			re, err := regexp.Compile(group.Regexp)
			if err != nil {
				return "", fmt.Errorf("failed to group into %q: %w", group.Title, err)
			}

			i := 0
			for _, entry := range entries {
				match := re.MatchString(entry)
				if match {
					item.entries = append(item.entries, li+entry)
				} else {
					// Keep unmatched entry.
					entries[i] = entry
					i++
				}
			}
			entries = entries[:i]
		}
		groups = append(groups, item)

		if len(entries) == 0 {
			break // No more entries to process.
		}
	}

	sort.Slice(groups, groupSort(groups))
	for _, group := range groups {
		if len(group.entries) > 0 {
			result = append(result, group.title)
			result = append(result, group.entries...)
		}
	}
	return strings.Join(result, newLineFor()), nil
}

func groupSort(groups []changelogGroup) func(i, j int) bool {
	return func(i, j int) bool {
		return groups[i].order < groups[j].order
	}
}

func filterAndPrefixItems(ss []string) []string {
	var r []string
	for _, s := range ss {
		if s != "" {
			r = append(r, li+s)
		}
	}
	return r
}

func checkSortDirection(mode string) error {
	switch mode {
	case "", "asc", "desc":
		return nil
	default:
		return errInvalidSortDirection
	}
}

func buildChangelog(ctx *context.Context) ([]string, error) {
	l, err := getChangeLogger(ctx)
	if err != nil {
		return nil, err
	}
	log, err := l.Log(ctx)
	if err != nil {
		return nil, err
	}
	entries := strings.Split(log, "\n")
	if lastLine := entries[len(entries)-1]; strings.TrimSpace(lastLine) == "" {
		entries = entries[0 : len(entries)-1]
	}
	entries, err = filterEntries(ctx, entries)
	if err != nil {
		return entries, err
	}
	return sortEntries(ctx, entries), nil
}

func filterEntries(ctx *context.Context, entries []string) ([]string, error) {
	filters := ctx.Config.Changelog.Filters
	if len(filters.Include) > 0 {
		var newEntries []string
		for _, filter := range filters.Include {
			r, err := regexp.Compile(filter)
			if err != nil {
				return entries, err
			}
			newEntries = append(newEntries, keep(r, entries)...)
		}
		return newEntries, nil
	}
	for _, filter := range filters.Exclude {
		r, err := regexp.Compile(filter)
		if err != nil {
			return entries, err
		}
		entries = remove(r, entries)
	}
	return entries, nil
}

func sortEntries(ctx *context.Context, entries []string) []string {
	direction := ctx.Config.Changelog.Sort
	if direction == "" {
		return entries
	}
	result := make([]string, len(entries))
	copy(result, entries)
	sort.Slice(result, func(i, j int) bool {
		imsg := extractCommitInfo(result[i])
		jmsg := extractCommitInfo(result[j])
		if direction == "asc" {
			return strings.Compare(imsg, jmsg) < 0
		}
		return strings.Compare(imsg, jmsg) > 0
	})
	return result
}

func keep(filter *regexp.Regexp, entries []string) (result []string) {
	for _, entry := range entries {
		if filter.MatchString(extractCommitInfo(entry)) {
			result = append(result, entry)
		}
	}
	return result
}

func remove(filter *regexp.Regexp, entries []string) (result []string) {
	for _, entry := range entries {
		if !filter.MatchString(extractCommitInfo(entry)) {
			result = append(result, entry)
		}
	}
	return result
}

func extractCommitInfo(line string) string {
	return strings.Join(strings.Split(line, " ")[1:], " ")
}

func getChangeLogger(ctx *context.Context) (changeLogger, error) {
	switch ctx.Config.Changelog.Use {
	case useGit:
		fallthrough
	case "":
		return gitChangeLogger{}, nil
	case useGitHub:
		return newSCMChangeLogger(ctx)
	default:
		return nil, fmt.Errorf("invalid changelog.use: %q", ctx.Config.Changelog.Use)
	}
}

type changeLogger interface {
	Log(ctx *context.Context) (string, error)
}

func newSCMChangeLogger(ctx *context.Context) (changeLogger, error) {
	cli, err := git.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := git.ExtractRepoFromConfig(ctx)
	if err != nil {
		return nil, err
	}
	if err := repo.CheckSCM(); err != nil {
		return nil, err
	}
	return &scmChangeLogger{
		client: cli,
		repo: git.Repo{
			Owner: repo.Owner,
			Name:  repo.Name,
		},
	}, nil
}

type gitChangeLogger struct{}

var validSHA1 = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

type scmChangeLogger struct {
	client git.Client
	repo   git.Repo
}

// Log returns a changelog
func (c *scmChangeLogger) Log(ctx *context.Context) (string, error) {
	prev, current := comparePair(ctx)
	return c.client.Changelog(ctx, c.repo, prev, current)
}

// Log returns a changelog
func (g gitChangeLogger) Log(ctx *context.Context) (string, error) {
	args := []string{"log", "--pretty=oneline", "--abbrev-commit", "--no-decorate", "--no-color"}
	prev, current := comparePair(ctx)
	if validSHA1.MatchString(prev) {
		args = append(args, prev, current)
	} else {
		args = append(args, fmt.Sprintf("tags/%s..tags/%s", ctx.Git.PreviousTag, ctx.Git.CurrentTag))
	}
	return git.Exec(ctx, args...)
}

func comparePair(ctx *context.Context) (prev string, current string) {
	prev = ctx.Git.PreviousTag
	current = ctx.Git.CurrentTag
	if prev == "" {
		prev = ctx.Git.FirstCommit
	}
	return
}
