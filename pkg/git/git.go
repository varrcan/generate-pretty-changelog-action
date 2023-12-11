package git

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path"
	"strings"

	"github.com/varrcan/generate-pretty-changelog/pkg/context"
)

// Run the git command
func Run(ctx *context.Context) error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git not present in PATH")
	}
	info, err := getInfo(ctx)
	if err != nil {
		return err
	}
	ctx.Git = info
	ctx.Version = strings.TrimPrefix(ctx.Git.CurrentTag, "v")
	return validate(ctx)
}

// ExtractRepoFromConfig gets the repo name from the Git config.
func ExtractRepoFromConfig(ctx *context.Context) (result Repo, err error) {
	if !isRepo(ctx) {
		return result, errors.New("current folder is not a git repository")
	}
	out, err := clean(Exec(ctx, "ls-remote", "--get-url"))
	if err != nil {
		return result, fmt.Errorf("no remote configured to list refs from")
	}
	return extractRepoFromURL(out)
}

// extractRepoFromURL gets the repo name from the URL
func extractRepoFromURL(rawurl string) (Repo, error) {
	s := strings.TrimSuffix(strings.TrimSpace(rawurl), ".git")
	if strings.Count(s, ":") == 1 {
		s = s[strings.LastIndex(s, ":")+1:]
	}

	u, err := url.Parse(s)
	if err != nil {
		return Repo{
			RawURL: rawurl,
		}, err
	}

	ss := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(ss) == 0 || ss[0] == "" {
		return Repo{
			RawURL: rawurl,
		}, fmt.Errorf("unsupported repository URL: %s", rawurl)
	}

	if len(ss) < 2 {
		return Repo{
			RawURL: rawurl,
			Owner:  ss[0],
		}, nil
	}
	repo := Repo{
		RawURL: rawurl,
		Owner:  path.Join(ss[:len(ss)-1]...),
		Name:   ss[len(ss)-1],
	}
	return repo, nil
}

func getInfo(ctx *context.Context) (context.GitInfo, error) {
	if !isRepo(ctx) {
		return context.GitInfo{}, errors.New("current folder is not a git repository")
	}
	info, err := getGitInfo(ctx)

	return info, err
}

func getGitInfo(ctx *context.Context) (context.GitInfo, error) {
	full, err := getFullCommit(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get current commit: %w", err)
	}
	first, err := getFirstCommit(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get first commit: %w", err)
	}

	gitURL, err := getURL(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get remote URL: %w", err)
	}

	if strings.HasPrefix(gitURL, "https://") {
		u, err := url.Parse(gitURL)
		if err != nil {
			return context.GitInfo{}, fmt.Errorf("couldn't parse remote URL: %w", err)
		}
		u.User = nil
		gitURL = u.String()
	}

	var excluding []string
	tag, err := getTag(ctx, excluding)
	if err != nil {
		return context.GitInfo{
			Commit:      full,
			FirstCommit: first,
			CurrentTag:  "v0.0.0",
		}, errors.New("git doesn't contain any tags")
	}

	previous, _ := getPreviousTag(ctx, tag, excluding)

	return context.GitInfo{
		CurrentTag:  tag,
		PreviousTag: previous,
		Commit:      full,
		FirstCommit: first,
	}, nil
}

func validate(ctx *context.Context) error {
	_, err := clean(Exec(ctx, "describe", "--exact-match", "--tags", "--match", ctx.Git.CurrentTag))
	if err != nil {
		return errWrongRef{
			commit: ctx.Git.Commit,
			tag:    ctx.Git.CurrentTag,
		}
	}
	return nil
}

func getFullCommit(ctx *context.Context) (string, error) {
	return clean(Exec(ctx, "show", "--format=%H", "HEAD", "--quiet"))
}

func getFirstCommit(ctx *context.Context) (string, error) {
	return clean(Exec(ctx, "rev-list", "--max-parents=0", "HEAD"))
}

func getTag(ctx *context.Context, excluding []string) (string, error) {
	for _, fn := range []func() ([]string, error){
		func() ([]string, error) {
			return gitTagsPointingAt(ctx, "HEAD")
		},
		func() ([]string, error) {
			return cleanAllLines(gitDescribe(ctx, "HEAD", excluding))
		},
	} {
		tags, err := fn()
		if err != nil {
			return "", err
		}
		if tag := filterOut(tags, excluding); tag != "" {
			return tag, err
		}
	}

	return "", nil
}

func getPreviousTag(ctx *context.Context, current string, excluding []string) (string, error) {
	for _, fn := range []func() ([]string, error){
		func() ([]string, error) {
			sha, err := previousTagSha(ctx, current, excluding)
			if err != nil {
				return nil, err
			}
			return gitTagsPointingAt(ctx, sha)
		},
	} {
		tags, err := fn()
		if err != nil {
			return "", err
		}
		if tag := filterOut(tags, excluding); tag != "" {
			return tag, nil
		}
	}

	return "", nil
}

func gitTagsPointingAt(ctx *context.Context, ref string) ([]string, error) {
	var args []string
	args = append(
		args,
		"tag",
		"--points-at",
		ref,
		"--sort",
		"-version:refname",
	)
	return cleanAllLines(Exec(ctx, args...))
}

func gitDescribe(ctx *context.Context, ref string, excluding []string) (string, error) {
	args := []string{
		"describe",
		"--tags",
		"--abbrev=0",
		ref,
	}
	for _, exclude := range excluding {
		args = append(args, "--exclude="+exclude)
	}
	return clean(Exec(ctx, args...))
}

func previousTagSha(ctx *context.Context, current string, excluding []string) (string, error) {
	tag, err := gitDescribe(ctx, fmt.Sprintf("tags/%s^", current), excluding)
	if err != nil {
		return "", err
	}
	return clean(Exec(ctx, "rev-list", "-n1", tag))
}

func getURL(ctx *context.Context) (string, error) {
	return clean(Exec(ctx, "ls-remote", "--get-url"))
}

func filterOut(tags []string, exclude []string) string {
	if len(exclude) == 0 && len(tags) > 0 {
		return tags[0]
	}
	for _, tag := range tags {
		for _, exl := range exclude {
			if exl != tag {
				return tag
			}
		}
	}
	return ""
}

// CheckSCM returns an error if the given url is not a valid scm url.
func (r Repo) CheckSCM() error {
	if r.isSCM() {
		return nil
	}
	return fmt.Errorf("invalid scm url: %s", r.RawURL)
}

// isSCM returns true if the repo has both an owner and name.
func (r Repo) isSCM() bool {
	return r.Owner != "" && r.Name != ""
}

// isRepo returns true if current folder is a git repository.
func isRepo(ctx *context.Context) bool {
	out, err := Exec(ctx, "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// runWithEnv runs a git command and returns its output or errors.
func runWithEnv(ctx *context.Context, env []string, args ...string) (string, error) {
	extraArgs := []string{
		"-c", "log.showSignature=false",
	}
	args = append(extraArgs, args...)
	/* #nosec */
	cmd := exec.CommandContext(ctx, "git", args...)

	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(cmd.Env, env...)

	err := cmd.Run()

	if err != nil {
		return "", errors.New(stderr.String())
	}

	return stdout.String(), nil
}

// Exec runs a git command and returns its output or errors.
func Exec(ctx *context.Context, args ...string) (string, error) {
	return runWithEnv(ctx, []string{}, args...)
}

// clean the output.
func clean(output string, err error) (string, error) {
	output = strings.ReplaceAll(strings.Split(output, "\n")[0], "'", "")
	if err != nil {
		err = errors.New(strings.TrimSuffix(err.Error(), "\n"))
	}
	return output, err
}

// cleanAllLines returns all the non empty lines of the output, cleaned up.
func cleanAllLines(output string, err error) ([]string, error) {
	var result []string
	for _, line := range strings.Split(output, "\n") {
		l := strings.TrimSpace(strings.ReplaceAll(line, "'", ""))
		if l == "" {
			continue
		}
		result = append(result, l)
	}
	if err != nil {
		err = errors.New(strings.TrimSuffix(err.Error(), "\n"))
	}
	return result, err
}

// errWrongRef happens when the HEAD reference is different from the tag being built.
type errWrongRef struct {
	commit, tag string
}

func (e errWrongRef) Error() string {
	return fmt.Sprintf("git tag %v was not made against commit %v", e.tag, e.commit)
}
