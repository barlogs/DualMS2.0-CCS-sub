package core

import (
	"crypto/sha256"
	"fmt"
)

func Sign1(pp PublicParams, sk SecretKey, keyAgg KeyAggResult, message string, sessionID string) (Round1Msg, Round1State, error) {
	if pp.Ring == nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("public params have nil ring")
	}
	if sessionID == "" {
		return Round1Msg{}, Round1State{}, fmt.Errorf("session id must not be empty")
	}
	if len(keyAgg.PublicKeys) == 0 {
		return Round1Msg{}, Round1State{}, fmt.Errorf("key aggregation result has no public keys")
	}
	if len(keyAgg.AggPK.TildeTExact.Data) != pp.MDim {
		return Round1Msg{}, Round1State{}, fmt.Errorf("aggregate exact public key dimension is %d, want %d", len(keyAgg.AggPK.TildeTExact.Data), pp.MDim)
	}
	if len(keyAgg.AggPK.TildeTRounded.Data) != pp.MDim {
		return Round1Msg{}, Round1State{}, fmt.Errorf("aggregate rounded public key dimension is %d, want %d", len(keyAgg.AggPK.TildeTRounded.Data), pp.MDim)
	}

	coeff, ok := keyAgg.Coeffs[sk.SignerID]
	if !ok {
		return Round1Msg{}, Round1State{}, fmt.Errorf("missing aggregation coefficient for signer %d", sk.SignerID)
	}
	if coeff.SignerID != sk.SignerID {
		return Round1Msg{}, Round1State{}, fmt.Errorf("aggregation coefficient signer id is %d, want %d", coeff.SignerID, sk.SignerID)
	}
	if len(sk.S.Data) != pp.LDim+pp.MDim {
		return Round1Msg{}, Round1State{}, fmt.Errorf("secret key dimension is %d, want %d", len(sk.S.Data), pp.LDim+pp.MDim)
	}

	com := HComRounded(pp, keyAgg.AggPK, message)
	y, err := SampleGaussianVector(pp, pp.LDim+pp.MDim, sign1SampleLabel(sessionID, sk.SignerID, message, "y"))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("sample y for signer %d: %w", sk.SignerID, err)
	}
	r, err := SampleGaussianPoly(pp, sign1SampleLabel(sessionID, sk.SignerID, message, "r"))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("sample r for signer %d: %w", sk.SignerID, err)
	}
	rBar, err := SampleSmallVectorEta1(pp, pp.LDim+pp.MDim, sign1SampleLabel(sessionID, sk.SignerID, message, "rbar"))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("sample rbar for signer %d: %w", sk.SignerID, err)
	}

	aY, err := MatrixVectorMulRq(pp.Ring, pp.ABar, y, fmt.Sprintf("Abar_y_%d", sk.SignerID))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("compute Abar*y for signer %d: %w", sk.SignerID, err)
	}
	rB, err := PolyVectorScalarMul(pp.Ring, r, com.BVec, fmt.Sprintf("r_b_%d", sk.SignerID))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("compute r*b for signer %d: %w", sk.SignerID, err)
	}
	bBarRBar, err := MatrixVectorMulRq(pp.Ring, com.BBar, rBar, fmt.Sprintf("Bbar_rbar_%d", sk.SignerID))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("compute Bbar*rbar for signer %d: %w", sk.SignerID, err)
	}

	wPartial, err := VectorAddRq(pp.Ring, aY, rB, fmt.Sprintf("w_partial_%d", sk.SignerID))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("add Abar*y and r*b for signer %d: %w", sk.SignerID, err)
	}
	w, err := VectorAddRq(pp.Ring, wPartial, bBarRBar, fmt.Sprintf("w_%d", sk.SignerID))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("add Bbar*rbar for signer %d: %w", sk.SignerID, err)
	}

	msg := Round1Msg{
		SignerID: sk.SignerID,
		W:        w,
	}

	st := Round1State{
		SignerID:         sk.SignerID,
		SK:               sk,
		AggPK:            keyAgg.AggPK,
		Message:          message,
		SessionID:        sessionID,
		Coeff:            coeff,
		KeyAggPublicKeys: copyPublicKeys(keyAgg.PublicKeys),
		KeyAggSignerIDs:  append([]int(nil), keyAgg.SignerIDs...),
		Y:                y,
		R:                r,
		RBar:             rBar,
		W:                msg.W,
	}

	return msg, st, nil
}

func copyPublicKeys(keys []PublicKey) []PublicKey {
	copied := make([]PublicKey, len(keys))
	copy(copied, keys)
	return copied
}

func sign1SampleLabel(sessionID string, signerID int, message string, variable string) string {
	messageDigest := sha256.Sum256([]byte(message))
	return fmt.Sprintf("sign1/session=%s/message=%x/signer=%d/%s", sessionID, messageDigest, signerID, variable)
}
