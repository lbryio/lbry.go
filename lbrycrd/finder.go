package lbrycrd

import (
	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/lbryio/lbcd/btcjson"
)

type outputFinder struct {
	unspent     []btcjson.ListUnspentResult
	lastChecked int
}

func newOutputFinder(unspentResults []btcjson.ListUnspentResult) *outputFinder {
	return &outputFinder{unspent: unspentResults, lastChecked: -1}
}

func (f *outputFinder) nextBatch(minAmount float64) ([]btcjson.ListUnspentResult, error) {
	var batch []btcjson.ListUnspentResult
	var lbcBatched float64
	for i, unspent := range f.unspent {
		if i > f.lastChecked {
			if unspent.Spendable {
				batch = append(batch, unspent)
				lbcBatched = lbcBatched + unspent.Amount
			}
		}
		if lbcBatched >= minAmount {
			f.lastChecked = i
			break
		}
		if i == len(f.unspent)-1 {
			return nil, errors.Err("Not enough unspent outputs to spend %d on supports.", minAmount)
		}
	}

	return batch, nil
}
