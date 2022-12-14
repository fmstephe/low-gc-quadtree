package bytestore

import (
	"math"
	"math/bits"
)

type Stats struct {
	TotalAllocs    int
	TotalFrees     int
	TotalRawAllocs int
	TotalLive      int
	TotalReused    int
	TotalChunks    int
	SlabStats      []ByteSlabStats
}

type Pointer struct {
	chunk      uint32
	slotOffset uint32
	byteOffset uint32
	size       uint32
}

func (p Pointer) IsNil() bool {
	return p.chunk == 0 && p.slotOffset == 0
}

type Store struct {
	slabs []byteSlab
}

func New() *Store {
	return &Store{
		slabs: initialiseSlabs(),
	}
}

func initialiseSlabs() []byteSlab {
	slabs := make([]byteSlab, 34)

	allocSize := uint32(1)
	for i := range slabs {
		// Special case for 0 size slab allocations
		if i == 0 {
			slabs[0] = newByteSlab(0)
			continue
		}
		// Special case for allocations greater than 2^31 In principal
		// we would want this slab to be sized 2^32, but with 32 bits
		// that's 0, so we get as close as we can.
		if i == 33 {
			slabs[33] = newByteSlab(math.MaxUint32)
			continue
		}
		slabs[i] = newByteSlab(allocSize)
		allocSize = allocSize << 1
	}

	return slabs
}

func (s *Store) Alloc(size uint32) (Pointer, []byte) {
	idx := indexForSize(size)
	// TODO we should explicitly panic if the idx is out of range here
	// Need a clear and explicit panic message
	// Eventually we will probably manage large allocations separately
	p := s.slabs[idx].alloc(size)
	return p, s.Get(p)
}

func (s *Store) Get(p Pointer) []byte {
	idx := indexForSize(p.size)
	return s.slabs[idx].get(p)
}

func (s *Store) Free(p Pointer) {
	idx := indexForSize(p.size)
	s.slabs[idx].free(p)
}

func (s *Store) GetStats() Stats {
	stats := Stats{
		SlabStats: make([]ByteSlabStats, len(s.slabs)),
	}
	for i := range s.slabs {
		slabStats := s.slabs[i].GetStats()

		stats.SlabStats[i] = slabStats

		stats.TotalAllocs += slabStats.Allocs
		stats.TotalFrees += slabStats.Frees
		stats.TotalRawAllocs += slabStats.RawAllocs
		stats.TotalLive += slabStats.Live
		stats.TotalReused += slabStats.Reused
		stats.TotalChunks += slabStats.Chunks
	}

	return stats
}

func indexForSize(size uint32) int {
	if size == 0 {
		return 0
	}
	idx := bits.Len32(size-1) + 1
	return idx
}
