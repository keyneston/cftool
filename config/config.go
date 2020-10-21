package config

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"gopkg.in/yaml.v2"
)

type StackSet map[string]*StackConfig

type GeneralConfig struct {
	AccountID string   `yaml:"account_id"`
	Regions   []string `yaml:"regions"`
	Profile   string   `yaml:"aws_profile"`
}

type StackConfig struct {
	Name   string                 `yaml:"name"`
	ARN    string                 `yaml:"arn"`
	File   string                 `yaml:"file"`
	Params map[string]interface{} `yaml:"params"`
}

func AWSClient() (*cf.CloudFormation, error) {
	sess := session.Must(session.NewSession())
	config := aws.NewConfig() // TODO: fix this

	srv := cf.New(sess, config)
	return srv, nil
}

func LoadStacksFromWD() (map[string]*StackConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting cwd: %#v", err)
		return nil, err
	}

	return LoadStacks(cwd)
}

func LoadStacks(root string) (map[string]*StackConfig, error) {
	stacks := map[string]*StackConfig{}

	filesToParse := []string{}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		ext := filepath.Ext(path)
		if ext == ".yml" || ext == ".yaml" {
			filesToParse = append(filesToParse, path)
		}
		return nil
	})

	for _, path := range filesToParse {
		stack, err := LoadStackFromFile(path)
		switch err {
		case nil:
			break
		case io.EOF:
			log.Printf("Warning: file %q empty", path)
			continue
		default:
			return nil, err // TODO: collect errors here and return as a batch
		}

		stacks[stack.Name] = stack
	}

	return stacks, nil
}

func (s *StackConfig) GetLive(client *cf.CloudFormation) (*cf.DescribeStacksOutput, error) {
	stackARN, err := arn.Parse(s.ARN)
	if err != nil {
		return nil, fmt.Errorf("Error parsing ARN: %q", err)
	}

	// FIXME: This is not thread safe
	// log.Printf("Setting region: %v => %v", *client.Client.Config.Region, stackARN.Region)
	client.Client.Config.WithRegion(stackARN.Region)
	// log.Printf("Set region: %v", *client.Client.Config.Region)

	splitStr := strings.SplitN(stackARN.Resource, "/", 4)
	if len(splitStr) != 3 {
		return nil, fmt.Errorf("ARN resources doesn't match expected: %q", err)
	}

	input := &cf.DescribeStacksInput{
		StackName: &splitStr[1],
	}

	out, err := client.DescribeStacks(input)
	if err != nil {
		return nil, fmt.Errorf("Error fetching stack [%q]: %q", s.Name, err)
	}

	return out, nil
}

func (s StackConfig) Region() (string, error) {
	stackARN, err := arn.Parse(s.ARN)
	if err != nil {
		return "", fmt.Errorf("Error parsing ARN: %q", err)
	}

	return stackARN.Region, nil
}

func LoadStackFromFile(file string) (*StackConfig, error) {
	stack := &StackConfig{}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&stack); err != nil {
		return nil, err
	}

	return stack, nil
}
