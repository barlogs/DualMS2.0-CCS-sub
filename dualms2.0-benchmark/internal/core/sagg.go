package core

import "fmt"

func SAgg(pp PublicParams, message string, aggPK AggPublicKey, signerIDs []int, allMsgs []Round1Msg, partials []PartialSig) (AggSig, error) {
	if pp.Ring == nil {
		return AggSig{}, fmt.Errorf("public params have nil ring")
	}
	if len(partials) == 0 {
		return AggSig{}, fmt.Errorf("no partial signatures")
	}

	round1Agg, err := aggregateRound1MessagesForAggregation(pp, signerIDs, allMsgs)
	if err != nil {
		return AggSig{}, err
	}
	challenge := partials[0].C
	zAgg := NewRingVector(pp.Ring, pp.LDim, "z_agg")
	rAgg := NewRingElement(pp.Ring, "r_agg")
	rBarAgg := NewRingVector(pp.Ring, pp.LDim, "rbar_agg")
	seen := make(map[int]struct{}, len(partials))
	collectedSignerIDs := make([]int, 0, len(partials))
	expected := make(map[int]struct{}, len(signerIDs))
	for _, signerID := range signerIDs {
		if _, ok := expected[signerID]; ok {
			return AggSig{}, fmt.Errorf("duplicate signer id in aggregation signer set: %d", signerID)
		}
		expected[signerID] = struct{}{}
	}

	for _, partial := range partials {
		if partial.SignerID < 0 {
			return AggSig{}, fmt.Errorf("signer id must be non-negative: %d", partial.SignerID)
		}
		if _, ok := expected[partial.SignerID]; !ok {
			return AggSig{}, fmt.Errorf("partial signature from signer %d is not in aggregation signer set", partial.SignerID)
		}
		if _, ok := seen[partial.SignerID]; ok {
			return AggSig{}, fmt.Errorf("duplicate partial signature for signer %d", partial.SignerID)
		}
		if partial.Aborted {
			return AggSig{}, fmt.Errorf("partial signature from signer %d aborted", partial.SignerID)
		}
		if partial.C.Value.Poly.N() == 0 {
			return AggSig{}, fmt.Errorf("partial signature from signer %d has empty challenge", partial.SignerID)
		}
		if !partial.C.Equal(challenge) {
			return AggSig{}, fmt.Errorf("challenge mismatch at signer %d", partial.SignerID)
		}
		if len(partial.Z1.Data) != pp.LDim {
			return AggSig{}, fmt.Errorf("partial signature from signer %d has z1 dimension %d, want %d", partial.SignerID, len(partial.Z1.Data), pp.LDim)
		}
		if partial.R.Poly.N() == 0 {
			return AggSig{}, fmt.Errorf("partial signature from signer %d has empty r", partial.SignerID)
		}
		if len(partial.RBar1.Data) != pp.LDim {
			return AggSig{}, fmt.Errorf("partial signature from signer %d has rbar1 dimension %d, want %d", partial.SignerID, len(partial.RBar1.Data), pp.LDim)
		}

		zAgg, err = VectorAddRq(pp.Ring, zAgg, partial.Z1, "z_agg")
		if err != nil {
			return AggSig{}, fmt.Errorf("accumulate z1 for signer %d: %w", partial.SignerID, err)
		}
		rAgg, err = RingElementAddRq(pp.Ring, rAgg, partial.R, "r_agg")
		if err != nil {
			return AggSig{}, fmt.Errorf("accumulate r for signer %d: %w", partial.SignerID, err)
		}
		rBarAgg, err = VectorAddRq(pp.Ring, rBarAgg, partial.RBar1, "rbar_agg")
		if err != nil {
			return AggSig{}, fmt.Errorf("accumulate rbar1 for signer %d: %w", partial.SignerID, err)
		}

		seen[partial.SignerID] = struct{}{}
		collectedSignerIDs = append(collectedSignerIDs, partial.SignerID)
	}
	if len(seen) != len(expected) {
		return AggSig{}, fmt.Errorf("partial signature signer set has size %d, want %d", len(seen), len(expected))
	}

	u, err := reconstructRoundedTarget(pp, message, challenge, zAgg, rAgg, rBarAgg, aggPK)
	if err != nil {
		return AggSig{}, fmt.Errorf("reconstruct rounded target: %w", err)
	}
	h, err := SubRoundedResiduesMod(round1Agg.WAggRoundedResidue, u, "hint")
	if err != nil {
		return AggSig{}, fmt.Errorf("compute hint: %w", err)
	}

	return AggSig{
		C:         challenge,
		ZAgg:      zAgg,
		RAgg:      rAgg,
		RBarAgg:   rBarAgg,
		H:         h,
		SignerIDs: collectedSignerIDs,
	}, nil
}
