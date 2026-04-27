package core

import (
	"fmt"
	"math/big"

	"github.com/tuneinsight/lattigo/v5/ring"
)

func RoundRingElementToRounded(r *ring.Ring, element RingElement, shift int, roundedModulus uint64, label string) (RoundedElement, error) {
	if err := validateRoundingPoly(r, element.Poly, element.Label); err != nil {
		return RoundedElement{}, err
	}
	if roundedModulus == 0 {
		return RoundedElement{}, fmt.Errorf("rounded modulus must be non-zero")
	}
	if shift <= 0 {
		return RoundedElement{}, fmt.Errorf("rounding shift must be positive")
	}

	coeffs := make([]*big.Int, r.N())
	r.PolyToBigint(element.Poly, 1, coeffs)

	rounded := make([]*big.Int, len(coeffs))
	for i, coeff := range coeffs {
		rounded[i] = roundCoeffByPow2(coeff, shift)
	}

	return RoundedElement{
		Coeffs:  rounded,
		Modulus: roundedModulus,
		Shift:   shift,
		Label:   label,
	}, nil
}

func RoundRingVectorToRounded(r *ring.Ring, vector RingVector, shift int, roundedModulus uint64, label string) (RoundedVector, error) {
	if r == nil {
		return RoundedVector{}, fmt.Errorf("ring is nil")
	}
	if roundedModulus == 0 {
		return RoundedVector{}, fmt.Errorf("rounded modulus must be non-zero")
	}
	if shift <= 0 {
		return RoundedVector{}, fmt.Errorf("rounding shift must be positive")
	}
	if len(vector.Data) == 0 {
		return RoundedVector{}, fmt.Errorf("vector %q has no elements", vector.Label)
	}

	data := make([][]*big.Int, len(vector.Data))
	for i, poly := range vector.Data {
		rounded, err := RoundRingElementToRounded(r, RingElement{Poly: poly, Label: fmt.Sprintf("%s[%d]", vector.Label, i)}, shift, roundedModulus, fmt.Sprintf("%s[%d]", label, i))
		if err != nil {
			return RoundedVector{}, err
		}
		data[i] = rounded.Coeffs
	}

	return RoundedVector{
		Data:    data,
		Modulus: roundedModulus,
		Shift:   shift,
		Label:   label,
	}, nil
}

func RoundedElementStandardReps(element RoundedElement) ([]*big.Int, error) {
	if err := validateRoundedElement(element); err != nil {
		return nil, err
	}

	standard := make([]*big.Int, len(element.Coeffs))
	modulus := new(big.Int).SetUint64(element.Modulus)
	for i, coeff := range element.Coeffs {
		standard[i] = new(big.Int).Mod(new(big.Int).Set(coeff), modulus)
		if standard[i].Sign() < 0 {
			standard[i].Add(standard[i], modulus)
		}
	}
	return standard, nil
}

func RoundedVectorStandardReps(vector RoundedVector) ([][]*big.Int, error) {
	if err := validateRoundedVector(vector); err != nil {
		return nil, err
	}

	standard := make([][]*big.Int, len(vector.Data))
	modulus := new(big.Int).SetUint64(vector.Modulus)
	for i, coeffs := range vector.Data {
		standard[i] = make([]*big.Int, len(coeffs))
		for j, coeff := range coeffs {
			standard[i][j] = new(big.Int).Mod(new(big.Int).Set(coeff), modulus)
			if standard[i][j].Sign() < 0 {
				standard[i][j].Add(standard[i][j], modulus)
			}
		}
	}
	return standard, nil
}

func RoundRingVectorToResidue(r *ring.Ring, vector RingVector, shift int, roundedModulus uint64, label string) (RoundedResidueVector, error) {
	rounded, err := RoundRingVectorToRounded(r, vector, shift, roundedModulus, label+"_quotient")
	if err != nil {
		return RoundedResidueVector{}, err
	}
	return roundedVectorToResidue(rounded, label)
}

func RoundedVectorToResidue(vector RoundedVector, label string) (RoundedResidueVector, error) {
	return roundedVectorToResidue(vector, label)
}

func EmbedRoundedElementToRq(r *ring.Ring, element RoundedElement, label string) (RingElement, error) {
	if r == nil {
		return RingElement{}, fmt.Errorf("ring is nil")
	}
	if err := validateRoundedElement(element); err != nil {
		return RingElement{}, err
	}

	out := NewRingElement(r, label)
	qBig := r.Modulus()
	scale := new(big.Int).Lsh(big.NewInt(1), uint(element.Shift))

	for i, coeff := range element.Coeffs {
		scaled := new(big.Int).Mul(new(big.Int).Set(coeff), scale)
		scaled.Mod(scaled, qBig)
		out.Poly.Coeffs[0][i] = scaled.Uint64()
	}
	return out, nil
}

func EmbedRoundedVectorToRq(r *ring.Ring, vector RoundedVector, label string) (RingVector, error) {
	if r == nil {
		return RingVector{}, fmt.Errorf("ring is nil")
	}
	if err := validateRoundedVector(vector); err != nil {
		return RingVector{}, err
	}

	out := NewRingVector(r, len(vector.Data), label)
	qBig := r.Modulus()
	scale := new(big.Int).Lsh(big.NewInt(1), uint(vector.Shift))
	for i, coeffs := range vector.Data {
		for j, coeff := range coeffs {
			scaled := new(big.Int).Mul(new(big.Int).Set(coeff), scale)
			scaled.Mod(scaled, qBig)
			out.Data[i].Coeffs[0][j] = scaled.Uint64()
		}
	}
	return out, nil
}

func ScaleRoundedElementForNorm(element RoundedElement) ([]*big.Int, error) {
	if err := validateRoundedElement(element); err != nil {
		return nil, err
	}

	centered, err := centeredRoundedCoeffs(element.Coeffs, element.Modulus)
	if err != nil {
		return nil, err
	}
	scale := new(big.Int).Lsh(big.NewInt(1), uint(element.Shift))

	scaled := make([]*big.Int, len(centered))
	for i, coeff := range centered {
		scaled[i] = new(big.Int).Mul(coeff, scale)
	}
	return scaled, nil
}

func ScaleRoundedVectorForNorm(vector RoundedVector) ([]*big.Int, error) {
	if err := validateRoundedVector(vector); err != nil {
		return nil, err
	}

	scale := new(big.Int).Lsh(big.NewInt(1), uint(vector.Shift))
	scaled := make([]*big.Int, 0, len(vector.Data)*len(vector.Data[0]))
	for _, coeffs := range vector.Data {
		centered, err := centeredRoundedCoeffs(coeffs, vector.Modulus)
		if err != nil {
			return nil, err
		}
		for _, coeff := range centered {
			scaled = append(scaled, new(big.Int).Mul(coeff, scale))
		}
	}
	return scaled, nil
}

func AddRoundedResiduesMod(lhs RoundedResidueVector, rhs RoundedResidueVector, label string) (RoundedResidueVector, error) {
	if err := validateRoundedResidueVectorPair(lhs, rhs); err != nil {
		return RoundedResidueVector{}, err
	}

	modulus := new(big.Int).SetUint64(lhs.Modulus)
	sum := make([][]*big.Int, len(lhs.Data))
	for i := range lhs.Data {
		sum[i] = make([]*big.Int, len(lhs.Data[i]))
		for j := range lhs.Data[i] {
			sum[i][j] = new(big.Int).Add(lhs.Data[i][j], rhs.Data[i][j])
			sum[i][j].Mod(sum[i][j], modulus)
			if sum[i][j].Sign() < 0 {
				sum[i][j].Add(sum[i][j], modulus)
			}
		}
	}

	return RoundedResidueVector{
		Data:    sum,
		Modulus: lhs.Modulus,
		Shift:   lhs.Shift,
		Label:   label,
	}, nil
}

func SubRoundedResiduesMod(lhs RoundedResidueVector, rhs RoundedResidueVector, label string) (RoundedResidueVector, error) {
	if err := validateRoundedResidueVectorPair(lhs, rhs); err != nil {
		return RoundedResidueVector{}, err
	}

	modulus := new(big.Int).SetUint64(lhs.Modulus)
	diff := make([][]*big.Int, len(lhs.Data))
	for i := range lhs.Data {
		diff[i] = make([]*big.Int, len(lhs.Data[i]))
		for j := range lhs.Data[i] {
			diff[i][j] = new(big.Int).Sub(lhs.Data[i][j], rhs.Data[i][j])
			diff[i][j].Mod(diff[i][j], modulus)
			if diff[i][j].Sign() < 0 {
				diff[i][j].Add(diff[i][j], modulus)
			}
		}
	}

	return RoundedResidueVector{
		Data:    diff,
		Modulus: lhs.Modulus,
		Shift:   lhs.Shift,
		Label:   label,
	}, nil
}

func validateRoundingPoly(r *ring.Ring, poly ring.Poly, label string) error {
	if r == nil {
		return fmt.Errorf("ring is nil")
	}
	if poly.N() == 0 {
		return fmt.Errorf("poly %q is empty", label)
	}
	if poly.N() != r.N() {
		return fmt.Errorf("poly %q has degree %d, want %d", label, poly.N(), r.N())
	}
	return nil
}

func validateRoundedElement(element RoundedElement) error {
	if element.Modulus == 0 {
		return fmt.Errorf("rounded modulus must be non-zero")
	}
	if element.Shift <= 0 {
		return fmt.Errorf("rounded shift must be positive")
	}
	if len(element.Coeffs) == 0 {
		return fmt.Errorf("rounded element %q has no coefficients", element.Label)
	}
	for i, coeff := range element.Coeffs {
		if coeff == nil {
			return fmt.Errorf("rounded element %q coefficient %d is nil", element.Label, i)
		}
	}
	return nil
}

func validateRoundedVector(vector RoundedVector) error {
	if vector.Modulus == 0 {
		return fmt.Errorf("rounded modulus must be non-zero")
	}
	if vector.Shift <= 0 {
		return fmt.Errorf("rounded shift must be positive")
	}
	if len(vector.Data) == 0 {
		return fmt.Errorf("rounded vector %q has no entries", vector.Label)
	}
	for i, coeffs := range vector.Data {
		if len(coeffs) == 0 {
			return fmt.Errorf("rounded vector %q entry %d has no coefficients", vector.Label, i)
		}
		for j, coeff := range coeffs {
			if coeff == nil {
				return fmt.Errorf("rounded vector %q coefficient [%d][%d] is nil", vector.Label, i, j)
			}
		}
	}
	return nil
}

func validateRoundedResidueVector(vector RoundedResidueVector) error {
	if vector.Modulus == 0 {
		return fmt.Errorf("rounded residue modulus must be non-zero")
	}
	if vector.Shift <= 0 {
		return fmt.Errorf("rounded residue shift must be positive")
	}
	if len(vector.Data) == 0 {
		return fmt.Errorf("rounded residue vector %q has no entries", vector.Label)
	}
	for i, coeffs := range vector.Data {
		if len(coeffs) == 0 {
			return fmt.Errorf("rounded residue vector %q entry %d has no coefficients", vector.Label, i)
		}
		for j, coeff := range coeffs {
			if coeff == nil {
				return fmt.Errorf("rounded residue vector %q coefficient [%d][%d] is nil", vector.Label, i, j)
			}
		}
	}
	return nil
}

func validateRoundedResidueVectorPair(lhs RoundedResidueVector, rhs RoundedResidueVector) error {
	if err := validateRoundedResidueVector(lhs); err != nil {
		return err
	}
	if err := validateRoundedResidueVector(rhs); err != nil {
		return err
	}
	if lhs.Modulus != rhs.Modulus {
		return fmt.Errorf("rounded residue modulus mismatch: %d vs %d", lhs.Modulus, rhs.Modulus)
	}
	if lhs.Shift != rhs.Shift {
		return fmt.Errorf("rounded residue shift mismatch: %d vs %d", lhs.Shift, rhs.Shift)
	}
	if len(lhs.Data) != len(rhs.Data) {
		return fmt.Errorf("rounded residue vector length mismatch: %d vs %d", len(lhs.Data), len(rhs.Data))
	}
	for i := range lhs.Data {
		if len(lhs.Data[i]) != len(rhs.Data[i]) {
			return fmt.Errorf("rounded residue vector coefficient length mismatch at entry %d: %d vs %d", i, len(lhs.Data[i]), len(rhs.Data[i]))
		}
	}
	return nil
}

func centeredRoundedCoeffs(coeffs []*big.Int, modulus uint64) ([]*big.Int, error) {
	if modulus == 0 {
		return nil, fmt.Errorf("rounded modulus must be non-zero")
	}
	modulusBig := new(big.Int).SetUint64(modulus)
	centered := make([]*big.Int, len(coeffs))
	for i, coeff := range coeffs {
		if coeff == nil {
			return nil, fmt.Errorf("rounded coefficient %d is nil", i)
		}
		standard := new(big.Int).Mod(new(big.Int).Set(coeff), modulusBig)
		if standard.Sign() < 0 {
			standard.Add(standard, modulusBig)
		}
		centered[i] = centerBigIntRingtail(standard, modulusBig)
	}
	return centered, nil
}

func roundCoeffByPow2(coeff *big.Int, shift int) *big.Int {
	if shift <= 0 {
		return new(big.Int).Set(coeff)
	}
	half := new(big.Int).Lsh(big.NewInt(1), uint(shift-1))
	numerator := new(big.Int).Add(new(big.Int).Set(coeff), half)
	return numerator.Rsh(numerator, uint(shift))
}

func roundedVectorToResidue(vector RoundedVector, label string) (RoundedResidueVector, error) {
	standard, err := RoundedVectorStandardReps(vector)
	if err != nil {
		return RoundedResidueVector{}, err
	}
	return RoundedResidueVector{
		Data:    standard,
		Modulus: vector.Modulus,
		Shift:   vector.Shift,
		Label:   label,
	}, nil
}
