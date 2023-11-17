package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Queries []Query `yaml:"queries"`
}

type Query struct {
	// GitHubArg is the argument passed to GH.
	GitHubArg string `yaml:"github_arg"`
	// QueryName is a long of the query that shows in the description.
	QueryName string `yaml:"query_name"`
	//  ShortName is a single letter identifier of the query.
	ShortName string `yaml:"short_name"`
}

const DefaultConfigYaml = `
queries:
  - github_arg: "--assignee=@me"
    query_name: "Assignee"
    short_name: "a"
  - github_arg: "--author=@me"
    query_name: "Author"
    short_name: "*"
  - github_arg: "--mentions=@me"
    query_name: "Mentions"
    short_name: "m"
  - github_arg: "--review-requested=@me"
    query_name: "ReviewRequested"
    short_name: "r"
`

func GetDefaultConfig() Config {
	b := []byte(strings.TrimSpace(DefaultConfigYaml))
	c, err := unmarshallConfig(b)
	if err != nil {
		panic("default config unmarshallable")
	}
	return c
}

func GetConfigFromFile(filename string) (Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config file: %s", err)
	}
	return unmarshallConfig(content)
}

func unmarshallConfig(content []byte) (Config, error) {
	var config Config
	err := yaml.Unmarshal(content, &config)
	if err != nil {
		return config, fmt.Errorf("error unmarshalling YAML: %s", err)
	}
	return config, nil
}
