package core

import (
	"fmt"

	"github.com/tuneinsight/lattigo/v5/ring"
	"github.com/tuneinsight/lattigo/v5/utils/structs"
)

func BuildABar(r *ring.Ring, a RingMatrix, label string) (RingMatrix, error) {
	rows, cols, err := MatrixDims(a)
	if err != nil {
		return RingMatrix{}, err
	}

	aBar := NewRingMatrix(r, rows, cols+rows, label)
	for i := range a.Data {
		for j := range a.Data[i] {
			aBar.Data[i][j].Copy(a.Data[i][j])
		}
		aBar.Data[i][cols+i].Copy(r.NewMonomialXi(0))
	}

	return aBar, nil
}

func MatrixVectorMulRq(r *ring.Ring, matrix RingMatrix, vector RingVector, label string) (RingVector, error) {
	rows, cols, err := MatrixDims(matrix)
	if err != nil {
		return RingVector{}, err
	}
	if len(vector.Data) != cols {
		return RingVector{}, fmt.Errorf("matrix-vector dimension mismatch: matrix is %dx%d, vector length is %d", rows, cols, len(vector.Data))
	}

	matrixNTT := make(structs.Matrix[ring.Poly], rows)
	for i := range matrix.Data {
		matrixNTT[i] = make([]ring.Poly, cols)
		for j := range matrix.Data[i] {
			matrixNTT[i][j] = toNTTMontgomeryPoly(r, matrix.Data[i][j])
		}
	}

	vectorNTT := make(structs.Vector[ring.Poly], len(vector.Data))
	for i := range vector.Data {
		vectorNTT[i] = toNTTMontgomeryPoly(r, vector.Data[i])
	}

	result := NewRingVector(r, rows, label)
	for i := range matrixNTT {
		for j := range matrixNTT[i] {
			r.MulCoeffsMontgomeryThenAdd(matrixNTT[i][j], vectorNTT[j], result.Data[i])
		}
		fromNTTMontgomeryPoly(r, result.Data[i])
	}

	return result, nil
}

func VectorAddRq(r *ring.Ring, lhs RingVector, rhs RingVector, label string) (RingVector, error) {
	if len(lhs.Data) == 0 {
		return RingVector{}, fmt.Errorf("left vector %q has no elements", lhs.Label)
	}
	if len(rhs.Data) == 0 {
		return RingVector{}, fmt.Errorf("right vector %q has no elements", rhs.Label)
	}
	if len(lhs.Data) != len(rhs.Data) {
		return RingVector{}, fmt.Errorf("vector dimension mismatch: lhs length is %d, rhs length is %d", len(lhs.Data), len(rhs.Data))
	}

	result := NewRingVector(r, len(lhs.Data), label)
	for i := range lhs.Data {
		r.Add(lhs.Data[i], rhs.Data[i], result.Data[i])
	}

	return result, nil
}

func RingElementAddRq(r *ring.Ring, lhs RingElement, rhs RingElement, label string) (RingElement, error) {
	if r == nil {
		return RingElement{}, fmt.Errorf("ring is nil")
	}
	if lhs.Poly.N() == 0 {
		return RingElement{}, fmt.Errorf("left ring element %q is empty", lhs.Label)
	}
	if rhs.Poly.N() == 0 {
		return RingElement{}, fmt.Errorf("right ring element %q is empty", rhs.Label)
	}

	result := NewRingElement(r, label)
	r.Add(lhs.Poly, rhs.Poly, result.Poly)

	return result, nil
}

func VectorSubRq(r *ring.Ring, lhs RingVector, rhs RingVector, label string) (RingVector, error) {
	if len(lhs.Data) == 0 {
		return RingVector{}, fmt.Errorf("left vector %q has no elements", lhs.Label)
	}
	if len(rhs.Data) == 0 {
		return RingVector{}, fmt.Errorf("right vector %q has no elements", rhs.Label)
	}
	if len(lhs.Data) != len(rhs.Data) {
		return RingVector{}, fmt.Errorf("vector dimension mismatch: lhs length is %d, rhs length is %d", len(lhs.Data), len(rhs.Data))
	}

	result := NewRingVector(r, len(lhs.Data), label)
	for i := range lhs.Data {
		r.Sub(lhs.Data[i], rhs.Data[i], result.Data[i])
	}

	return result, nil
}

func PolyVectorScalarMul(r *ring.Ring, scalar RingElement, vector RingVector, label string) (RingVector, error) {
	if r == nil {
		return RingVector{}, fmt.Errorf("ring is nil")
	}
	if scalar.Poly.N() == 0 {
		return RingVector{}, fmt.Errorf("scalar %q is empty", scalar.Label)
	}
	if len(vector.Data) == 0 {
		return RingVector{}, fmt.Errorf("vector %q has no elements", vector.Label)
	}

	scalarNTT := toNTTMontgomeryPoly(r, scalar.Poly)
	result := NewRingVector(r, len(vector.Data), label)
	for i := range vector.Data {
		vectorPolyNTT := toNTTMontgomeryPoly(r, vector.Data[i])
		r.MulCoeffsMontgomery(scalarNTT, vectorPolyNTT, result.Data[i])
		fromNTTMontgomeryPoly(r, result.Data[i])
	}

	return result, nil
}

func MatrixDims(matrix RingMatrix) (rows int, cols int, err error) {
	rows = len(matrix.Data)
	if rows == 0 {
		return 0, 0, fmt.Errorf("matrix %q has no rows", matrix.Label)
	}

	cols = len(matrix.Data[0])
	if cols == 0 {
		return 0, 0, fmt.Errorf("matrix %q has no columns", matrix.Label)
	}

	for i := range matrix.Data {
		if len(matrix.Data[i]) != cols {
			return 0, 0, fmt.Errorf("matrix %q row %d has length %d, want %d", matrix.Label, i, len(matrix.Data[i]), cols)
		}
	}

	return rows, cols, nil
}

func toNTTMontgomeryPoly(r *ring.Ring, poly ring.Poly) ring.Poly {
	converted := r.NewPoly()
	converted.Copy(poly)
	r.NTT(converted, converted)
	r.MForm(converted, converted)
	return converted
}

func fromNTTMontgomeryPoly(r *ring.Ring, poly ring.Poly) {
	r.IMForm(poly, poly)
	r.INTT(poly, poly)
}
