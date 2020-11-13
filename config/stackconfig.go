package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/keyneston/cftool/awshelpers"
	"github.com/keyneston/cftool/helpers"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type StackConfig struct {
	Name     string                       `json:"name"   yaml:"name"`
	ARN      string                       `json:"arn"    yaml:"arn"`
	File     string                       `json:"file"   yaml:"file"`
	Params   map[string]string            `json:"params" yaml:"params"`
	Servers  map[string]*ServerCacheEntry `json:"servers" yaml:"servers"`
	Source   string                       `json:"source" yaml:"-"`
	Hydrated bool                         `json:"-" yaml:"-"`

	client    *cf.CloudFormation
	parsedARN arn.ARN
	stackName string

	cacheDir string
	cfRoot   string
	log      *logrus.Logger
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

	region := s.parsedARN.Region
	s.client = awshelpers.GetCloudFormationClient(region)

	return s.client, nil
}

func (s *StackConfig) GetASGClient() (*autoscaling.AutoScaling, error) {
	if err := s.parseARN(); err != nil {
		return nil, err
	}

	region := s.parsedARN.Region
	return awshelpers.GetASGClient(region), nil
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

	if err := s.HydrateServers(); err != nil {
		return err
	}

	s.Hydrated = true

	return nil
}

func (s StackConfig) HydrateServers() error {
	if s.Servers == nil {
		s.Servers = map[string]*ServerCacheEntry{}
	}

	client, err := s.GetClient()
	if err != nil {
		return err
	}

	out, err := client.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: &s.stackName,
	})
	if err != nil {
		return err
	}

	asgs := []string{}

	for _, obj := range out.StackResources {
		if obj.ResourceType == nil {
			continue
		}

		switch *obj.ResourceType {
		case "AWS::AutoScaling::AutoScalingGroup":
			asgs = append(asgs, *obj.PhysicalResourceId)
		default:
			s.log.Debugf("Skipping resource type %v", *obj.ResourceType)
		}
	}

	for _, asg := range asgs {
		res, err := s.getServerCacheEntriesFromASG(context.TODO(), asg)
		if err != nil {
			return err
		}

		for _, r := range res {
			s.Servers[r.ARN] = r
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

func (s *StackConfig) Save(location string) error {
	s.log.Debugf("Saving to %v", location)
	dir := filepath.Dir(location)
	if dir == "" {
		return fmt.Errorf("Invalid directory to save into: %v", dir)
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	f, err := os.OpenFile(location, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
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

	if s.cacheDir == "" {
		log.Fatalf("CacheDir not set: %#v", s)
	}

	return filepath.Clean(path.Join(s.cacheDir, s.parsedARN.Region, s.Name+".yml"))
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
	return filepath.Join(s.cfRoot, s.File)
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

func (s *StackConfig) getServerCacheEntriesFromASG(ctx context.Context, asgNames ...string) ([]*ServerCacheEntry, error) {
	input := &autoscaling.DescribeAutoScalingGroupsInput{}
	for _, asgName := range asgNames {
		input.AutoScalingGroupNames = append(input.AutoScalingGroupNames, &asgName)
	}

	asgClient, err := s.GetASGClient()
	if err != nil {
		return nil, err
	}

	instanceIds := []*string{}

	asgClient.DescribeAutoScalingGroupsPagesWithContext(ctx, input, func(output *autoscaling.DescribeAutoScalingGroupsOutput, lastPage bool) bool {
		for _, asg := range output.AutoScalingGroups {
			for _, instance := range asg.Instances {
				instanceIds = append(instanceIds, instance.InstanceId)
			}
		}
		return true
	})

	instancesInput := ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	}

	servers := []*ServerCacheEntry{}
	awshelpers.GetEC2Client(s.parsedARN.Region).DescribeInstancesPagesWithContext(
		ctx,
		&instancesInput,
		func(output *ec2.DescribeInstancesOutput, lastPage bool) bool {
			for _, resv := range output.Reservations {
				for _, instance := range resv.Instances {
					servers = append(servers, &ServerCacheEntry{
						PrivateIP:  strPointer(instance.PrivateIpAddress),
						PublicIP:   strPointer(instance.PublicIpAddress),
						PublicDNS:  strPointer(instance.PublicDnsName),
						PrivateDNS: strPointer(instance.PrivateDnsName),
						VPCID:      strPointer(instance.VpcId),
						ARN:        strPointer(instance.InstanceId),
					})
				}
			}
			return true
		})

	return servers, nil
}

func strPointer(in *string) string {
	if in == nil {
		return ""
	}

	return *in
}
