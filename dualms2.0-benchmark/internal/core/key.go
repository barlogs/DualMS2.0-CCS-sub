package core

import "fmt"

func KeyGen(pp PublicParams, signerID int) (PublicKey, SecretKey, error) {
	if signerID < 0 {
		return PublicKey{}, SecretKey{}, fmt.Errorf("signer id must be non-negative: %d", signerID)
	}

	s, err := SampleSmallVectorEta1(pp, pp.LDim+pp.MDim, fmt.Sprintf("keygen/signer=%d/s", signerID))
	if err != nil {
		return PublicKey{}, SecretKey{}, fmt.Errorf("sample secret key for signer %d: %w", signerID, err)
	}

	t, err := MatrixVectorMulRq(pp.Ring, pp.ABar, s, fmt.Sprintf("t_%d", signerID))
	if err != nil {
		return PublicKey{}, SecretKey{}, fmt.Errorf("compute public key for signer %d: %w", signerID, err)
	}

	return PublicKey{
			SignerID: signerID,
			T:        t,
		}, SecretKey{
			SignerID: signerID,
			S:        s,
		}, nil
}

func KeyAgg(pp PublicParams, keys []PublicKey) (KeyAggResult, error) {
	ids, err := validatePublicKeysForKeyAgg(pp, keys)
	if err != nil {
		return KeyAggResult{}, err
	}
	listDigest := HAggListDigest(pp, keys)
	return keyAggWithListDigestValidated(pp, keys, ids, listDigest)
}

func KeyAggWithListDigest(pp PublicParams, keys []PublicKey, listDigest []byte) (KeyAggResult, error) {
	ids, err := validatePublicKeysForKeyAgg(pp, keys)
	if err != nil {
		return KeyAggResult{}, err
	}
	return keyAggWithListDigestValidated(pp, keys, ids, listDigest)
}

func validatePublicKeysForKeyAgg(pp PublicParams, keys []PublicKey) ([]int, error) {
	if pp.Ring == nil {
		return nil, fmt.Errorf("public params have nil ring")
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no public keys")
	}

	ids := make([]int, 0, len(keys))
	seen := make(map[int]struct{}, len(keys))
	for _, key := range keys {
		if key.SignerID < 0 {
			return nil, fmt.Errorf("signer id must be non-negative: %d", key.SignerID)
		}
		if _, ok := seen[key.SignerID]; ok {
			return nil, fmt.Errorf("duplicate signer id: %d", key.SignerID)
		}
		if len(key.T.Data) != pp.MDim {
			return nil, fmt.Errorf("public key for signer %d has dimension %d, want %d", key.SignerID, len(key.T.Data), pp.MDim)
		}

		seen[key.SignerID] = struct{}{}
		ids = append(ids, key.SignerID)
	}

	return ids, nil
}

func keyAggWithListDigestValidated(pp PublicParams, keys []PublicKey, ids []int, listDigest []byte) (KeyAggResult, error) {
	coeffs := make(map[int]AggregationCoefficient, len(keys))
	var tAggExact RingVector
	for idx, key := range keys {
		a := HAggFromDigest(pp, listDigest, key)
		coeffs[key.SignerID] = AggregationCoefficient{
			SignerID: key.SignerID,
			A:        a,
		}

		term, err := PolyVectorScalarMul(pp.Ring, a.Value, key.T, fmt.Sprintf("a_%d_t_%d", key.SignerID, key.SignerID))
		if err != nil {
			return KeyAggResult{}, fmt.Errorf("compute a_%d * t_%d: %w", key.SignerID, key.SignerID, err)
		}
		if idx == 0 {
			tAggExact = term
			tAggExact.Label = "t_agg_exact"
			continue
		}

		tAggExact, err = VectorAddRq(pp.Ring, tAggExact, term, "t_agg_exact")
		if err != nil {
			return KeyAggResult{}, fmt.Errorf("accumulate aggregate public key for signer %d: %w", key.SignerID, err)
		}
	}

	tAggRounded, err := RoundRingVectorToRounded(pp.Ring, tAggExact, pp.NuB, pp.QNuBFloor, "t_agg_round")
	if err != nil {
		return KeyAggResult{}, fmt.Errorf("round aggregate public key: %w", err)
	}
	tAggResidue, err := RoundedVectorStandardReps(tAggRounded)
	if err != nil {
		return KeyAggResult{}, fmt.Errorf("reduce aggregate public key into Z_q_nu_b: %w", err)
	}

	return KeyAggResult{
		PublicKeys: keys,
		Coeffs:     coeffs,
		AggPK: AggPublicKey{
			TildeTExact:          tAggExact,
			TildeTRounded:        tAggRounded,
			TildeTRoundedResidue: tAggResidue,
		},
		SignerIDs: ids,
	}, nil
}
