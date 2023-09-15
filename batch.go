package pipelines

import (
	"context"
	"time"
)

// SimpleBatch splits channel stream into chunks
//
// Use carefully it's not for production, is has no timeouts
func SimpleBatch[T any](
	batchSize int,
	channel chan T,
) (batchChan chan []T) {
	batchChan = make(chan []T)
	go func() {
		defer close(batchChan)

		batch := make([]T, 0, batchSize)

		for value := range channel {
			// filling batch
			batch = append(batch, value)
			if len(batch) < batchSize {
				continue
			}

			batchChan <- batch
			batch = make([]T, 0, batchSize)
		}
	}()

	return batchChan
}

// Batch splits channel stream into chunks
//
// It sends batch on batchSize or batchWaitTimeout
func Batch[T any](
	ctx context.Context,
	batchSize int,
	batchWaitTimeout time.Duration,
	channel chan T,
) (batchChan chan []T) {
	batchChan = make(chan []T)

	go func() {
		defer close(batchChan)

		var value T
		batch := make([]T, 0, batchSize)
		timer := time.NewTimer(batchWaitTimeout)

		for {
			select {
			// job finished
			case <-ctx.Done():
				return
			// events wait
			case value = <-channel:
				// filling batch
				batch = append(batch, value)
				if len(batch) < batchSize {
					continue
				}

				// stop timer
				if !timer.Stop() {
					<-timer.C
				}
			// batch wait timeout
			case <-timer.C:
			}

			if len(batch) > 0 {
				batchChan <- batch
			}

			// reset timer
			timer.Reset(batchWaitTimeout)

			// reset batch slice
			batch = make([]T, 0, batchSize)
		}
	}()

	return batchChan
}
