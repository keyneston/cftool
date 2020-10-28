package config

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type GeneralConfig struct {
	// TODO: AccountID string   `yaml:"account_id"`
	// TODO: Profile   string   `yaml:"aws_profile"`

	Regions []string `json:"regions" yaml:"regions"`
	Source  string   `json:"source" yaml:"-"`

	CloudFormationRoot string `json:"cloud_formation_root" yaml:"cloud_formation_root"`
	StacksDir          string `json:"stacks_dir" yaml:"stacks_dir"`
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

	if generalConfig.CloudFormationRoot == "" {
		return nil, fmt.Errorf("`cloud_formation_root` is empty")
	}
	if generalConfig.CloudFormationRoot == "" {
		return nil, fmt.Errorf("`stacks_dir` is empty")
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

func (g *GeneralConfig) LoadStacks() (*StacksDB, error) {
	db := &StacksDB{}

	root := g.StacksDir
	if root == "" {
		return nil, fmt.Errorf("No `stacks_dir` configured in %q", FindConfig())
	}

	filesToParse := []string{}
	config := FindConfig()
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Don't re-parse the config file
		if filepath.Base(path) == config {
			return nil
		}

		ext := filepath.Ext(path)
		if ext == ".yml" || ext == ".yaml" {
			filesToParse = append(filesToParse, path)
		}
		return nil
	})

	for _, path := range filesToParse {
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		stack, err := LoadStackFromFile(path)
		switch err {
		case nil:
			break
		case io.EOF:
			log.Printf("Warning: file %q empty", relativePath)
			continue
		default:
			return nil, err // TODO: collect errors here and return as a batch
		}

		stack.stacksDir = g.StacksDir
		stack.cfRoot = g.CloudFormationRoot

		db.AddStack(stack)
	}

	return db, nil
}
