package awshelpers

import (
	"context"
	"log"
	"math/rand"
	"time"

	"golang.org/x/sync/semaphore"
)

const MaxConcurrentAWS = 3

var concurrentAWS = semaphore.NewWeighted(MaxConcurrentAWS)

type Doer func()

func Ratelimit(ctx context.Context, doer Doer) {
	// Add up to one second of jitter:
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))

	if err := concurrentAWS.Acquire(ctx, 1); err != nil {
		log.Printf("Error: error acquiring semaphore: %v", err)
		return
	}
	defer concurrentAWS.Release(1)
	doer()
}
