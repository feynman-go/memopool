package memopool

import (
	"testing"
	"unsafe"
	"strconv"
)


func TestMemoPoolAlloc(t *testing.T) {
	type TestData struct {
		Name string
		Value int
	}
	mps := New(int(unsafe.Sizeof(TestData{})), 100, 10)
	ptrs := make([]*TestData, 0, 1000)
	for i := 0; i < 1000; i++ {
		p := mps.Alloc()
		if p == nil {
			t.Fatal("can not alloc data for data index:", i)
		}
		ptr := (*TestData)(p)
		ptr.Name = "name-" + strconv.Itoa(i)
		ptr.Value = i
		ptrs = append(ptrs, ptr)
	}

	for i := 0; i < 1000; i++ {
		if ptrs[i].Name != "name-" + strconv.Itoa(i) {
			t.Fatal(
				"check name is invalid for index:",
				i,
				"expect:",
				"name-" + strconv.Itoa(i),
				"but:",
				ptrs[i].Name,
				)
		}
		if ptrs[i].Value != i {
			t.Fatal(
				"check name is invalid for index:",
				i,
				"expect:",
				i,
				"but:",
				ptrs[i].Value,
			)
		}
	}
	p := mps.Alloc()
	if p != nil {
		t.Fatal("alloc over max block size should return nil")
	}
}

func TestMemoPoolFree(t *testing.T) {
	type TestData struct {
		Name string
		Value int
	}
	mps := New(int(unsafe.Sizeof(TestData{})), 100, 10)
	ptrs := make([]*TestData, 0, 1000)
	for j := 0 ; j < 100 ; j++ {
		for i := 0; i < 50; i++ {
			p := mps.Alloc()
			if p == nil {
				t.Fatal("can not alloc data for data index:", i)
			}
			ptr := (*TestData)(p)
			ptr.Name = "name-" + strconv.Itoa(i)
			ptr.Value = i
			ptrs = append(ptrs, ptr)
		}
		for i := 0; i < 50; i++ {
			if !mps.Free(unsafe.Pointer(ptrs[i])) {
				t.Fatal("can not free data for data index:", i)
			}
		}
	}
}
