package awshelpers

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	cfClientCache  = &sync.Map{}
	asgClientCache = &sync.Map{}
	ec2ClientCache = &sync.Map{}
	sessionCache   = &sync.Map{}
)

func GetSession(region string) *session.Session {
	if region == "" {
		region = "us-east-1"
	}

	c, ok := sessionCache.Load(region)
	if ok {
		return c.(*session.Session)
	}

	sess := session.Must(session.NewSession())

	c, _ = sessionCache.LoadOrStore(region, sess)
	return c.(*session.Session)
}

func config(region string) *aws.Config {
	return request.WithRetryer(aws.NewConfig().WithRegion(region), client.DefaultRetryer{
		NumMaxRetries:    10,
		MinRetryDelay:    5 * time.Millisecond,
		MinThrottleDelay: 20 * time.Millisecond,
		MaxRetryDelay:    5 * time.Second,
		MaxThrottleDelay: 5 * time.Second,
	})
}

func GetCloudFormationClient(region string) *cloudformation.CloudFormation {
	return cf.New(GetSession(region), config(region))
}

func GetASGClient(region string) *autoscaling.AutoScaling {
	return autoscaling.New(GetSession(region), config(region))
}

func GetEC2Client(region string) *ec2.EC2 {
	return ec2.New(GetSession(region), config(region))
}
