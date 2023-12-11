package config

import (
	"embed"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// ChangelogFile embed config file.
var ChangelogFile embed.FS

// filters config.
type filters struct {
	Include []string `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// changelog Config.
type changelog struct {
	Filters filters          `yaml:"filters,omitempty" json:"filters,omitempty"`
	Sort    string           `yaml:"sort,omitempty" json:"sort,omitempty" jsonschema:"enum=asc,enum=desc,enum=,default="`
	Use     string           `yaml:"use,omitempty" json:"use,omitempty" jsonschema:"enum=provider,enum=github,enum=github-native,enum=gitlab,default=provider"`
	Groups  []changelogGroup `yaml:"groups,omitempty" json:"groups,omitempty"`
	Abbrev  int              `yaml:"abbrev,omitempty" json:"abbrev,omitempty"`
}

// changelogGroup holds the grouping criteria for the changelog.
type changelogGroup struct {
	Title  string `yaml:"title,omitempty" json:"title,omitempty"`
	Regexp string `yaml:"regexp,omitempty" json:"regexp,omitempty"`
	Order  int    `yaml:"order,omitempty" json:"order,omitempty"`
}

// Config includes all configuration.
type Config struct {
	Env       []string  `yaml:"env,omitempty" json:"env,omitempty"`
	Changelog changelog `yaml:"changelog,omitempty" json:"changelog,omitempty"`
}

// Load config file.
func Load(file string) (config Config, err error) {
	f, err := os.Open(file) // #nosec
	if err != nil {
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			return
		}
	}(f)
	return loadReader(f)
}

// loadReader config via io.Reader.
func loadReader(fd io.Reader) (config Config, err error) {
	data, err := io.ReadAll(fd)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(data, &config)
	return config, err
}

// LoadEmbed config via embed.FS
func LoadEmbed() (config Config, err error) {
	data, err := ChangelogFile.ReadFile("changelog.yaml")
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(data, &config)
	return config, err
}
