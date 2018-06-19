package bits

import (
	"math/big"
	"testing"
)

func TestMaxRange(t *testing.T) {
	start := FromHexP("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	end := FromHexP("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	r := MaxRange()

	if !r.Start.Equals(start) {
		t.Error("max range does not start at the beginning")
	}
	if !r.End.Equals(end) {
		t.Error("max range does not end at the end")
	}
}

func TestRange_IntervalP(t *testing.T) {
	max := MaxRange()

	numIntervals := 97
	expectedAvg := (&big.Int{}).Div(max.IntervalSize(), big.NewInt(int64(numIntervals)))
	maxDiff := big.NewInt(int64(numIntervals))

	var lastEnd Bitmap

	for i := 1; i <= numIntervals; i++ {
		ival := max.IntervalP(i, numIntervals)
		if i == 1 && !ival.Start.Equals(max.Start) {
			t.Error("first interval does not start at 0")
		}
		if i == numIntervals && !ival.End.Equals(max.End) {
			t.Error("last interval does not end at max")
		}
		if i > 1 && !ival.Start.Equals(lastEnd.Add(FromShortHexP("1"))) {
			t.Errorf("interval %d of %d: last end was %s, this start is %s", i, numIntervals, lastEnd.Hex(), ival.Start.Hex())
		}

		if ival.IntervalSize().Cmp((&big.Int{}).Add(expectedAvg, maxDiff)) > 0 || ival.IntervalSize().Cmp((&big.Int{}).Sub(expectedAvg, maxDiff)) < 0 {
			t.Errorf("interval %d of %d: interval size is outside the normal range", i, numIntervals)
		}

		lastEnd = ival.End
	}
}
