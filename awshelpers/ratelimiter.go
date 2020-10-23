package awshelpers

import (
	"context"
	"log"
	"sync"

	"golang.org/x/sync/semaphore"
)

const MaxConcurrentAWS = 3

var semaphores = &sync.Map{}

type Doer func()

func Ratelimit(ctx context.Context, region string, doer Doer) {
	untypedSemp, _ := semaphores.LoadOrStore(region, semaphore.NewWeighted(MaxConcurrentAWS))
	semp := untypedSemp.(*semaphore.Weighted)

	if err := semp.Acquire(ctx, 1); err != nil {
		log.Printf("Error: error acquiring semaphore: %v", err)
		return
	}
	defer semp.Release(1)
	doer()
}
