package pipelines_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"github.com/rusinikita/pipelines"
)

func TestBatching(t *testing.T) {
	gen := func(len int) []int {
		testValues := make([]int, 0, len)
		for i := 0; i < len; i++ {
			testValues = append(testValues, rand.Int())
		}

		return testValues
	}

	run := func(testValues []int, batchSize int, batchWaitTimeout time.Duration) chan []int {
		// arrange
		ch := make(chan int)
		ctx, cancel := context.WithCancel(context.TODO())

		// act
		go func() {
			for _, value := range testValues {
				ch <- value
				time.Sleep(time.Microsecond)
			}

			cancel()
		}()

		return pipelines.Batch(ctx, batchSize, batchWaitTimeout, ch)
	}

	t.Run("large timeout", func(t *testing.T) {
		t.Parallel()

		var (
			totalRequests         = 10000
			batchSize             = 10
			batchWaitTimeout      = 10 * time.Second
			expectedBatchRequests = totalRequests / batchSize
		)

		var (
			testValues = gen(totalRequests)
			batchCh    = run(testValues, batchSize, batchWaitTimeout)
		)

		// assert
		calls := 0
		for call := range batchCh {
			assert.Len(t, call, batchSize)

			expected := testValues[calls*batchSize : calls*batchSize+batchSize]
			assert.EqualValues(t, expected, call)

			calls++
		}

		assert.Equal(t, calls, expectedBatchRequests)
	})

	t.Run("with timeout", func(t *testing.T) {
		t.Parallel()

		var (
			totalRequests    = 10000
			batchSize        = 10
			batchWaitTimeout = 12 * time.Microsecond
		)

		batchCh := run(gen(totalRequests), batchSize, batchWaitTimeout)

		// assert
		calls := 0
		for call := range batchCh {
			assert.LessOrEqual(t, len(call), batchSize)

			calls++
		}

		assert.Greater(t, calls, totalRequests/batchSize)
		assert.Less(t, calls, totalRequests/3)
	})
}

type Request struct {
	Param    int
	Callback chan Result
}

type Result struct {
	Value int
	Err   error
}

func BenchmarkRequests(b *testing.B) {
	b.SetParallelism(1000)
	connPool := 1000
	roundTrip := 50 * time.Millisecond

	ch := make(chan Request)

	go func() {
		g, _ := errgroup.WithContext(context.TODO())
		g.SetLimit(connPool)

		for call := range ch {
			call := call
			g.Go(func() error {
				time.Sleep(roundTrip)

				call.Callback <- Result{
					Value: call.Param * 2,
				}

				return nil
			})
		}
	}()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			call := Request{
				Param:    rand.Int(),
				Callback: make(chan Result),
			}

			ch <- call
			result := <-call.Callback

			assert.Equal(b, call.Param*2, result)
		}
	})
}

func BenchmarkPipeline(b *testing.B) {
	b.SetParallelism(1000)
	connPool := 100
	roundTrip := 200 * time.Millisecond

	ch := make(chan Request)
	ctx := context.TODO()
	batchCh := pipelines.Batch(ctx, 50, 50*time.Millisecond, ch)

	go func() {
		g, _ := errgroup.WithContext(ctx)
		g.SetLimit(connPool)

		for calls := range batchCh {
			calls := calls
			g.Go(func() error {
				time.Sleep(roundTrip)

				for _, call := range calls {
					call.Callback <- Result{
						Value: call.Param * 2,
					}
				}

				return nil
			})
		}
	}()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			call := Request{
				Param:    rand.Int(),
				Callback: make(chan Result),
			}

			ch <- call
			result := <-call.Callback

			assert.Equal(b, call.Param*2, result)
		}
	})
}
