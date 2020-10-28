package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/keyneston/cftool/awshelpers"
	"github.com/keyneston/cftool/helpers"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

type StackConfig struct {
	Name   string            `json:"name"   yaml:"name"`
	ARN    string            `json:"arn"    yaml:"arn"`
	File   string            `json:"file"   yaml:"file"`
	Params map[string]string `json:"params" yaml:"params"`
	Source string            `json:"source" yaml:"-"`

	client    *cf.CloudFormation
	parsedARN arn.ARN
	stackName string

	stacksDir string
	cfRoot    string
}

func (s *StackConfig) parseARN() error {
	// Only do this once
	if s.stackName != "" {
		return nil
	}

	stackARN, err := arn.Parse(s.ARN)
	if err != nil {
		return fmt.Errorf("Error parsing ARN: %q", err)
	}
	s.parsedARN = stackARN

	splitStr := strings.SplitN(stackARN.Resource, "/", 4)
	if len(splitStr) != 3 {
		return fmt.Errorf("ARN resources doesn't match expected: %q", err)
	}
	s.stackName = splitStr[1]

	return nil
}

func (s *StackConfig) GetClient() (*cf.CloudFormation, error) {
	if err := s.parseARN(); err != nil {
		return nil, err
	}

	if s.client != nil {
		return s.client, nil
	}

	client, err := AWSClient(s.parsedARN.Region)
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

	srv := awshelpers.GetClient(region)
	return srv, nil
}

func (s *StackConfig) GetLiveTemplate() (string, error) {
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

func (s *StackConfig) Hydrate() error {
	live, err := s.GetLive()
	if err != nil {
		return err
	}

	if len(live.Stacks) == 0 {
		return nil
	}
	cur := live.Stacks[0]

	if len(cur.Parameters) > 0 && s.Params == nil {
		s.Params = map[string]string{}
	}

	for _, pair := range cur.Parameters {
		if pair.ParameterKey != nil && pair.ParameterValue != nil {
			s.Params[*pair.ParameterKey] = *pair.ParameterValue
		}
	}

	return nil
}

func (s StackConfig) Region() (string, error) {
	if err := s.parseARN(); err != nil {
		return "", err
	}

	return s.parsedARN.Region, nil
}

func LoadStackFromFile(file string) (*StackConfig, error) {
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

	return stack, nil
}

func (s *StackConfig) Save(location string) error {
	f, err := os.OpenFile(location, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o655)
	if err != nil {
		return err
	}

	if err := yaml.NewEncoder(f).Encode(s); err != nil {
		return err
	}

	return nil
}

func (s StackConfig) Location() string {
	if s.Source != "" {
		return s.Source
	}

	return filepath.Clean(path.Join("examples", s.Name+".yml"))
}

func (s StackConfig) GetLiveTemplateHash() (string, error) {
	template, err := s.GetLiveTemplate()
	if err != nil {
		return "", err
	}
	liveTemplateHash := helpers.HashString(template)

	return liveTemplateHash, nil
}

func (s StackConfig) GetDiskTemplateLocation() string {
	long := filepath.Join(s.cfRoot, s.File)

	expanded, err := homedir.Expand(long)
	if err == nil {
		// If we can expand ~ without error return it
		return expanded
	}

	// otherwise just return what we have.
	return long
}

func (s StackConfig) GetDiskTemplateHash() (string, error) {
	return helpers.HashFile(s.GetDiskTemplateLocation())
}

func (s StackConfig) GetDiskTemplate() (string, error) {
	f, err := os.Open(s.GetDiskTemplateLocation())
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *StackConfig) StackName() string {
	if err := s.parseARN(); err != nil {
		return ""
	}

	return s.stackName
}

func (s *StackConfig) AWSParams() []*cloudformation.Parameter {
	awsParams := []*cloudformation.Parameter{}

	for k, v := range s.Params {
		awsParams = append(awsParams, &cloudformation.Parameter{
			// Use aws.String to clone and then take a pointer to the clone:
			ParameterKey:   aws.String(k),
			ParameterValue: aws.String(v),
		})
	}

	return awsParams
}
