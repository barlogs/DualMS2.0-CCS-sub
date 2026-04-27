package core

import (
	"fmt"

	"github.com/tuneinsight/lattigo/v5/ring"
)

func SAgg(pp PublicParams, partials []PartialSig) (AggSig, error) {
	if pp.Ring == nil {
		return AggSig{}, fmt.Errorf("public params have nil ring")
	}
	if len(partials) == 0 {
		return AggSig{}, fmt.Errorf("no partial signatures")
	}

	challenge := partials[0].C
	if challenge.Value.Poly.N() == 0 {
		return AggSig{}, fmt.Errorf("first partial signature has empty challenge")
	}

	zAgg := NewRingVector(pp.Ring, pp.LDim, "z_tilde")
	rAgg := NewRingVector(pp.Ring, pp.LPrimeDim, "r_tilde")
	eAgg := NewRingVector(pp.Ring, pp.KDim, "e_tilde")
	signerIDs := make([]int, 0, len(partials))
	seen := make(map[int]struct{}, len(partials))

	for _, partial := range partials {
		if err := validatePartialForSAgg(pp, partial, challenge, seen); err != nil {
			return AggSig{}, err
		}

		zHead, zTail, err := SplitVector(pp.Ring, partial.Z, pp.LDim, fmt.Sprintf("z_%d", partial.SignerID))
		if err != nil {
			return AggSig{}, fmt.Errorf("split z for signer %d: %w", partial.SignerID, err)
		}
		rHead, rTail, err := SplitVector(pp.Ring, partial.R, pp.LPrimeDim, fmt.Sprintf("r_%d", partial.SignerID))
		if err != nil {
			return AggSig{}, fmt.Errorf("split r for signer %d: %w", partial.SignerID, err)
		}
		if len(zTail.Data) != pp.KDim {
			return AggSig{}, fmt.Errorf("z tail for signer %d has dimension %d, want %d", partial.SignerID, len(zTail.Data), pp.KDim)
		}
		if len(rTail.Data) != pp.KDim {
			return AggSig{}, fmt.Errorf("r tail for signer %d has dimension %d, want %d", partial.SignerID, len(rTail.Data), pp.KDim)
		}

		zAgg, err = VectorAddRq(pp.Ring, zAgg, zHead, "z_tilde")
		if err != nil {
			return AggSig{}, fmt.Errorf("accumulate z for signer %d: %w", partial.SignerID, err)
		}
		rAgg, err = VectorAddRq(pp.Ring, rAgg, rHead, "r_tilde")
		if err != nil {
			return AggSig{}, fmt.Errorf("accumulate r for signer %d: %w", partial.SignerID, err)
		}
		ePart, err := VectorAddRq(pp.Ring, zTail, rTail, fmt.Sprintf("e_%d", partial.SignerID))
		if err != nil {
			return AggSig{}, fmt.Errorf("compute e part for signer %d: %w", partial.SignerID, err)
		}
		eAgg, err = VectorAddRq(pp.Ring, eAgg, ePart, "e_tilde")
		if err != nil {
			return AggSig{}, fmt.Errorf("accumulate e for signer %d: %w", partial.SignerID, err)
		}

		seen[partial.SignerID] = struct{}{}
		signerIDs = append(signerIDs, partial.SignerID)
	}

	return AggSig{
		C:         challenge,
		ZAgg:      zAgg,
		RAgg:      rAgg,
		EAgg:      eAgg,
		SignerIDs: signerIDs,
	}, nil
}

func validatePartialForSAgg(pp PublicParams, partial PartialSig, challenge Challenge, seen map[int]struct{}) error {
	if partial.SignerID < 0 {
		return fmt.Errorf("signer id must be non-negative: %d", partial.SignerID)
	}
	if _, ok := seen[partial.SignerID]; ok {
		return fmt.Errorf("duplicate partial signature for signer %d", partial.SignerID)
	}
	if partial.Aborted {
		return fmt.Errorf("partial signature from signer %d aborted", partial.SignerID)
	}
	if partial.C.Value.Poly.N() == 0 {
		return fmt.Errorf("partial signature from signer %d has empty challenge", partial.SignerID)
	}
	if !partial.C.Equal(challenge) {
		return fmt.Errorf("challenge mismatch at signer %d", partial.SignerID)
	}
	if len(partial.Z.Data) != pp.SecretDim() {
		return fmt.Errorf("partial signature from signer %d has z dimension %d, want %d", partial.SignerID, len(partial.Z.Data), pp.SecretDim())
	}
	if len(partial.R.Data) != pp.CommitmentRandomDim() {
		return fmt.Errorf("partial signature from signer %d has r dimension %d, want %d", partial.SignerID, len(partial.R.Data), pp.CommitmentRandomDim())
	}
	return nil
}

func SplitVector(r *ring.Ring, vector RingVector, headLen int, label string) (RingVector, RingVector, error) {
	if r == nil {
		return RingVector{}, RingVector{}, fmt.Errorf("ring is nil")
	}
	if headLen <= 0 {
		return RingVector{}, RingVector{}, fmt.Errorf("head length must be positive: %d", headLen)
	}
	if len(vector.Data) <= headLen {
		return RingVector{}, RingVector{}, fmt.Errorf("vector %q length is %d, need more than head length %d", vector.Label, len(vector.Data), headLen)
	}

	tailLen := len(vector.Data) - headLen
	head := NewRingVector(r, headLen, label+"_head")
	tail := NewRingVector(r, tailLen, label+"_tail")
	for i := 0; i < headLen; i++ {
		head.Data[i].Copy(vector.Data[i])
	}
	for i := 0; i < tailLen; i++ {
		tail.Data[i].Copy(vector.Data[headLen+i])
	}
	return head, tail, nil
}
