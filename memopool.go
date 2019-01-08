package memopool

import (
	"unsafe"
	"log"
	"reflect"
	"sync"
)

/**
	A package for memory pool. More info can read 'https://en.wikipedia.org/wiki/Memory_pool'
*/

var intSize = unsafe.Sizeof(int(0))

/** MemoPool is a base struct for memory pool.
	A memory pool is composed with multi of memoBlocks as a double linked list.
	For Best performance, hot data should in the top (first) memory block.
 */
type MemoPool struct {
	full    memoBlock // empty block used as full list head
	partial memoBlock // empty block used as partial list head
	empty   memoBlock // empty block used as empty list head

	blockCnt int // current block count
	maxBlockCnt int // max block count. If is negative then no max block limit.

	nUintSize uintptr // unit byte count
	nUnitCount uintptr // unit count per block
}

func New(nUintSize int, nUnitCount int, maxBlockCnt int) *MemoPool {
	ret := &MemoPool{
		full:        *newMemoPool(0, 0 ),
		partial:     *newMemoPool(0, 0 ),
		empty:       *newMemoPool(0, 0 ),
		blockCnt:    0,
		maxBlockCnt: maxBlockCnt,
		nUintSize:   uintptr(nUintSize),
		nUnitCount:  uintptr(nUnitCount),
	}

	moveAfter(&ret.partial, newMemoPool(ret.nUintSize, ret.nUnitCount))
	ret.blockCnt ++
	return ret
}

// Alloc a unit memory and return nil if can not alloc more.
func (mp *MemoPool) Alloc() unsafe.Pointer {
	if mp.partial.Next == nil {
		// check empty is empty
		if mp.empty.Next == nil {
			if mp.blockCnt >= mp.maxBlockCnt && mp.maxBlockCnt > 0 {
				// over max block
				log.Println("over max block count")
				return nil
			}
			// move to empty first
			moveAfter(&mp.empty, newMemoPool(mp.nUintSize, mp.nUnitCount))
			mp.blockCnt++
		}

		//move memory pool from empty to partial,
		moveAfter(&mp.partial, mp.empty.Next)
	}

	partial := mp.partial.Next
	var ptr unsafe.Pointer
	ptr = partial.alloc()

	if partial.nFree <= 0 {
		// move partial to full list
		moveAfter(&mp.full, partial)
	}
	return ptr
}

// Free a unit memory that starts with point 'pt'. Return false if can not free. Do not free same id continuously
func (mp *MemoPool) Free(pt unsafe.Pointer) bool {
	var partition = mp.partial.Next
	var full = mp.full.Next

	var ok bool

	// inspired search partial first
	if partition != nil {
		if partition, ok = mp.searchAndFree(partition, pt); ok {
			return true
		}
		// inspired search partial first two
		for i := 0 ; i < 2 && partition != nil; i++ {
			if partition, ok = mp.searchAndFree(partition, pt); ok {
				break
			}
		}
	}

	if !ok {
		for partition != nil || full != nil {
			if partition != nil {
				partition, ok = mp.searchAndFree(partition, pt)
				if ok {
					break
				}
			}
			if full != nil {
				full, ok = mp.searchAndFree(full, pt)
				if ok {
					break
				}
			}
		}
	}

	return ok
}

func (mp *MemoPool) searchAndFree(mb *memoBlock, pt unsafe.Pointer) (*memoBlock, bool) {
	if mb.free(pt) {
		if mb.nFree == mp.nUnitCount && mb != mp.partial.Next {
			moveAfter(&mp.empty, mb)
		} else {
			// move to partial head
			moveAfter(&mp.partial, mb)
		}
		return mb, true
	}
	mb = mb.Next
	return mb, false
}


type memoBlock struct {
	pBlock     uintptr
	Next       *memoBlock
	Pre        *memoBlock
	nUnitSize  uintptr
	nUnitCount uintptr
	nFree      uintptr
	nFirst     uintptr // offset unit
	bs         []byte //only hold to avoid gc

	unitTtlSize uintptr
	endPtr uintptr
}


func newMemoPool(nUnitSize, nUnitCount uintptr) *memoBlock {
	ln := nUnitCount * (nUnitSize + intSize)
	bs := make([]byte, ln)
	pBlock := (*reflect.SliceHeader)(unsafe.Pointer(&bs)).Data

	var pCurUnit = pBlock
	for i := uintptr(0) ; i < nUnitCount; i ++ {
		*(*uintptr)(unsafe.Pointer(pCurUnit)) = i + 1
		pCurUnit += uintptr(nUnitSize + intSize)
	}

	return &memoBlock{
		pBlock:      pBlock,
		nUnitSize:   nUnitSize,
		nUnitCount:  nUnitCount,
		nFree:       nUnitCount,
		nFirst:      0,
		bs:          bs,
		unitTtlSize: nUnitSize + intSize,
		endPtr: pBlock + uintptr(len(bs)),
	}
}

func (mb *memoBlock) alloc() (unsafe.Pointer) {
	if mb.nFree == 0 {
		return nil
	}

	// unit start ptr
	ptr := mb.pBlock + (mb.unitTtlSize) * mb.nFirst

	next := *(*uintptr)(unsafe.Pointer(ptr))
	mb.nFirst = next
	mb.nFree --
	return unsafe.Pointer(ptr + intSize)
}

func (mb *memoBlock) free(ptr unsafe.Pointer) bool {
	startPtr := uintptr(ptr) - intSize
	if startPtr < mb.pBlock || startPtr >= mb.endPtr {
		return false
	}
	unitSize := mb.nUnitSize + intSize
	if (startPtr - mb.pBlock) % unitSize != 0 {
		return false
	}
	idx := (startPtr - mb.pBlock) / unitSize
	first := mb.nFirst
	mb.nFirst = idx
	*(*uintptr)(unsafe.Pointer(startPtr)) = first
	mb.nFree ++
	return true
}

// insert memo pool before current memo pool
func moveAfter(mp *memoBlock, insert *memoBlock)  {
	if mp.Next == insert {
		return
	}
	insert.remove()
	next := mp.Next
	if next != nil {
		next.Pre = insert
	}
	insert.Pre = mp
	insert.Next = next
	mp.Next = insert
}

// remove self from the list
func (mb *memoBlock) remove() {
	pre := mb.Pre
	next := mb.Next
	if pre != nil {
		pre.Next = next
	}
	if next != nil {
		next.Pre = pre
	}
	mb.Next = nil
	mb.Pre = nil
}

/**
	Use local id to isolate memory pool, and use share memory pool to fetch slow shared memory slowly.
	Hot data should be kept in local pool (top blocks).
*/
type Pools struct {
	max int
	mps []*MemoPool
	shared *MemoPool
	m sync.Mutex
}

func NewParallelPools(maxLocal int, unitSize, localUnitCount, maxLocalBlock, shareUnitCount , maxShareBlock int) *Pools {
	mps := make([]*MemoPool, maxLocal)
	for i := 0 ; i < maxLocal; i ++ {
		mps[i] = New(unitSize, localUnitCount, maxLocalBlock)
	}
	return &Pools{
		max: maxLocal,
		mps: mps,
		shared: New(unitSize, shareUnitCount, maxShareBlock),
	}
}

//Alloc a unit memory and return the pointer. Same local id should keep same order to recalling method.
func (mp *Pools) Alloc(local int) unsafe.Pointer {
	var ptr unsafe.Pointer
	if ptr = mp.mps[local].Alloc(); ptr != nil {
		return ptr
	}
	log.Println("alloc lock start")
	mp.m.Lock()
	ptr = mp.shared.Alloc()
	mp.m.Unlock()
	return ptr
}

//Free a unit memory with the start pointer. Same local id should keep order to recalling method.
func (mp *Pools) Free(local int, p unsafe.Pointer) bool {
	if mp.mps[local].Free(p) {
		return true
	}
	log.Println("free lock start")
	mp.m.Lock()
	ret := mp.shared.Free(p)
	mp.m.Unlock()
	return ret
}

