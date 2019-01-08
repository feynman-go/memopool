# memopool

Memopool is a golang memory pool package which is a little faster than sync.Pool in ideal condition.

## Requirement

- go (>= 1.8)

## Installation

```shell
go get github.com/feynman-go/memopool
```

## Example

```go

    type ExampleData struct {
		Value [128]byte
		ID int64
	}

	var unitSize = int(reflect.TypeOf(ExampleData{}).Size()) // use unsafe or reflect get data size
	var unitCount = 100 // unit count for each block. Hot data should be in the first block
	var maxBlockCnt = 5 // max block count. If current block over this value, alloc will fail (return nil) .

	pool := New(unitSize, unitCount, maxBlockCnt)
	ptr := pool.Alloc()
	if ptr == nil {
		// alloc failed
		fmt.Println("alloc failed")
		return
	}

	dataPtr := (*ExampleData)(ptr)
	dataPtr.ID = 0
	// .... use the data ptr


	pool.Free(ptr) // free the unit of pointer 'ptr'

```

## Benchmark
[memopool](https://github.com/feynman-go/memopool) vs [sync.Pool](https://github.com/golang/go/tree/master/src/sync)(Standard library)

```
goos: darwin
goarch: amd64
pkg: github.com/feynman-go/memopool
BenchmarkStandardGC_Alloc-4                     100000             18251 ns/op           115200 B/op        100 allocs/op
BenchmarkSyncPool_BatchAllocFree-4              500000             2247 ns/op            0 B/op          0 allocs/op
BenchmarkMemoPool_BatchAllocFree-4              1000000            1993 ns/op            0 B/op          0 allocs/op
BenchmarkStandardGC_RandomAllocFree-4           10000000           140 ns/op             767 B/op          0 allocs/op
BenchmarkSyncPool_RandomAllocFree-4             50000000           33.7 ns/op            9 B/op          0 allocs/op
BenchmarkMemoPool_RandomAllocFree-4             50000000           23.2 ns/op            0 B/op          0 allocs/op
BenchmarkSyncPools_ParallelAllocFree-4          200000000          7.57 ns/op            0 B/op          0 allocs/op
BenchmarkMemoryPool_ParallelAllocFree-4         200000000          8.60 ns/op            0 B/op          0 allocs/op
PASS
ok      github.com/feynman-go/memopool  26.536s

```

## Author
[hlts2](https://github.com/feynman-go)

## LICENSE
memopool released under MIT license.