package context

import (
	stdctx "context"
	"os"
	"strings"

	"github.com/varrcan/generate-pretty-changelog/pkg/config"
)

// GitInfo includes tags and diffs used in some point.
type GitInfo struct {
	CurrentTag  string
	PreviousTag string
	Commit      string
	FirstCommit string
}

// env is the environment variables.
type env map[string]string

// Context carries along some data through the pipes.
type Context struct {
	stdctx.Context
	Config       config.Config
	Env          env
	Token        string
	Git          GitInfo
	ReleaseNotes string
	Version      string
}

// New context.
func New(config config.Config) *Context {
	return wrap(stdctx.Background(), config)
}

// wrap wraps an existing context.
func wrap(ctx stdctx.Context, config config.Config) *Context {
	return &Context{
		Context: ctx,
		Config:  config,
		Env:     toEnv(append(os.Environ(), config.Env...)),
	}
}

// toEnv converts a list of strings to an env (aka a map[string]string).
func toEnv(environ []string) env {
	r := env{}
	for _, e := range environ {
		k, v, ok := strings.Cut(e, "=")
		if !ok || k == "" {
			continue
		}
		r[k] = v
	}
	return r
}
