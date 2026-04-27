package core

import "fmt"

func Verify(pp PublicParams, message string, sig AggSig, aggPK AggPublicKey) bool {
	wCheck, err := verifyWCheck(pp, message, sig, aggPK)
	if err != nil {
		return false
	}

	expectedChallenge := HSig(pp, aggPK, message, wCheck)
	if !expectedChallenge.Equal(sig.C) {
		return false
	}

	normOK, err := verifyNormBounds(pp, sig)
	if err != nil {
		return false
	}
	return normOK
}

func verifyWCheck(pp PublicParams, message string, sig AggSig, aggPK AggPublicKey) (RingVector, error) {
	if err := validateVerifyInputs(pp, sig, aggPK); err != nil {
		return RingVector{}, err
	}

	com := HCom(pp, aggPK, message)
	aZ, err := MatrixVectorMulRq(pp.Ring, pp.A, sig.ZAgg, "verify_A_z_tilde")
	if err != nil {
		return RingVector{}, fmt.Errorf("compute A*z_tilde: %w", err)
	}
	bR, err := MatrixVectorMulRq(pp.Ring, com.B, sig.RAgg, "verify_B_r_tilde")
	if err != nil {
		return RingVector{}, fmt.Errorf("compute B*r_tilde: %w", err)
	}
	cT, err := PolyVectorScalarMul(pp.Ring, sig.C.Value, aggPK.TildeT, "verify_c_t_tilde")
	if err != nil {
		return RingVector{}, fmt.Errorf("compute c*t_tilde: %w", err)
	}

	wCheck, err := VectorAddRq(pp.Ring, aZ, bR, "verify_Az_plus_Br")
	if err != nil {
		return RingVector{}, fmt.Errorf("add A*z_tilde and B*r_tilde: %w", err)
	}
	wCheck, err = VectorAddRq(pp.Ring, wCheck, sig.EAgg, "verify_add_e_tilde")
	if err != nil {
		return RingVector{}, fmt.Errorf("add e_tilde: %w", err)
	}
	wCheck, err = VectorSubRq(pp.Ring, wCheck, cT, "verify_w_tilde")
	if err != nil {
		return RingVector{}, fmt.Errorf("subtract c*t_tilde: %w", err)
	}

	return wCheck, nil
}

func validateVerifyInputs(pp PublicParams, sig AggSig, aggPK AggPublicKey) error {
	if pp.Ring == nil {
		return fmt.Errorf("public params have nil ring")
	}
	if sig.C.Value.Poly.N() == 0 {
		return fmt.Errorf("aggregate signature challenge is empty")
	}
	if len(aggPK.TildeT.Data) != pp.KDim {
		return fmt.Errorf("aggregate public key dimension is %d, want %d", len(aggPK.TildeT.Data), pp.KDim)
	}
	if len(sig.ZAgg.Data) != pp.LDim {
		return fmt.Errorf("aggregate z dimension is %d, want %d", len(sig.ZAgg.Data), pp.LDim)
	}
	if len(sig.RAgg.Data) != pp.LPrimeDim {
		return fmt.Errorf("aggregate r dimension is %d, want %d", len(sig.RAgg.Data), pp.LPrimeDim)
	}
	if len(sig.EAgg.Data) != pp.KDim {
		return fmt.Errorf("aggregate e dimension is %d, want %d", len(sig.EAgg.Data), pp.KDim)
	}
	return nil
}
