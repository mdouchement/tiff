package tiff

import (
	"fmt"
	"math"
	"math/big"
)

type tag struct {
	id       uint16
	datatype uint
	val      []uint
}

// firstVal returns the first uint of the features entry with the given tag,
// or 0 if the tag does not exist.
func (t tag) firstVal() uint {
	if len(t.val) == 0 {
		return 0
	}
	return t.val[0]
}

// rational returns the first unsigned rational at index of the features entry with the given tag,
// or 0 if the tag does not exist.
func (t tag) rational(index int) *big.Rat {
	if len(t.val) <= index {
		return big.NewRat(0, 0)
	}
	u64 := uint64(t.val[index])
	num := int64(u64 & 0xFFFFFFFF)
	denom := int64(u64 >> 32)
	return big.NewRat(num, denom)
}

// sRational returns the rational at index of the features entry with the given tag,
// or 0 if the tag does not exist.
func (t tag) sRational(index int) *big.Rat {
	if len(t.val) <= index {
		return big.NewRat(0, 0)
	}
	u64 := uint64(t.val[index])
	num := int32(u64 & 0xFFFFFFFF)
	denom := int32(u64 >> 32)
	return big.NewRat(int64(num), int64(denom))
}

// double returns the float64 at index of the features entry with the given tag,
// or 0 if the tag does not exist.
func (t tag) double(index int) float64 {
	if len(t.val) <= index {
		return 0
	}
	return math.Float64frombits(uint64(t.val[index]))
}

// asFloat returns the converted float64 at index of the features entry with the given tag,
// or 0 if the tag does not exist.
func (t tag) asFloat(index int) float64 {
	switch t.datatype {
	case dtRational:
		v, _ := t.rational(index).Float64()
		return v
	case dtSRational:
		v, _ := t.sRational(index).Float64()
		return v
	case dtDouble:
		return t.double(index)
	default:
		if len(t.val) <= index {
			return 0
		}
		return float64(t.val[index])
	}
}

// Name returns the common name of the tag.
func (t tag) Name() string {
	return tagname(t.id)
}

// PrettyPrintedValue returns the formatted value.
func (t tag) PrettyPrintedValue() string {
	return valuename(t)
}

// String nimplements Stringer.
func (t tag) String() string {
	return fmt.Sprintf("%s: %s", t.Name(), t.PrettyPrintedValue())
}
