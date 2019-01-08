package memopool

import (
	"reflect"
	"fmt"
	"runtime"
)

func ExampleMemoryPool() {
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
}

func ExampleParallelPool() {
	type ExampleData struct {
		Value [128]byte
		ID int64
	}

	var localCount = runtime.GOMAXPROCS(-1)
	var unitSize = int(reflect.TypeOf(ExampleData{}).Size()) // use unsafe or reflect get data size
	var unitCount = 100 // unit count for each block. Hot data should be in the first block
	var maxBlockCnt = 5 // max block count. If current block over this value, alloc will fail (return nil) .

	pool := NewParallelPools(localCount, unitSize, unitCount, maxBlockCnt, unitCount, maxBlockCnt * 2)

	ptr := pool.Alloc(0)
	if ptr == nil {
		// alloc failed
		fmt.Println("alloc failed")
		return
	}

	dataPtr := (*ExampleData)(ptr)
	dataPtr.ID = 0
	// .... use the data ptr


	pool.Free(0, ptr) // free the unit of pointer 'ptr'
}
