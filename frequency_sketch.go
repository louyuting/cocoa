package cocoa

import "math"

var (
	Seed        = [...]uint64{0xc3a5c85c97cb3127, 0xb492b66fbe98f273, 0x9ae16a3b2f90404f, 0xcbf29ce484222325}
	ResetMask   = uint64(0x7777777777777777)
	OneMask     = uint64(0x1111111111111111)
	MaxCapacity = 1 << 30
)

type FrequencySketch struct {
	table     []uint64
	tableMask uint64

	sampleSize uint64
	size       uint64
}

func NewFrequencySketch(capacity int) *FrequencySketch {
	f := &FrequencySketch{}
	f.ensureCapacity(capacity)
	return f
}

func (f *FrequencySketch) ensureCapacity(maxSize int) {
	if maxSize < 0 {
		maxSize = 0
	}
	maxSize = int(math.Min(float64(maxSize), float64(MaxCapacity)))
	if len(f.table) > maxSize {
		return
	}

	maxSize = ceilingPowerOfTwo(maxSize)
	f.table = make([]uint64, maxSize)
	f.tableMask = uint64(maxSize - 1)
	f.size = 0
	f.sampleSize = 10 * uint64(maxSize)

}

// frequency returns the estimated number of occurrences of an element, up to the maximum (15).
func (f *FrequencySketch) frequency(key []byte) int {
	// hash the key
	hash := hash(key)
	// counter index in table[idx]
	// start in [0,4,8,12]
	start := int((hash & 3) << 2)

	frequency := 15
	for i := 0; i < 4; i++ {
		idx := f.indexOf(hash, i)
		count := (f.table[idx] >> ((start + i) << 2)) & 0xf
		frequency = int(math.Min(float64(frequency), float64(count)))
	}

	return frequency
}

// increment increments the popularity of the element if it does not exceed the maximum (15). The popularity
// of all elements will be periodically down sampled when the observed events exceeds a threshold.
// This process provides a frequency aging to allow expired long term entries to fade away.
func (f *FrequencySketch) increment(key []byte) {
	// hash the key
	hash := hash(key)
	// counter index in table[idx]
	// start in [0,4,8,12]
	start := int((hash & 3) << 2)

	idx1 := f.indexOf(hash, 0)
	idx2 := f.indexOf(hash, 1)
	idx3 := f.indexOf(hash, 2)
	idx4 := f.indexOf(hash, 3)

	added := f.increaseAt(idx1, start)
	added = f.increaseAt(idx2, start+1) || added
	added = f.increaseAt(idx3, start+2) || added
	added = f.increaseAt(idx4, start+3) || added
	f.size++

	if added && f.size == f.sampleSize {
		f.reset()
	}
}

// Reduces every counter by half of its original value.
func (f *FrequencySketch) reset() {
	count := 0
	for i := 0; i < len(f.table); i++ {
		count += bitCount(f.table[i] & OneMask)
		f.table[i] = (f.table[i] >> 1) & ResetMask
	}
	f.size = (f.size >> 1) - uint64(count>>2)
}

// Returns the table index for the counter at the specified depth.
func (f *FrequencySketch) indexOf(hash uint64, depth int) (idx int) {
	h := (hash + Seed[depth]) * Seed[depth]
	h += h >> 32
	idx = int(h & f.tableMask)
	return
}

// Increments the specified counter by 1 if it is not already at the maximum value (15).
// idx: the table index (table[idx] has 16 counters)
// ordinate: the counter to increase, value in [0-15]
// return: if incremented
func (f *FrequencySketch) increaseAt(idx, ordinate int) bool {
	// offset in [0,4,8,12,  16,20,24,28,  32,36,40,44,  48,52,56,60]
	offset := uint64(ordinate << 2)
	mask := uint64(0xf << offset)

	if f.table[idx]&mask != mask {
		f.table[idx] += 1 << offset
		return true
	}
	return false
}
