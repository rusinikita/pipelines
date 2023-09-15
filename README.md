Generic Golang pipeline functions

## Introduction

Channels and goroutines gives Golang advantage over many languages.

Materials to read:
- [Concurrency is not parallelism](https://go.dev/blog/waza-talk)
- [Go pipelines](https://go.dev/blog/pipelines)
- [Redis pipelining](https://redis.io/docs/manual/pipelining/)

## Batch

Collect many parallel connections into fewer request chunks. Optimize reads and writes by key.

| Before                          | Now                                                     |
|---------------------------------|---------------------------------------------------------|
| 1M users creates 1M DB requests | 1M users creates ~100K DB requests by ~10 keys in batch |

### Usage

```go
// Make request structure with callback
type Request struct {
	Param    int
	Callback chan Result
}

type Result struct {
	Value int
	Err   error
}
```

```go
// Run batch
batchCh := Batch(ctx, 50, 50*time.Millisecond, ch)
```

```go
// Run batch handler goroutine. Call callbacks for all batch
for _, call := range calls {
    call.Callback <- Result{
        Value: call.Param * 2,
    }
}
```

```go
// Make single request calls via channel
call := Request{
    Param:    rand.Int(),
    Callback: make(chan Result),
}

ch <- call
result := <-call.Callback
```
