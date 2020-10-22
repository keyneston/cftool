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

type StackConfig struct {
	Name   string                 `json:"name"   yaml:"name"`
	ARN    string                 `json:"arn"    yaml:"arn"`
	File   string                 `json:"file"   yaml:"file"`
	Params map[string]interface{} `json:"params" yaml:"params"`

	client    *cf.CloudFormation
	parsedARN arn.ARN
	stackName string
}

func (s *StackConfig) GetClient() (*cf.CloudFormation, error) {
	if s.client != nil {
		return s.client, nil
	}

	stackARN, err := arn.Parse(s.ARN)
	if err != nil {
		return nil, fmt.Errorf("Error parsing ARN: %q", err)
	}
	s.parsedARN = stackARN

	splitStr := strings.SplitN(stackARN.Resource, "/", 4)
	if len(splitStr) != 3 {
		return nil, fmt.Errorf("ARN resources doesn't match expected: %q", err)
	}
	s.stackName = splitStr[1]

	client, err := AWSClient(stackARN.Region)
	if err != nil {
		return nil, err
	}

	s.client = client
	return s.client, nil
}

func AWSClient(region string) (*cf.CloudFormation, error) {
	if region == "" {
		region = "us-east-1"
	}

	sess := session.Must(session.NewSession())
	config := aws.NewConfig().WithRegion(region) // TODO: fix this

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

func (s *StackConfig) GetTemplate() (string, error) {
	client, err := s.GetClient()
	if err != nil {
		return "", err
	}

	template, err := client.GetTemplate(&cf.GetTemplateInput{
		StackName:     &s.stackName,
		TemplateStage: aws.String("Original"),
	})
	if err != nil {
		return "", fmt.Errorf("GetTemplate: %v", err)
	}

	if template.TemplateBody != nil {
		return *template.TemplateBody, nil
	} else {
		return "", fmt.Errorf("no template found")
	}
}

func (s *StackConfig) GetLive() (*cf.DescribeStacksOutput, error) {
	client, err := s.GetClient()
	if err != nil {
		return nil, err
	}

	input := &cf.DescribeStacksInput{
		StackName: &s.stackName,
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
