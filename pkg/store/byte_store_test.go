package store

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Assert we can generate reliable byteSlab indexes for all sizes of
// allocation Once we have this size->index mapping we can build separate
// byteSlabs for each allocation size range.
func TestIndexForSize(t *testing.T) {
	// Zero is a special case - we test it specially
	assert.Equal(t, uint32(0), indexForSize(0))

	for i := uint32(1); i <= 4; i++ {
		assert.Equal(t, uint32(0), indexForSize(i))
	}

	for i := uint32(5); i <= 12; i++ {
		assert.Equal(t, uint32(1), indexForSize(i))
	}

	for i := uint32(13); i <= 28; i++ {
		assert.Equal(t, uint32(2), indexForSize(i))
	}

	for i := uint32(29); i <= 60; i++ {
		assert.Equal(t, uint32(3), indexForSize(i))
	}

	for i := uint32(61); i <= 124; i++ {
		assert.Equal(t, uint32(4), indexForSize(i))
	}

	for i := uint32(125); i <= 252; i++ {
		assert.Equal(t, uint32(5), indexForSize(i))
	}

	for i := uint32(253); i <= 508; i++ {
		assert.Equal(t, uint32(6), indexForSize(i))
	}

	for i := uint32(509); i <= 1020; i++ {
		assert.Equal(t, uint32(7), indexForSize(i))
	}

	// Hopefully these are enough cases, it's hard to write a generic
	// test without just writing out the indexForSize method again
}

// To make the 0->4 allocations sizes all point to the same byteSlab we had to
// convert 0 sized allocations to 1 sized allocations when deriving the index.
// This is a pretty horrific, method as it was implemented without branching.
// One could argue this was important for performance, but the truth is, it was
// a fun challenge (and surprisingly hard)
func TestZeroToOne(t *testing.T) {
	// Assert that 0 converts to 1
	assert.Equal(t, uint32(1), zeroToOne(0))

	// Assert that some very low values convert to 0
	// Assert that some very high values convert to 0
	for value := uint32(1); value < 32; value++ {
		assert.Equal(t, uint32(0), zeroToOne(value))
		assert.Equal(t, uint32(0), zeroToOne(0-value))
	}

	// Assert that larger values around power of 2 thresholds convert to 0
	// There's a bit of overlap with the previous loop - but we're ok with that
	for shift := 1; shift < 32; shift++ {
		value := uint32(1) << shift
		assert.Equal(t, uint32(0), zeroToOne(value))
		assert.Equal(t, uint32(0), zeroToOne(value+1))
		assert.Equal(t, uint32(0), zeroToOne(value-1))
	}
}

// Demonstrate that we can create bytes, then get those bytes and modify them
// we can then get those bytes again and will see the modification We ensure
// that we allocate so many bytes that we will need more than one chunk to
// store all bytes.
func Test_Bytes_GetModifyGet(t *testing.T) {
	const chunkSize = 1024
	bs := NewByteStore(chunkSize)

	// Create all the byte slices
	pointers := make([]BytePointer, chunkSize*3)
	for i := range pointers {
		p, err := bs.New(8)
		require.NoError(t, err)
		pointers[i] = p
	}

	// Get the bytes from the store and write data into it
	for i, p := range pointers {
		bytes := bs.Get(p)
		intBytes := intToBytes(i)
		copy(bytes, intBytes)
	}

	// Assert that all of the modifications are visible
	for i, p := range pointers {
		bytes := bs.Get(p)
		value := bytesToInt(bytes)
		assert.Equal(t, i, value)
	}
}

// Demonstrate that we can create bytes, then get those bytes and modify them
// we can then get those bytes again and will see the modification.  We ensure
// that we allocate so many bytes that we will need more than one chunk to
// store all bytes.
func Test_Bytes_GetModifyGet_OddSizing(t *testing.T) {
	const chunkSize = 1024
	bs := NewByteStore(chunkSize)

	// Create all the byte slices
	pointers := make([]BytePointer, chunkSize*3)
	size := uint32(0)
	for i := range pointers {
		p, err := bs.New(size)
		size++
		if size > chunkSize {
			size = 0
		}
		require.NoError(t, err)
		pointers[i] = p
	}

	// Get the bytes from the store and write data into it
	for i, p := range pointers {
		bytes := bs.Get(p)

		// Write value into the bytes
		for j := range bytes {
			bytes[j] = byte(i)
		}
	}

	// Assert that all of the modifications are visible
	for i, p := range pointers {
		bytes := bs.Get(p)

		// Write value into the expected bytes
		expectedBytes := make([]byte, len(bytes))
		for j := range expectedBytes {
			expectedBytes[j] = byte(i)
		}

		assert.Equal(t, expectedBytes, bytes)
	}
}

func intToBytes(value int) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, uint64(value))
	return bytes
}

func bytesToInt(bytes []byte) int {
	return int(binary.LittleEndian.Uint64(bytes))
}
