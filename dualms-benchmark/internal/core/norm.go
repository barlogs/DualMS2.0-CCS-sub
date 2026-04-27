package core

import (
	"fmt"
	"math/big"

	"github.com/tuneinsight/lattigo/v5/ring"
)

func CenteredCoeff(coeff uint64, modulus uint64) (*big.Int, error) {
	if modulus == 0 {
		return nil, fmt.Errorf("modulus must be non-zero")
	}

	normalized := new(big.Int).SetUint64(coeff % modulus)
	return centerBigIntRingtail(normalized, new(big.Int).SetUint64(modulus)), nil
}

func RingElementNorm2(r *ring.Ring, element RingElement) (*big.Int, error) {
	if err := validateNormPoly(r, element.Poly, element.Label); err != nil {
		return nil, err
	}
	return polyNorm2(r, element.Poly)
}

func RingVectorNorm2(r *ring.Ring, vector RingVector) (*big.Int, error) {
	if r == nil {
		return nil, fmt.Errorf("ring is nil")
	}
	if len(vector.Data) == 0 {
		return nil, fmt.Errorf("vector %q has no elements", vector.Label)
	}

	sum := big.NewInt(0)
	for i, poly := range vector.Data {
		if err := validateNormPoly(r, poly, fmt.Sprintf("%s[%d]", vector.Label, i)); err != nil {
			return nil, err
		}
		polyNorm, err := polyNorm2(r, poly)
		if err != nil {
			return nil, err
		}
		sum.Add(sum, polyNorm)
	}

	return sum, nil
}

func polyNorm2(r *ring.Ring, poly ring.Poly) (*big.Int, error) {
	coeffs := make([]*big.Int, r.N())
	r.PolyToBigint(poly, 1, coeffs)

	modulus := r.Modulus()
	sum := big.NewInt(0)
	for _, coeff := range coeffs {
		centered := centerBigIntRingtail(coeff, modulus)
		sum.Add(sum, new(big.Int).Mul(centered, centered))
	}

	return sum, nil
}

func validateNormPoly(r *ring.Ring, poly ring.Poly, label string) error {
	if r == nil {
		return fmt.Errorf("ring is nil")
	}
	if poly.N() == 0 {
		return fmt.Errorf("poly %q is empty", label)
	}
	if poly.N() != r.N() {
		return fmt.Errorf("poly %q has degree %d, want %d", label, poly.N(), r.N())
	}
	if poly.Level() < r.Level() {
		return fmt.Errorf("poly %q has level %d, want at least %d", label, poly.Level(), r.Level())
	}
	return nil
}

func centerBigIntRingtail(coeff *big.Int, modulus *big.Int) *big.Int {
	centered := new(big.Int).Set(coeff)
	halfModulus := new(big.Int).Rsh(new(big.Int).Set(modulus), 1)
	if centered.Cmp(halfModulus) > 0 {
		centered.Sub(centered, modulus)
	}
	return centered
}
