package bits

import (
	"math/big"

	"github.com/lbryio/errors.go"
)

// Range has a start and end
type Range struct {
	Start Bitmap
	End   Bitmap
}

func MaxRange() Range {
	return Range{
		Start: Bitmap{},
		End:   MaxP(),
	}
}

// IntervalP divides the range into `num` intervals and returns the `n`th one
// intervals are approximately the same size, but may not be exact because of rounding issues
// the first interval always starts at the beginning of the range, and the last interval always ends at the end
func (r Range) IntervalP(n, num int) Range {
	if num < 1 || n < 1 || n > num {
		panic(errors.Err("invalid interval %d of %d", n, num))
	}

	start := r.intervalStart(n, num)
	end := new(big.Int)
	if n == num {
		end = r.End.Big()
	} else {
		end = r.intervalStart(n+1, num)
		end.Sub(end, big.NewInt(1))
	}

	return Range{FromBigP(start), FromBigP(end)}
}

func (r Range) intervalStart(n, num int) *big.Int {
	// formula:
	// size = (end - start) / num
	// rem = (end - start) % num
	// intervalStart = rangeStart + (size * n-1) + ((rem * n-1) % num)

	size := new(big.Int)
	rem := new(big.Int)
	size.Sub(r.End.Big(), r.Start.Big()).DivMod(size, big.NewInt(int64(num)), rem)

	size.Mul(size, big.NewInt(int64(n-1)))
	rem.Mul(rem, big.NewInt(int64(n-1))).Mod(rem, big.NewInt(int64(num)))

	start := r.Start.Big()
	start.Add(start, size).Add(start, rem)

	return start
}

func (r Range) IntervalSize() *big.Int {
	return (&big.Int{}).Sub(r.End.Big(), r.Start.Big())
}

func (r Range) Contains(b Bitmap) bool {
	return r.Start.Cmp(b) <= 0 && r.End.Cmp(b) >= 0
}
