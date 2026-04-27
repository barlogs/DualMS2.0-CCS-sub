package core

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	"github.com/tuneinsight/lattigo/v5/ring"
)

type RejSampResult struct {
	Z         RingVector
	Accepted  bool
	LogAccept float64
}

func Sign2(pp PublicParams, st Round1State, allMsgs []Round1Msg) (PartialSig, error) {
	if err := validateSign2State(pp, st); err != nil {
		return PartialSig{}, err
	}

	wAgg, err := AggregateRound1Messages(pp, st, allMsgs)
	if err != nil {
		return PartialSig{}, err
	}
	c := HSigRounded(pp, wAgg.WAggRounded, st.Message, st.AggPK)

	aS, err := PolyVectorScalarMul(pp.Ring, st.Coeff.A.Value, st.SK.S, fmt.Sprintf("a_%d_s_%d", st.SignerID, st.SignerID))
	if err != nil {
		return PartialSig{}, fmt.Errorf("compute a_%d*s_%d: %w", st.SignerID, st.SignerID, err)
	}
	shift, err := PolyVectorScalarMul(pp.Ring, c.Value, aS, fmt.Sprintf("a_%d_c_s_%d", st.SignerID, st.SignerID))
	if err != nil {
		return PartialSig{}, fmt.Errorf("compute a_%d*c*s_%d: %w", st.SignerID, st.SignerID, err)
	}

	z, accepted, err := sign2RejSamp(pp, st, c, shift)
	if err != nil {
		return PartialSig{}, fmt.Errorf("run RejSamp for signer %d: %w", st.SignerID, err)
	}
	if !accepted {
		return PartialSig{
			SignerID: st.SignerID,
			C:        c,
			Aborted:  true,
		}, nil
	}

	z1, err := vectorPrefix(pp.Ring, z, pp.LDim, fmt.Sprintf("z1_%d", st.SignerID))
	if err != nil {
		return PartialSig{}, fmt.Errorf("truncate z for signer %d: %w", st.SignerID, err)
	}
	rBar1, err := vectorPrefix(pp.Ring, st.RBar, pp.LDim, fmt.Sprintf("rbar1_%d", st.SignerID))
	if err != nil {
		return PartialSig{}, fmt.Errorf("truncate rbar for signer %d: %w", st.SignerID, err)
	}

	return PartialSig{
		SignerID: st.SignerID,
		C:        c,
		Z1:       z1,
		R:        st.R,
		RBar1:    rBar1,
		Aborted:  false,
	}, nil
}

func AggregateRound1Messages(pp PublicParams, st Round1State, allMsgs []Round1Msg) (AggregatedRound1, error) {
	return aggregateRound1MessagesWithExpectedSignerSet(pp, st.KeyAggSignerIDs, allMsgs, st.SignerID, &st.W)
}

func aggregateRound1MessagesForAggregation(pp PublicParams, signerIDs []int, allMsgs []Round1Msg) (AggregatedRound1, error) {
	return aggregateRound1MessagesWithExpectedSignerSet(pp, signerIDs, allMsgs, -1, nil)
}

func aggregateRound1MessagesWithExpectedSignerSet(pp PublicParams, signerIDs []int, allMsgs []Round1Msg, localSignerID int, localW *RingVector) (AggregatedRound1, error) {
	if pp.QNuWFloor == 0 {
		return AggregatedRound1{}, fmt.Errorf("q_nu_w floor is zero for parameter set %q; Sign2 under the paper floor definition is not implemented for this profile", pp.Description)
	}
	if len(allMsgs) == 0 {
		return AggregatedRound1{}, fmt.Errorf("no round 1 messages")
	}

	expected := make(map[int]struct{}, len(signerIDs))
	for _, signerID := range signerIDs {
		if _, ok := expected[signerID]; ok {
			return AggregatedRound1{}, fmt.Errorf("round 1 state has duplicate key aggregation signer id: %d", signerID)
		}
		expected[signerID] = struct{}{}
	}

	seen := make(map[int]struct{}, len(allMsgs))
	wAggExact := NewRingVector(pp.Ring, pp.MDim, fmt.Sprintf("w_agg_exact_%d_msgs", len(allMsgs)))
	for _, msg := range allMsgs {
		if _, ok := seen[msg.SignerID]; ok {
			return AggregatedRound1{}, fmt.Errorf("duplicate round 1 message for signer %d", msg.SignerID)
		}
		if _, ok := expected[msg.SignerID]; !ok {
			return AggregatedRound1{}, fmt.Errorf("round 1 message from signer %d is not in key aggregation signer set", msg.SignerID)
		}
		if len(msg.W.Data) != pp.MDim {
			return AggregatedRound1{}, fmt.Errorf("round 1 message for signer %d has w dimension %d, want %d", msg.SignerID, len(msg.W.Data), pp.MDim)
		}
		if localW != nil && msg.SignerID == localSignerID && !ringVectorsEqual(pp.Ring, msg.W, *localW) {
			return AggregatedRound1{}, fmt.Errorf("round 1 message for signer %d does not match local round 1 state", msg.SignerID)
		}

		var err error
		wAggExact, err = VectorAddRq(pp.Ring, wAggExact, msg.W, "w_agg_exact")
		if err != nil {
			return AggregatedRound1{}, fmt.Errorf("accumulate w for signer %d: %w", msg.SignerID, err)
		}
		seen[msg.SignerID] = struct{}{}
	}

	if localW != nil {
		if _, ok := seen[localSignerID]; !ok {
			return AggregatedRound1{}, fmt.Errorf("missing local round 1 message for signer %d", localSignerID)
		}
	}
	if len(seen) != len(expected) {
		return AggregatedRound1{}, fmt.Errorf("round 1 message signer set has size %d, want %d", len(seen), len(expected))
	}

	wAggRounded, err := RoundRingVectorToRounded(pp.Ring, wAggExact, pp.NuW, pp.QNuWFloor, "w_agg_round")
	if err != nil {
		return AggregatedRound1{}, fmt.Errorf("round aggregate commitment: %w", err)
	}
	wAggResidue, err := RoundedVectorToResidue(wAggRounded, "w_agg_round_residue")
	if err != nil {
		return AggregatedRound1{}, fmt.Errorf("reduce aggregate commitment into Z_q_nu_w: %w", err)
	}

	return AggregatedRound1{
		WAggExact:          wAggExact,
		WAggRounded:        wAggRounded,
		WAggRoundedResidue: wAggResidue,
	}, nil
}

func validateSign2State(pp PublicParams, st Round1State) error {
	if pp.Ring == nil {
		return fmt.Errorf("public params have nil ring")
	}
	if len(st.AggPK.TildeTExact.Data) != pp.MDim {
		return fmt.Errorf("round 1 state exact aggregate public key dimension is %d, want %d", len(st.AggPK.TildeTExact.Data), pp.MDim)
	}
	if len(st.AggPK.TildeTRounded.Data) != pp.MDim {
		return fmt.Errorf("round 1 state rounded aggregate public key dimension is %d, want %d", len(st.AggPK.TildeTRounded.Data), pp.MDim)
	}
	if len(st.SK.S.Data) != pp.LDim+pp.MDim {
		return fmt.Errorf("round 1 state secret key dimension is %d, want %d", len(st.SK.S.Data), pp.LDim+pp.MDim)
	}
	if len(st.Y.Data) != pp.LDim+pp.MDim {
		return fmt.Errorf("round 1 state y dimension is %d, want %d", len(st.Y.Data), pp.LDim+pp.MDim)
	}
	if len(st.RBar.Data) != pp.LDim+pp.MDim {
		return fmt.Errorf("round 1 state rbar dimension is %d, want %d", len(st.RBar.Data), pp.LDim+pp.MDim)
	}
	if len(st.W.Data) != pp.MDim {
		return fmt.Errorf("round 1 state w dimension is %d, want %d", len(st.W.Data), pp.MDim)
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

func RejSamp(pp PublicParams, y RingVector, shift RingVector, uniform float64, label string) (RejSampResult, error) {
	if err := validateRejSampParams(pp, y, shift, uniform); err != nil {
		return RejSampResult{}, err
	}

	z, err := VectorAddRq(pp.Ring, y, shift, label)
	if err != nil {
		return RejSampResult{}, fmt.Errorf("compute rejection candidate z = y + v: %w", err)
	}

	logAccept, err := rejSampLogAccept(pp, z, shift)
	if err != nil {
		return RejSampResult{}, err
	}

	accepted := logAccept >= 0 || math.Log(uniform) <= logAccept
	return RejSampResult{
		Z:         z,
		Accepted:  accepted,
		LogAccept: logAccept,
	}, nil
}

func RingVectorDotCentered(r *ring.Ring, lhs RingVector, rhs RingVector) (*big.Int, error) {
	if r == nil {
		return nil, fmt.Errorf("ring is nil")
	}
	if len(lhs.Data) == 0 {
		return nil, fmt.Errorf("left vector %q has no elements", lhs.Label)
	}
	if len(rhs.Data) == 0 {
		return nil, fmt.Errorf("right vector %q has no elements", rhs.Label)
	}
	if len(lhs.Data) != len(rhs.Data) {
		return nil, fmt.Errorf("vector dimension mismatch: left has %d elements, right has %d", len(lhs.Data), len(rhs.Data))
	}

	modulus := r.Modulus()
	sum := big.NewInt(0)
	for i := range lhs.Data {
		if err := validateNormPoly(r, lhs.Data[i], fmt.Sprintf("%s[%d]", lhs.Label, i)); err != nil {
			return nil, err
		}
		if err := validateNormPoly(r, rhs.Data[i], fmt.Sprintf("%s[%d]", rhs.Label, i)); err != nil {
			return nil, err
		}

		leftCoeffs := make([]*big.Int, r.N())
		rightCoeffs := make([]*big.Int, r.N())
		r.PolyToBigint(lhs.Data[i], 1, leftCoeffs)
		r.PolyToBigint(rhs.Data[i], 1, rightCoeffs)

		for j := 0; j < r.N(); j++ {
			left := centerBigIntRingtail(leftCoeffs[j], modulus)
			right := centerBigIntRingtail(rightCoeffs[j], modulus)
			sum.Add(sum, new(big.Int).Mul(left, right))
		}
	}

	return sum, nil
}

func sign2RejSamp(pp PublicParams, st Round1State, c Challenge, shift RingVector) (RingVector, bool, error) {
	if pp.RejectionM == 0 {
		z, err := VectorAddRq(pp.Ring, st.Y, shift, fmt.Sprintf("z_%d_no_reject", st.SignerID))
		if err != nil {
			return RingVector{}, false, err
		}
		return z, true, nil
	}

	uniform := rejectionUniform01(pp, st, c)
	result, err := RejSamp(pp, st.Y, shift, uniform, fmt.Sprintf("z_%d_rejsamp", st.SignerID))
	if err != nil {
		return RingVector{}, false, err
	}
	return result.Z, result.Accepted, nil
}

func rejSampLogAccept(pp PublicParams, z RingVector, shift RingVector) (float64, error) {
	dot, err := RingVectorDotCentered(pp.Ring, z, shift)
	if err != nil {
		return 0, fmt.Errorf("compute <z,v>: %w", err)
	}
	shiftNorm2, err := RingVectorDotCentered(pp.Ring, shift, shift)
	if err != nil {
		return 0, fmt.Errorf("compute ||v||^2: %w", err)
	}

	dotFloat, err := bigIntToFiniteFloat64(dot, "<z,v>")
	if err != nil {
		return 0, err
	}
	shiftNormFloat, err := bigIntToFiniteFloat64(shiftNorm2, "||v||^2")
	if err != nil {
		return 0, err
	}

	return ((-2*dotFloat)+shiftNormFloat)/(2*pp.Sigma*pp.Sigma) - math.Log(pp.RejectionM), nil
}

func validateRejSampParams(pp PublicParams, y RingVector, shift RingVector, uniform float64) error {
	if pp.Ring == nil {
		return fmt.Errorf("public params have nil ring")
	}
	if pp.Sigma <= 0 || math.IsNaN(pp.Sigma) || math.IsInf(pp.Sigma, 0) {
		return fmt.Errorf("invalid rejection sigma: %v", pp.Sigma)
	}
	if pp.RejectionM < 1 || math.IsNaN(pp.RejectionM) || math.IsInf(pp.RejectionM, 0) {
		return fmt.Errorf("invalid rejection M: %v, want >= 1", pp.RejectionM)
	}
	if uniform <= 0 || uniform >= 1 || math.IsNaN(uniform) {
		return fmt.Errorf("invalid rejection uniform sample: %v, want 0 < u < 1", uniform)
	}
	if len(y.Data) == 0 {
		return fmt.Errorf("rejection y vector has no elements")
	}
	if len(shift.Data) == 0 {
		return fmt.Errorf("rejection shift vector has no elements")
	}
	if len(y.Data) != len(shift.Data) {
		return fmt.Errorf("rejection vector dimension mismatch: y has %d elements, shift has %d", len(y.Data), len(shift.Data))
	}
	return nil
}

func rejectionUniform01(pp PublicParams, st Round1State, c Challenge) float64 {
	input := &hashInput{}
	input.writeString("Rounding01/RejSamp/uniform")
	input.writePublicParameterFingerprint(pp)
	input.writeString(st.SessionID)
	input.writeInt(st.SignerID)
	input.writeString(st.Message)
	input.writeRingVector(st.W)
	input.writeRoundedVectorStandard(st.AggPK.TildeTRounded)
	input.writePoly(c.Value.Poly)

	digest := sha256.Sum256(input.Bytes())
	raw := binary.BigEndian.Uint64(digest[:8])
	return math.Ldexp(float64(raw)+0.5, -64)
}

func bigIntToFiniteFloat64(value *big.Int, label string) (float64, error) {
	asFloat, _ := new(big.Float).SetInt(value).Float64()
	if math.IsInf(asFloat, 0) || math.IsNaN(asFloat) {
		return 0, fmt.Errorf("%s is outside finite float64 range", label)
	}
	return asFloat, nil
}

func vectorPrefix(r *ring.Ring, vector RingVector, prefixLen int, label string) (RingVector, error) {
	if r == nil {
		return RingVector{}, fmt.Errorf("ring is nil")
	}
	if prefixLen < 0 || prefixLen > len(vector.Data) {
		return RingVector{}, fmt.Errorf("invalid prefix length %d for vector of dimension %d", prefixLen, len(vector.Data))
	}
	prefix := NewRingVector(r, prefixLen, label)
	for i := 0; i < prefixLen; i++ {
		prefix.Data[i] = r.NewPoly()
		prefix.Data[i].Copy(vector.Data[i])
	}
	return prefix, nil
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
