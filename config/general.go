package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type GeneralConfig struct {
	// TODO: AccountID string   `yaml:"account_id"`
	Regions []string `json:"regions" yaml:"regions"`
	// TODO: Profile   string   `yaml:"aws_profile"`
	Source string `json:"source" yaml:"-"`
}

func LoadConfig() (*GeneralConfig, error) {
	generalConfig := &GeneralConfig{}
	generalConfig.Source = FindConfig()

	f, err := os.Open(generalConfig.Source)
	if err != nil {
		return nil, err
	}

	if err := yaml.NewDecoder(f).Decode(generalConfig); err != nil {
		return nil, err
	}

	return generalConfig, nil
}

func FindConfig() string {
	env := os.Getenv("CFTOOLRC")
	if env != "" {
		return env
	}

	return "config.yml"
}
