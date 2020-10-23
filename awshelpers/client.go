package awshelpers

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
)

var clientCache = &sync.Map{}

func GetClient(region string) *cloudformation.CloudFormation {
	c, ok := clientCache.Load(region)
	if ok {
		return c.(*cloudformation.CloudFormation)
	}

	sess := session.Must(session.NewSession())
	config := request.WithRetryer(aws.NewConfig().WithRegion(region), client.DefaultRetryer{
		NumMaxRetries:    10,
		MinRetryDelay:    5 * time.Millisecond,
		MinThrottleDelay: 20 * time.Millisecond,
		MaxRetryDelay:    5 * time.Second,
		MaxThrottleDelay: 5 * time.Second,
	})
	c = cf.New(sess, config)

	c, _ = clientCache.LoadOrStore(region, c)
	return c.(*cloudformation.CloudFormation)
}
