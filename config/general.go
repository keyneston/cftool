package config

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/keyneston/cftool/helpers"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type GeneralConfig struct {
	// TODO: AccountID string   `yaml:"account_id"`
	// TODO: Profile   string   `yaml:"aws_profile"`

	Regions []string `json:"regions" yaml:"regions"`
	Source  string   `json:"source" yaml:"-"`

	CloudFormationRoot string `json:"cloud_formation_root" yaml:"cloud_formation_root"`
	CacheDir           string `json:"cache" yaml:"cache"`

	LogLevel logrus.Level   `json:"log_level" yaml:"log_level"`
	Log      *logrus.Logger `json:"-" yaml:"-"`
}

func LoadConfig() (*GeneralConfig, error) {
	generalConfig := &GeneralConfig{
		Log: logrus.New(),
	}
	generalConfig.Source = FindConfig()

	f, err := os.Open(helpers.Expand(generalConfig.Source))
	if err != nil {
		return nil, err
	}

	if err := yaml.NewDecoder(f).Decode(generalConfig); err != nil {
		return nil, err
	}

	if generalConfig.CloudFormationRoot == "" {
		return nil, fmt.Errorf("`cloud_formation_root` is empty")
	}
	if generalConfig.CacheDir == "" {
		generalConfig.CacheDir = helpers.Expand(DefaultCacheDir)
	}

	generalConfig.CloudFormationRoot, err = homedir.Expand(generalConfig.CloudFormationRoot)
	if err != nil {
		return nil, err
	}

	generalConfig.CacheDir, err = homedir.Expand(generalConfig.CacheDir)
	if err != nil {
		return nil, err
	}

	generalConfig.Log.SetLevel(generalConfig.LogLevel)

	return generalConfig, nil
}

func (g *GeneralConfig) SetLevel(level logrus.Level) {
	g.LogLevel = level
	g.Log.SetLevel(level)
}

func (g *GeneralConfig) LoadStacks() (*StacksDB, error) {
	db := &StacksDB{}

	root := g.CacheDir

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
		stack, err := g.LoadStackFromFile(path)
		switch err {
		case nil:
			break
		case io.EOF:
			log.Printf("Warning: file %q empty", relativePath)
			continue
		default:
			return nil, err // TODO: collect errors here and return as a batch
		}

		db.AddStack(stack)
	}

	return db, nil
}

func (g GeneralConfig) NewStack(name, arn string) *StackConfig {
	return &StackConfig{
		Name: name,
		ARN:  arn,

		cacheDir: g.CacheDir,
		cfRoot:   g.CloudFormationRoot,
	}
}

func (g GeneralConfig) LoadStackFromFile(file string) (*StackConfig, error) {
	stack := &StackConfig{}
	stack.Source = file

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&stack); err != nil {
		return nil, err
	}

	stack.cacheDir = g.CacheDir
	stack.cfRoot = g.CloudFormationRoot

	return stack, nil
}
