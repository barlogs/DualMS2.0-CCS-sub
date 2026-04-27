package core

import (
	"fmt"
	"math/big"

	"github.com/tuneinsight/lattigo/v5/ring"
	"github.com/tuneinsight/lattigo/v5/utils/structs"
)

type RingElement struct {
	Poly  ring.Poly
	Label string
}

type RingVector struct {
	Data  structs.Vector[ring.Poly]
	Label string
}

type RingMatrix struct {
	Data  structs.Matrix[ring.Poly]
	Label string
}

type Challenge struct {
	Value RingElement
}

type PublicParams struct {
	ParameterSet         string
	SecurityBits         int
	Description          string
	Ring                 *ring.Ring
	LogN                 int
	Q                    uint64
	PaperQ               uint64
	PaperQLogCeil        int
	KDim                 int
	LDim                 int
	LPrimeDim            int
	Eta                  int
	EtaPrime             int
	ChallengeKappa       int
	Gamma                float64
	TParam               float64
	Alpha                float64
	Sigma                float64
	SigmaPrime           float64
	GaussianBound        float64
	GaussianPrimeBound   float64
	RejectionM           float64
	TargetRejectionMn    float64
	SamplingSeed         []byte
	ZAggNormBoundSquared *big.Int
	RAggNormBoundSquared *big.Int
	EAggNormBoundSquared *big.Int
	EnforceNormBounds    bool
	A                    RingMatrix
	ABar                 RingMatrix
}

func (pp PublicParams) SecretDim() int {
	return pp.LDim + pp.KDim
}

func (pp PublicParams) CommitmentRandomDim() int {
	return pp.LPrimeDim + pp.KDim
}

type SecretKey struct {
	SignerID int
	S        RingVector
}

type PublicKey struct {
	SignerID int
	T        RingVector
}

type Signer struct {
	ID int
	SK SecretKey
	PK PublicKey
}

type AggregationCoefficient struct {
	SignerID int
	A        Challenge
}

type AggPublicKey struct {
	TildeT RingVector
}

type KeyAggResult struct {
	PublicKeys []PublicKey
	Coeffs     map[int]AggregationCoefficient
	AggPK      AggPublicKey
	SignerIDs  []int
}

type HComResult struct {
	B    RingMatrix
	BBar RingMatrix
}

type Round1Msg struct {
	SignerID int
	W        RingVector
}

type Round1State struct {
	SignerID         int
	SK               SecretKey
	TAgg             AggPublicKey
	Message          string
	SessionID        string
	Coeff            AggregationCoefficient
	KeyAggPublicKeys []PublicKey
	KeyAggSignerIDs  []int
	Y                RingVector
	R                RingVector
	W                RingVector
}

type PartialSig struct {
	SignerID int
	C        Challenge
	Z        RingVector
	R        RingVector
	Aborted  bool
}

type AggSig struct {
	C         Challenge
	ZAgg      RingVector
	RAgg      RingVector
	EAgg      RingVector
	SignerIDs []int
}

func NewSigner(id int) *Signer {
	return &Signer{ID: id}
}

func NewRingElement(r *ring.Ring, label string) RingElement {
	return RingElement{Poly: r.NewPoly(), Label: label}
}

func NewRingVector(r *ring.Ring, length int, label string) RingVector {
	vec := make(structs.Vector[ring.Poly], length)
	for i := range vec {
		vec[i] = r.NewPoly()
	}
	return RingVector{Data: vec, Label: label}
}

func NewRingMatrix(r *ring.Ring, rows int, cols int, label string) RingMatrix {
	matrix := make(structs.Matrix[ring.Poly], rows)
	for i := range matrix {
		matrix[i] = make([]ring.Poly, cols)
		for j := range matrix[i] {
			matrix[i][j] = r.NewPoly()
		}
	}
	return RingMatrix{Data: matrix, Label: label}
}

func (e RingElement) String() string {
	if e.Label != "" {
		return e.Label
	}
	return fmt.Sprintf("poly[N=%d]", e.Poly.N())
}

func (v RingVector) String() string {
	if v.Label != "" {
		return v.Label
	}
	return fmt.Sprintf("vector[len=%d]", len(v.Data))
}

func (m RingMatrix) String() string {
	if m.Label != "" {
		return m.Label
	}
	return fmt.Sprintf("matrix[rows=%d]", len(m.Data))
}

func (c Challenge) Equal(other Challenge) bool {
	if c.Value.Poly.N() == 0 || other.Value.Poly.N() == 0 {
		return c.Value.Label == other.Value.Label
	}
	return c.Value.Poly.Equal(&other.Value.Poly)
}
