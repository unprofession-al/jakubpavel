package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Checks map[string]CheckConfig `json:"checks" yaml:"checks"`
}

type CheckConfig struct {
	Resolver        string       `yaml:"resolver" json:"resolver"`
	ResolverTimeout string       `yaml:"resolver_timeout" json:"resolver_timeout"`
	Resolve         string       `yaml:"resolve" json:"resolve"`
	Expect          ExpectConfig `yaml:"expect" json:"expect"`
}

type ExpectConfig struct {
	AnswerSection     []string `yaml:"answer_section" json:"answer_section"`
	AuthoritySection  []string `yaml:"authority_section" json:"authority_section"`
	AdditionalSection []string `yaml:"additional_section" json:"additional_section"`
}

func NewConfig(path string) (*Config, error) {
	config := Config{}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return &config, err
	}

	err = yaml.Unmarshal(data, &config)
	return &config, err
}
