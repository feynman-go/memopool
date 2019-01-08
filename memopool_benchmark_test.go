package memopool

import (
	"testing"
	"unsafe"
	"sync"
	"runtime"
	"time"
	"sync/atomic"
	"math/rand"
)

func BenchmarkStandardGC_AllocFree(b *testing.B) {
	type TestData struct {
		Name string
		Value [128]int
	}
	b.ResetTimer()
	for i := 0 ; i < b.N; i ++{
		var ptr unsafe.Pointer
		for j := 0 ; j < 100 ; j ++ {
			ptr = unsafe.Pointer(new(TestData))
			if ptr == nil {
				b.Fatal("alloc return empty ptr for index:", j)
			}
		}
	}
}

func BenchmarkSyncPool_BatchAllocFree(b *testing.B) {
	type TestData struct {
		Name string
		Value int
	}
	mps := sync.Pool{
		New: func() interface{} {
			return new(TestData)
		},
	}
	ps := make([]*TestData, 0, 1000)
	b.ResetTimer()
	for i := 0 ; i < b.N; i ++{
		ps = ps[:0]
		var ptr *TestData
		for j := 0 ; j < 50 ; j ++ {
			ptr = mps.Get().(*TestData)
			ptr.Value = 0
			ptr.Name = ""
			ps = append(ps, ptr)
		}
		for j := 0 ; j < 50 ; j ++ {
			mps.Put(ps[j])
		}
	}
}

func BenchmarkMemoPool_BatchAllocFree(b *testing.B) {
	type TestData struct {
		Name string
		Value int
	}
	mps := New(int(unsafe.Sizeof(TestData{})), 100, 10)
	ls := make([]unsafe.Pointer, 0, 100)
	b.ResetTimer()
	for i := 0 ; i < b.N; i ++{
		ls = ls[:0]
		var ptr unsafe.Pointer
		for j := 0 ; j < 100 ; j ++ {
			ptr = mps.Alloc()
			if ptr == nil {
				b.Fatal("alloc return empty ptr for index:", j)
			}
			p := (*TestData)(ptr)
			p.Value = 0
			p.Name = ""
			ls = append(ls, ptr)
		}
		for j := 0 ; j < 100 ; j ++ {
			if !mps.Free(ls[j]) {
				b.Fatal("free return false for index:", j)
			}
		}
	}
}

func BenchmarkStandardGC_RandomAllocFree(b *testing.B) {
	type TestData struct {
		Name string
		Value [128]int
	}

	var factors = make([]int, 0, 10000)
	for i := 0; i < 10000; i ++ {
		factors = append(factors, rand.Int())
	}

	ps := make([]*TestData, 0, 1000)
	b.ResetTimer()
	var ptr *TestData
	var last int
	for i := 0 ; i < b.N; i ++{
		switch factors[i % 10000] % 3 {
		case 0:
			ptr = new(TestData)
			ps = append(ps, ptr)
		case 1:
			if last = len(ps) - 1; last > -1 {
				ps = ps[:last]
			}
		case 2:
			if last = len(ps) - 1; last > -1 {
				ps = ps[:last]
			}
			ptr = new(TestData)
			ps = append(ps, (*TestData)(ptr))
		}
	}
}

func BenchmarkSyncPool_RandomAllocFree(b *testing.B) {
	type TestData struct {
		Name string
		Value [128]int
	}

	var factors = make([]int, 0, 10000)
	for i := 0; i < 10000; i ++ {
		factors = append(factors, rand.Int())
	}

	mps := sync.Pool{
		New: func() interface{} {
			return new(TestData)
		},
	}
	ps := make([]*TestData, 0, 1000)
	b.ResetTimer()
	var ptr *TestData
	var last int
	for i := 0 ; i < b.N; i ++{
		switch factors[i % 10000] % 3 {
		case 0:
			ptr = mps.Get().(*TestData)
			ps = append(ps, ptr)
		case 1:
			if last = len(ps) - 1; last > -1 {
				mps.Put(ps[last])
				ps = ps[:last]
			}
		case 2:
			if last = len(ps) - 1; last > -1 {
				mps.Put(ps[last])
				ps = ps[:last]
			}
			ptr = mps.Get().(*TestData)
			ps = append(ps, ptr)
		}
	}
}

func BenchmarkMemoPool_RandomAllocFree(b *testing.B) {
	type TestData struct {
		Name string
		Value [128]int
	}

	var factors = make([]int, 0, 10000)
	for i := 0; i < 10000; i ++ {
		factors = append(factors, rand.Int())
	}

	mps := New(int(unsafe.Sizeof(TestData{})), 100, -1)
	ps := make([]*TestData, 0, 1000)
	b.ResetTimer()
	var ptr unsafe.Pointer
	var last int
	for i := 0 ; i < b.N; i ++{
		switch factors[i % 10000] % 3 {
		case 0:
			ptr = mps.Alloc()
			ps = append(ps, (*TestData)(unsafe.Pointer(ptr)))
		case 1:
			if last = len(ps) - 1; last > -1 {
				mps.Free(unsafe.Pointer(ps[last]))
				ps = ps[:last]
			}
		case 2:
			if last = len(ps) - 1; last > -1 {
				mps.Free(unsafe.Pointer(ps[last]))
				ps = ps[:last]
			}
			ptr = mps.Alloc()
			ps = append(ps, (*TestData)(unsafe.Pointer(ptr)))
		}
	}
}

func BenchmarkSyncPools_ParallelAllocFree(b *testing.B) {
	type TestData struct {
		Name string
		Value [1024]int
	}

	localCount := runtime.GOMAXPROCS(-1)
	mps := sync.Pool{
		New: func() interface{} {
			return new(TestData)
		},
	}
	b.SetParallelism(1)
	ps := make([][]*TestData, localCount)
	for i := 0 ; i < localCount ; i ++ {
		ps[i] = make([]*TestData, 0, 100)
	}

	var factors = make([]int, 0, 10000)
	for i := 0; i < 10000; i ++ {
		factors = append(factors, rand.Int())
	}

	var local int32 = -1

	time.Sleep(1 * time.Second)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		lc := atomic.AddInt32(&local, 1)
		ls := ps[lc][:0]
		var allocCount = 0
		var p *TestData
		i := 0
		for pb.Next() {
			if factors[i % 1000] % 5 == 0 || allocCount > 200 {
				if allocCount > 0 {
					mps.Put(ls[len(ls) - 1])
					ls = ls[:len(ls) - 1]
					allocCount--
				}

			} else {
				p = mps.Get().(*TestData)
				if p == nil {
					b.Fatal("fatal alloc for allocCount:", allocCount + 1)
				}
				ls = append(ls, p)
				allocCount ++
			}
			i++
		}
	})
}

func BenchmarkMemoryPool_ParallelAllocFree(b *testing.B) {
	type TestData struct {
		Name string
		Value [1024]int
	}

	localCount := runtime.GOMAXPROCS(-1)
	mps := NewParallelPools(localCount, int(unsafe.Sizeof(TestData{})), 100, 100, 10, 10)
	b.SetParallelism(1)
	ps := make([][]unsafe.Pointer, localCount)
	for i := 0 ; i < localCount ; i ++ {
		ps[i] = make([]unsafe.Pointer, 0, 100)
	}

	var factors = make([]int, 0, 10000)
	for i := 0; i < 10000; i ++ {
		factors = append(factors, rand.Int())
	}

	var local int32 = -1

	time.Sleep(1 * time.Second)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		lc := atomic.AddInt32(&local, 1)
		ls := ps[lc][:0]
		var allocCount = 0
		var p unsafe.Pointer
		i := 0
		for pb.Next() {
			if factors[i % 1000] % 5 == 0 || allocCount > 200 {
				if allocCount > 0 {
					if !mps.Free(int(lc), ls[len(ls) - 1]) {
						b.Fatal("free err", ls[len(ls) - 1], i, allocCount)
					}
					ls = ls[:len(ls) - 1]
					allocCount--
				}

			} else {
				p = mps.Alloc(int(lc))
				if p == nil {
					b.Fatal("fatal alloc for allocCount:", allocCount + 1)
				}
				ls = append(ls, p)
				allocCount ++
			}
			i++
		}
	})
}