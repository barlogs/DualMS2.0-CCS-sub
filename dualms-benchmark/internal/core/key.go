package core

import (
	"fmt"
	"sort"
)

func KeyGen(pp PublicParams, signerID int) (PublicKey, SecretKey, error) {
	if pp.Ring == nil {
		return PublicKey{}, SecretKey{}, fmt.Errorf("public params have nil ring")
	}
	if signerID < 0 {
		return PublicKey{}, SecretKey{}, fmt.Errorf("signer id must be non-negative: %d", signerID)
	}

	s, err := SampleSmallVectorEta1(pp, pp.SecretDim(), fmt.Sprintf("s_%d", signerID))
	if err != nil {
		return PublicKey{}, SecretKey{}, fmt.Errorf("sample secret key: %w", err)
	}

	t, err := MatrixVectorMulRq(pp.Ring, pp.ABar, s, fmt.Sprintf("t_%d", signerID))
	if err != nil {
		return PublicKey{}, SecretKey{}, fmt.Errorf("compute public key: %w", err)
	}

	return PublicKey{SignerID: signerID, T: t}, SecretKey{SignerID: signerID, S: s}, nil
}

func KeyAgg(pp PublicParams, keys []PublicKey) (KeyAggResult, error) {
	sortedKeys, signerIDs, err := validateAndSortPublicKeysForKeyAgg(pp, keys)
	if err != nil {
		return KeyAggResult{}, err
	}
	listDigest := HAggListDigest(pp, sortedKeys)
	return keyAggWithListDigestSorted(pp, sortedKeys, signerIDs, listDigest)
}

func KeyAggWithListDigest(pp PublicParams, keys []PublicKey, listDigest []byte) (KeyAggResult, error) {
	sortedKeys, signerIDs, err := validateAndSortPublicKeysForKeyAgg(pp, keys)
	if err != nil {
		return KeyAggResult{}, err
	}
	return keyAggWithListDigestSorted(pp, sortedKeys, signerIDs, listDigest)
}

func validateAndSortPublicKeysForKeyAgg(pp PublicParams, keys []PublicKey) ([]PublicKey, []int, error) {
	if pp.Ring == nil {
		return nil, nil, fmt.Errorf("public params have nil ring")
	}
	if len(keys) == 0 {
		return nil, nil, fmt.Errorf("no public keys")
	}

	sortedKeys := append([]PublicKey(nil), keys...)
	sort.Slice(sortedKeys, func(i int, j int) bool {
		return sortedKeys[i].SignerID < sortedKeys[j].SignerID
	})

	seen := make(map[int]struct{}, len(sortedKeys))
	signerIDs := make([]int, 0, len(sortedKeys))

	for _, key := range sortedKeys {
		if key.SignerID < 0 {
			return nil, nil, fmt.Errorf("signer id must be non-negative: %d", key.SignerID)
		}
		if _, ok := seen[key.SignerID]; ok {
			return nil, nil, fmt.Errorf("duplicate signer id: %d", key.SignerID)
		}
		if len(key.T.Data) != pp.KDim {
			return nil, nil, fmt.Errorf("public key for signer %d has dimension %d, want %d", key.SignerID, len(key.T.Data), pp.KDim)
		}

		seen[key.SignerID] = struct{}{}
		signerIDs = append(signerIDs, key.SignerID)
	}

	return sortedKeys, signerIDs, nil
}

func keyAggWithListDigestSorted(pp PublicParams, sortedKeys []PublicKey, signerIDs []int, listDigest []byte) (KeyAggResult, error) {
	tAgg := NewRingVector(pp.Ring, pp.KDim, "t_tilde")
	coeffs := make(map[int]AggregationCoefficient, len(sortedKeys))

	for _, key := range sortedKeys {
		coeff := AggregationCoefficient{SignerID: key.SignerID, A: HAggFromDigest(pp, listDigest, key)}
		weightedT, err := PolyVectorScalarMul(pp.Ring, coeff.A.Value, key.T, fmt.Sprintf("a_%d_t_%d", key.SignerID, key.SignerID))
		if err != nil {
			return KeyAggResult{}, fmt.Errorf("weight public key for signer %d: %w", key.SignerID, err)
		}

		var addErr error
		tAgg, addErr = VectorAddRq(pp.Ring, tAgg, weightedT, "t_tilde")
		if addErr != nil {
			return KeyAggResult{}, fmt.Errorf("accumulate public key for signer %d: %w", key.SignerID, addErr)
		}

		coeffs[key.SignerID] = coeff
	}

	return KeyAggResult{
		PublicKeys: sortedKeys,
		Coeffs:     coeffs,
		AggPK:      AggPublicKey{TildeT: tAgg},
		SignerIDs:  signerIDs,
	}, nil
}
