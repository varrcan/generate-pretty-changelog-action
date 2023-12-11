package main

import (
	"embed"
	"fmt"

	"github.com/sethvargo/go-githubactions"

	"github.com/varrcan/generate-pretty-changelog/pkg/config"
	"github.com/varrcan/generate-pretty-changelog/pkg/context"
	"github.com/varrcan/generate-pretty-changelog/pkg/git"
)

//go:embed changelog.yaml
var configFile embed.FS

func main() {
	// forwarding file variable to package
	config.ChangelogFile = configFile

	configPath := githubactions.GetInput("config")
	if configPath == "" {
		configPath = "embed"
	}

	cfg, _ := loadConfig(configPath)
	ctx := context.New(cfg)

	use := githubactions.GetInput("use")
	if use != "" {
		ctx.Config.Changelog.Use = use
	}

	if ctx.Config.Changelog.Use == "github" {
		token := githubactions.GetInput("token")
		if token == "" {
			fmt.Println("token is required for use=github")
			return
		}
		ctx.Token = token
	}

	err := git.Run(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = generate(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(ctx.ReleaseNotes)
	if ctx.ReleaseNotes != "" {
		githubactions.SetOutput("changelog", ctx.ReleaseNotes)
	}
}

func loadConfig(path string) (config.Config, error) {
	p, path, err := loadConfigCheck(path)
	return p, err
}

func loadConfigCheck(path string) (config.Config, string, error) {
	if path == "embed" {
		p, err := config.LoadEmbed()
		return p, path, err
	}
	if path != "" {
		p, err := config.Load(path)
		return p, path, err
	}

	return config.Config{}, "", nil
}
