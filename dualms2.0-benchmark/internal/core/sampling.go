package core

import (
	"crypto/sha256"
	"fmt"

	"github.com/tuneinsight/lattigo/v5/ring"
	"github.com/tuneinsight/lattigo/v5/utils/sampling"
	"github.com/tuneinsight/lattigo/v5/utils/structs"
)

func SampleUniformMatrix(r *ring.Ring, seed []byte, domain string, rows int, cols int, label string) (RingMatrix, error) {
	if rows <= 0 || cols <= 0 {
		return RingMatrix{}, fmt.Errorf("invalid matrix dimensions: rows=%d cols=%d", rows, cols)
	}

	prng, err := newFixedPRNG(seed, domain)
	if err != nil {
		return RingMatrix{}, fmt.Errorf("create uniform sampler prng: %w", err)
	}

	sampler := ring.NewUniformSampler(prng, r)
	matrix := make(structs.Matrix[ring.Poly], rows)
	for i := range matrix {
		matrix[i] = make([]ring.Poly, cols)
		for j := range matrix[i] {
			matrix[i][j] = sampler.ReadNew()
		}
	}

	return RingMatrix{
		Data:  matrix,
		Label: label,
	}, nil
}

func SampleGaussianPoly(pp PublicParams, label string) (RingElement, error) {
	if err := validateSamplingParams(pp); err != nil {
		return RingElement{}, err
	}

	sampler, err := newGaussianSampler(pp, "gaussian-poly:"+label)
	if err != nil {
		return RingElement{}, err
	}

	return RingElement{
		Poly:  sampler.ReadNew(),
		Label: label,
	}, nil
}

func SampleGaussianVector(pp PublicParams, length int, label string) (RingVector, error) {
	if err := validateSamplingParams(pp); err != nil {
		return RingVector{}, err
	}
	if length <= 0 {
		return RingVector{}, fmt.Errorf("invalid vector length: %d", length)
	}

	sampler, err := newGaussianSampler(pp, "gaussian-vector:"+label)
	if err != nil {
		return RingVector{}, err
	}

	return sampleVector(pp.Ring, length, sampler, label), nil
}

func SampleSmallPolyEta1(pp PublicParams, label string) (RingElement, error) {
	if err := validateSamplingParams(pp); err != nil {
		return RingElement{}, err
	}
	if pp.Eta != 1 {
		return RingElement{}, fmt.Errorf("SampleSmallPolyEta1 requires eta=1, got eta=%d", pp.Eta)
	}

	sampler, err := newTernaryEta1Sampler(pp, "small-poly:"+label)
	if err != nil {
		return RingElement{}, err
	}

	return RingElement{
		Poly:  sampler.ReadNew(),
		Label: label,
	}, nil
}

func SampleSmallVectorEta1(pp PublicParams, length int, label string) (RingVector, error) {
	if err := validateSamplingParams(pp); err != nil {
		return RingVector{}, err
	}
	if pp.Eta != 1 {
		return RingVector{}, fmt.Errorf("SampleSmallVectorEta1 requires eta=1, got eta=%d", pp.Eta)
	}
	if length <= 0 {
		return RingVector{}, fmt.Errorf("invalid vector length: %d", length)
	}

	sampler, err := newTernaryEta1Sampler(pp, "small-vector:"+label)
	if err != nil {
		return RingVector{}, err
	}

	return sampleVector(pp.Ring, length, sampler, label), nil
}

func sampleVector(r *ring.Ring, length int, sampler ring.Sampler, label string) RingVector {
	vector := make(structs.Vector[ring.Poly], length)
	for i := range vector {
		vector[i] = sampler.ReadNew()
	}
	return RingVector{
		Data:  vector,
		Label: label,
	}
}

func newGaussianSampler(pp PublicParams, domain string) (*ring.GaussianSampler, error) {
	prng, err := newFixedPRNG(pp.SamplingSeed, domain)
	if err != nil {
		return nil, err
	}

	params := ring.DiscreteGaussian{
		Sigma: pp.Sigma,
		Bound: pp.GaussianBound,
	}
	return ring.NewGaussianSampler(prng, pp.Ring, params, false), nil
}

func newTernaryEta1Sampler(pp PublicParams, domain string) (*ring.TernarySampler, error) {
	prng, err := newFixedPRNG(pp.SamplingSeed, domain)
	if err != nil {
		return nil, err
	}

	return ring.NewTernarySampler(prng, pp.Ring, ring.Ternary{P: 2.0 / 3.0}, false)
}

func validateSamplingParams(pp PublicParams) error {
	if pp.Ring == nil {
		return fmt.Errorf("public params have nil ring")
	}
	if pp.Sigma <= 0 {
		return fmt.Errorf("invalid sigma: %f", pp.Sigma)
	}
	if pp.GaussianBound <= 0 {
		return fmt.Errorf("invalid gaussian bound: %f", pp.GaussianBound)
	}
	return nil
}

func newFixedPRNG(seed []byte, domain string) (*sampling.KeyedPRNG, error) {
	digest := sha256.New()
	digest.Write(copySamplingSeed(seed))
	digest.Write([]byte{0})
	digest.Write([]byte(domain))
	return sampling.NewKeyedPRNG(digest.Sum(nil))
}

func copySamplingSeed(seed []byte) []byte {
	if len(seed) == 0 {
		return make([]byte, defaultSeedSize)
	}
	seedCopy := make([]byte, len(seed))
	copy(seedCopy, seed)
	return seedCopy
}
