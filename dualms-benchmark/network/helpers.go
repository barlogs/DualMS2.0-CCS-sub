package network

import (
	"time"

	core "dualmspoc/internal/core"
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
	return float64(duration.Nanoseconds()) / float64(time.Millisecond)
}

func aggregateRound1Messages(pp PublicParams, st Round1State, allMsgs []Round1Msg) (RingVector, error) {
	return core.AggregateRound1Messages(pp, st, allMsgs)
}

func ringVectorsEqualRq(pp PublicParams, lhs RingVector, rhs RingVector) bool {
	return core.RingVectorsEqualRq(pp, lhs, rhs)
}
