package core

import (
	"fmt"
)

func Sign2(pp PublicParams, st Round1State, allMsgs []Round1Msg) (PartialSig, error) {
	if err := validateSign2State(pp, st); err != nil {
		return PartialSig{}, err
	}

	wAgg, err := aggregateRound1Messages(pp, st, allMsgs)
	if err != nil {
		return PartialSig{}, err
	}

	c := HSig(pp, st.TAgg, st.Message, wAgg)
	aS, err := PolyVectorScalarMul(pp.Ring, st.Coeff.A.Value, st.SK.S, fmt.Sprintf("a_%d_s_%d", st.SignerID, st.SignerID))
	if err != nil {
		return PartialSig{}, fmt.Errorf("compute a_%d*s_%d: %w", st.SignerID, st.SignerID, err)
	}
	acs, err := PolyVectorScalarMul(pp.Ring, c.Value, aS, fmt.Sprintf("a_%d_c_s_%d", st.SignerID, st.SignerID))
	if err != nil {
		return PartialSig{}, fmt.Errorf("compute a_%d*c*s_%d: %w", st.SignerID, st.SignerID, err)
	}

	z, accepted, err := sign2RejSamp(pp, st, c, acs)
	if err != nil {
		return PartialSig{}, fmt.Errorf("run rejection sampling for signer %d: %w", st.SignerID, err)
	}
	if !accepted {
		return PartialSig{
			SignerID: st.SignerID,
			C:        c,
			Aborted:  true,
		}, nil
	}

	return PartialSig{
		SignerID: st.SignerID,
		C:        c,
		Z:        z,
		R:        st.R,
		Aborted:  false,
	}, nil
}

func validateSign2State(pp PublicParams, st Round1State) error {
	if pp.Ring == nil {
		return fmt.Errorf("public params have nil ring")
	}
	if st.SignerID < 0 {
		return fmt.Errorf("signer id must be non-negative: %d", st.SignerID)
	}
	if len(st.TAgg.TildeT.Data) != pp.KDim {
		return fmt.Errorf("round 1 state aggregate public key dimension is %d, want %d", len(st.TAgg.TildeT.Data), pp.KDim)
	}
	if len(st.SK.S.Data) != pp.SecretDim() {
		return fmt.Errorf("round 1 state secret key dimension is %d, want %d", len(st.SK.S.Data), pp.SecretDim())
	}
	if len(st.Y.Data) != pp.SecretDim() {
		return fmt.Errorf("round 1 state y dimension is %d, want %d", len(st.Y.Data), pp.SecretDim())
	}
	if len(st.R.Data) != pp.CommitmentRandomDim() {
		return fmt.Errorf("round 1 state r dimension is %d, want %d", len(st.R.Data), pp.CommitmentRandomDim())
	}
	if len(st.W.Data) != pp.KDim {
		return fmt.Errorf("round 1 state w dimension is %d, want %d", len(st.W.Data), pp.KDim)
	}
	if st.Coeff.SignerID != st.SignerID {
		return fmt.Errorf("round 1 state coefficient signer id is %d, want %d", st.Coeff.SignerID, st.SignerID)
	}
	if st.Coeff.A.Value.Poly.N() == 0 {
		return fmt.Errorf("round 1 state aggregation coefficient is empty")
	}
	if len(st.KeyAggSignerIDs) == 0 {
		return fmt.Errorf("round 1 state has no key aggregation signer ids")
	}
	return nil
}

func AggregateRound1Messages(pp PublicParams, st Round1State, allMsgs []Round1Msg) (RingVector, error) {
	return aggregateRound1Messages(pp, st, allMsgs)
}

func RingVectorsEqualRq(pp PublicParams, lhs RingVector, rhs RingVector) bool {
	return ringVectorsEqualRq(pp, lhs, rhs)
}

func aggregateRound1Messages(pp PublicParams, st Round1State, allMsgs []Round1Msg) (RingVector, error) {
	if len(allMsgs) == 0 {
		return RingVector{}, fmt.Errorf("no round 1 messages")
	}

	expected := make(map[int]struct{}, len(st.KeyAggSignerIDs))
	for _, signerID := range st.KeyAggSignerIDs {
		if signerID < 0 {
			return RingVector{}, fmt.Errorf("round 1 state has negative key aggregation signer id: %d", signerID)
		}
		if _, ok := expected[signerID]; ok {
			return RingVector{}, fmt.Errorf("round 1 state has duplicate key aggregation signer id: %d", signerID)
		}
		expected[signerID] = struct{}{}
	}

	seen := make(map[int]struct{}, len(allMsgs))
	wAgg := NewRingVector(pp.Ring, pp.KDim, fmt.Sprintf("w_tilde_%d_msgs", len(allMsgs)))
	for _, msg := range allMsgs {
		if _, ok := seen[msg.SignerID]; ok {
			return RingVector{}, fmt.Errorf("duplicate round 1 message for signer %d", msg.SignerID)
		}
		if _, ok := expected[msg.SignerID]; !ok {
			return RingVector{}, fmt.Errorf("round 1 message from signer %d is not in key aggregation signer set", msg.SignerID)
		}
		if len(msg.W.Data) != pp.KDim {
			return RingVector{}, fmt.Errorf("round 1 message for signer %d has w dimension %d, want %d", msg.SignerID, len(msg.W.Data), pp.KDim)
		}
		if msg.SignerID == st.SignerID && !ringVectorsEqualRq(pp, msg.W, st.W) {
			return RingVector{}, fmt.Errorf("round 1 message for signer %d does not match local round 1 state", msg.SignerID)
		}

		var err error
		wAgg, err = VectorAddRq(pp.Ring, wAgg, msg.W, "w_tilde")
		if err != nil {
			return RingVector{}, fmt.Errorf("accumulate w for signer %d: %w", msg.SignerID, err)
		}
		seen[msg.SignerID] = struct{}{}
	}

	if _, ok := seen[st.SignerID]; !ok {
		return RingVector{}, fmt.Errorf("missing local round 1 message for signer %d", st.SignerID)
	}
	if len(seen) != len(expected) {
		return RingVector{}, fmt.Errorf("round 1 message signer set has size %d, want %d", len(seen), len(expected))
	}

	return wAgg, nil
}

func ringVectorsEqualRq(pp PublicParams, lhs RingVector, rhs RingVector) bool {
	if pp.Ring == nil || len(lhs.Data) != len(rhs.Data) {
		return false
	}
	for i := range lhs.Data {
		if !pp.Ring.Equal(lhs.Data[i], rhs.Data[i]) {
			return false
		}
	}
	return true
}
