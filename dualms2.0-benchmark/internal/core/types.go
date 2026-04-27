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

type SecretKey struct {
	SignerID int
	S        RingVector
}

type PublicKey struct {
	SignerID int
	T        RingVector
}

type RoundedElement struct {
	Coeffs  []*big.Int
	Modulus uint64
	Shift   int
	Label   string
}

type RoundedVector struct {
	Data    [][]*big.Int
	Modulus uint64
	Shift   int
	Label   string
}

type RoundedResidueVector struct {
	Data    [][]*big.Int
	Modulus uint64
	Shift   int
	Label   string
}

type AggregationCoefficient struct {
	SignerID int
	A        Challenge
}

type AggPublicKey struct {
	TildeTExact          RingVector
	TildeTRounded        RoundedVector
	TildeTRoundedResidue [][]*big.Int
}

type KeyAggResult struct {
	PublicKeys []PublicKey
	Coeffs     map[int]AggregationCoefficient
	AggPK      AggPublicKey
	SignerIDs  []int
}

type CommitmentMaterial struct {
	BVec RingVector
	BMat RingMatrix
	BBar RingMatrix
}

type Round1Msg struct {
	SignerID int
	W        RingVector
}

type Round1State struct {
	SignerID         int
	SK               SecretKey
	AggPK            AggPublicKey
	Message          string
	SessionID        string
	Coeff            AggregationCoefficient
	KeyAggPublicKeys []PublicKey
	KeyAggSignerIDs  []int
	Y                RingVector
	R                RingElement
	RBar             RingVector
	W                RingVector
}

type AggregatedRound1 struct {
	WAggExact          RingVector
	WAggRounded        RoundedVector
	WAggRoundedResidue RoundedResidueVector
}

type PartialSig struct {
	SignerID int
	C        Challenge
	Z1       RingVector
	R        RingElement
	RBar1    RingVector
	Aborted  bool
}

type AggSig struct {
	C         Challenge
	ZAgg      RingVector
	RAgg      RingElement
	RBarAgg   RingVector
	H         RoundedResidueVector
	SignerIDs []int
}

type PublicParams struct {
	SecurityBits            int
	Description             string
	Ring                    *ring.Ring
	LogN                    int
	Q                       uint64
	PaperQ                  uint64
	PaperQLogCeil           int
	LDim                    int
	MDim                    int
	NuB                     int
	NuW                     int
	QNuBFloor               uint64
	QNuBNearest             uint64
	QNuWFloor               uint64
	QNuWNearest             uint64
	Eta                     int
	ChallengeKappa          int
	Sigma                   float64
	GaussianBound           float64
	RejectionM              float64
	TargetRejectionMn       float64
	SamplingSeed            []byte
	ZHNormBoundSquared      *big.Int
	RAggNormBoundSquared    *big.Int
	RBarAggNormBoundSquared *big.Int
	EnforceNormBounds       bool
	A                       RingMatrix
	ABar                    RingMatrix
}

func NewRingElement(r *ring.Ring, label string) RingElement {
	return RingElement{
		Poly:  r.NewPoly(),
		Label: label,
	}
}

func NewRingVector(r *ring.Ring, length int, label string) RingVector {
	vec := make(structs.Vector[ring.Poly], length)
	for i := range vec {
		vec[i] = r.NewPoly()
	}
	return RingVector{
		Data:  vec,
		Label: label,
	}
}

func NewRingMatrix(r *ring.Ring, rows int, cols int, label string) RingMatrix {
	matrix := make(structs.Matrix[ring.Poly], rows)
	for i := range matrix {
		matrix[i] = make([]ring.Poly, cols)
		for j := range matrix[i] {
			matrix[i][j] = r.NewPoly()
		}
	}
	return RingMatrix{
		Data:  matrix,
		Label: label,
	}
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

func (e RoundedElement) String() string {
	if e.Label != "" {
		return e.Label
	}
	return fmt.Sprintf("rounded-poly[len=%d,mod=%d,shift=%d]", len(e.Coeffs), e.Modulus, e.Shift)
}

func (v RoundedVector) String() string {
	if v.Label != "" {
		return v.Label
	}
	return fmt.Sprintf("rounded-vector[len=%d,mod=%d,shift=%d]", len(v.Data), v.Modulus, v.Shift)
}

func (v RoundedResidueVector) String() string {
	if v.Label != "" {
		return v.Label
	}
	return fmt.Sprintf("rounded-residue-vector[len=%d,mod=%d,shift=%d]", len(v.Data), v.Modulus, v.Shift)
}

func (c Challenge) Equal(other Challenge) bool {
	if c.Value.Poly.N() == 0 || other.Value.Poly.N() == 0 {
		return c.Value.Label == other.Value.Label
	}
	return c.Value.Poly.Equal(&other.Value.Poly)
}
