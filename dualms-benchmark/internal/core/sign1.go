package core

import (
	"crypto/sha256"
	"fmt"
)

func Sign1(pp PublicParams, sk SecretKey, keyAgg KeyAggResult, message string, sessionID string) (Round1Msg, Round1State, error) {
	if err := validateSign1Inputs(pp, sk, keyAgg, sessionID); err != nil {
		return Round1Msg{}, Round1State{}, err
	}

	coeff := keyAgg.Coeffs[sk.SignerID]
	com := HCom(pp, keyAgg.AggPK, message)

	y, err := SampleGaussianVector(pp, pp.SecretDim(), sign1SampleLabel(sessionID, sk.SignerID, message, "y"))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("sample y for signer %d: %w", sk.SignerID, err)
	}
	r, err := SampleGaussianPrimeVector(pp, pp.CommitmentRandomDim(), sign1SampleLabel(sessionID, sk.SignerID, message, "r"))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("sample r for signer %d: %w", sk.SignerID, err)
	}

	aBarY, err := MatrixVectorMulRq(pp.Ring, pp.ABar, y, fmt.Sprintf("Abar_y_%d", sk.SignerID))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("compute Abar*y for signer %d: %w", sk.SignerID, err)
	}
	bBarR, err := MatrixVectorMulRq(pp.Ring, com.BBar, r, fmt.Sprintf("Bbar_r_%d", sk.SignerID))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("compute Bbar*r for signer %d: %w", sk.SignerID, err)
	}
	w, err := VectorAddRq(pp.Ring, aBarY, bBarR, fmt.Sprintf("w_%d", sk.SignerID))
	if err != nil {
		return Round1Msg{}, Round1State{}, fmt.Errorf("compute w for signer %d: %w", sk.SignerID, err)
	}

	msg := Round1Msg{
		SignerID: sk.SignerID,
		W:        w,
	}
	state := Round1State{
		SignerID:         sk.SignerID,
		SK:               sk,
		TAgg:             keyAgg.AggPK,
		Message:          message,
		SessionID:        sessionID,
		Coeff:            coeff,
		KeyAggPublicKeys: copyPublicKeys(keyAgg.PublicKeys),
		KeyAggSignerIDs:  append([]int(nil), keyAgg.SignerIDs...),
		Y:                y,
		R:                r,
		W:                w,
	}

	return msg, state, nil
}

func validateSign1Inputs(pp PublicParams, sk SecretKey, keyAgg KeyAggResult, sessionID string) error {
	if pp.Ring == nil {
		return fmt.Errorf("public params have nil ring")
	}
	if sessionID == "" {
		return fmt.Errorf("session id must not be empty")
	}
	if sk.SignerID < 0 {
		return fmt.Errorf("signer id must be non-negative: %d", sk.SignerID)
	}
	if len(sk.S.Data) != pp.SecretDim() {
		return fmt.Errorf("secret key dimension is %d, want %d", len(sk.S.Data), pp.SecretDim())
	}
	if len(keyAgg.PublicKeys) == 0 {
		return fmt.Errorf("key aggregation result has no public keys")
	}
	if len(keyAgg.SignerIDs) == 0 {
		return fmt.Errorf("key aggregation result has no signer ids")
	}
	if len(keyAgg.AggPK.TildeT.Data) != pp.KDim {
		return fmt.Errorf("aggregate public key dimension is %d, want %d", len(keyAgg.AggPK.TildeT.Data), pp.KDim)
	}

	coeff, ok := keyAgg.Coeffs[sk.SignerID]
	if !ok {
		return fmt.Errorf("missing aggregation coefficient for signer %d", sk.SignerID)
	}
	if coeff.SignerID != sk.SignerID {
		return fmt.Errorf("aggregation coefficient signer id is %d, want %d", coeff.SignerID, sk.SignerID)
	}
	if coeff.A.Value.Poly.N() == 0 {
		return fmt.Errorf("aggregation coefficient for signer %d is empty", sk.SignerID)
	}

	return nil
}

func copyPublicKeys(keys []PublicKey) []PublicKey {
	copied := make([]PublicKey, len(keys))
	copy(copied, keys)
	return copied
}

func sign1SampleLabel(sessionID string, signerID int, message string, variable string) string {
	messageDigest := sha256.Sum256([]byte(message))
	return fmt.Sprintf("dualms/sign1/session=%s/message=%x/signer=%d/%s", sessionID, messageDigest, signerID, variable)
}
