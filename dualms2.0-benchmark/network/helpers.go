package network

import (
	"time"

	"github.com/tuneinsight/lattigo/v5/ring"
)

func collectAbortedSignerIDs(partials []PartialSig) []int {
	ids := make([]int, 0)
	for _, partial := range partials {
		if partial.Aborted {
			ids = append(ids, partial.SignerID)
		}
	}
	return ids
}

func durationMillis(duration time.Duration) float64 {
	return float64(duration.Nanoseconds()) / 1e6
}

func ringVectorsEqual(r *ring.Ring, lhs RingVector, rhs RingVector) bool {
	if len(lhs.Data) != len(rhs.Data) {
		return false
	}
	for i := range lhs.Data {
		if !r.Equal(lhs.Data[i], rhs.Data[i]) {
			return false
		}
	}
	return true
}
