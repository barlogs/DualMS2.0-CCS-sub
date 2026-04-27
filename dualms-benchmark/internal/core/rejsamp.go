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

func acceptanceProbabilityFromLog(logAccept float64) float64 {
	if logAccept >= 0 {
		return 1
	}
	return math.Exp(logAccept)
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
	input.writeString("DualMS/RejSamp/uniform")
	input.writePublicParameterFingerprint(pp)
	input.writeString(st.SessionID)
	input.writeInt(st.SignerID)
	input.writeString(st.Message)
	input.writeRingVector(st.W)
	input.writeRingVector(st.TAgg.TildeT)
	input.writePoly(c.Value.Poly)

	digest := sha256.Sum256(input.Bytes())
	raw := binary.BigEndian.Uint64(digest[:8])
	return math.Ldexp(float64(raw>>11)+0.5, -53)
}

func bigIntToFiniteFloat64(value *big.Int, label string) (float64, error) {
	asFloat, _ := new(big.Float).SetInt(value).Float64()
	if math.IsInf(asFloat, 0) || math.IsNaN(asFloat) {
		return 0, fmt.Errorf("%s is outside finite float64 range", label)
	}
	return asFloat, nil
}
