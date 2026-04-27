package core

import "fmt"

func Verify(pp PublicParams, message string, sig AggSig, aggPK AggPublicKey) (bool, error) {
	coreOK, err := VerifyCore(pp, message, sig, aggPK)
	if err != nil || !coreOK {
		return coreOK, err
	}
	return verifyNormBounds(pp, sig)
}

func VerifyCore(pp PublicParams, message string, sig AggSig, aggPK AggPublicKey) (bool, error) {
	wCheck, err := verifyCoreReconstructRoundedCommitment(pp, message, sig, aggPK)
	if err != nil {
		return false, err
	}

	expectedChallenge := HSigRoundedResidue(pp, wCheck, message, aggPK)
	return expectedChallenge.Equal(sig.C), nil
}

func verifyCoreReconstructRoundedCommitment(pp PublicParams, message string, sig AggSig, aggPK AggPublicKey) (RoundedResidueVector, error) {
	u, err := reconstructRoundedTarget(pp, message, sig.C, sig.ZAgg, sig.RAgg, sig.RBarAgg, aggPK)
	if err != nil {
		return RoundedResidueVector{}, err
	}
	return AddRoundedResiduesMod(u, sig.H, "w_check")
}

func reconstructRoundedTarget(pp PublicParams, message string, c Challenge, zAgg RingVector, rAgg RingElement, rBarAgg RingVector, aggPK AggPublicKey) (RoundedResidueVector, error) {
	if err := validateVerifyCoreInputs(pp, c, zAgg, rAgg, rBarAgg, aggPK); err != nil {
		return RoundedResidueVector{}, err
	}

	com := HComRounded(pp, aggPK, message)
	aZ, err := MatrixVectorMulRq(pp.Ring, pp.A, zAgg, "verify_A_zAgg")
	if err != nil {
		return RoundedResidueVector{}, fmt.Errorf("compute A*zAgg: %w", err)
	}
	tAggEmbedded, err := EmbedRoundedVectorToRq(pp.Ring, aggPK.TildeTRounded, "verify_embed_tAgg")
	if err != nil {
		return RoundedResidueVector{}, fmt.Errorf("embed rounded aggregate public key: %w", err)
	}
	cTAgg, err := PolyVectorScalarMul(pp.Ring, c.Value, tAggEmbedded, "verify_c_tAgg")
	if err != nil {
		return RoundedResidueVector{}, fmt.Errorf("compute c*(2^nu_b*tAggRounded): %w", err)
	}
	rB, err := PolyVectorScalarMul(pp.Ring, rAgg, com.BVec, "verify_rAgg_b")
	if err != nil {
		return RoundedResidueVector{}, fmt.Errorf("compute rAgg*b: %w", err)
	}
	bRBar, err := MatrixVectorMulRq(pp.Ring, com.BMat, rBarAgg, "verify_B_rBarAgg")
	if err != nil {
		return RoundedResidueVector{}, fmt.Errorf("compute B*rBarAgg: %w", err)
	}

	uExact, err := VectorSubRq(pp.Ring, aZ, cTAgg, "verify_Az_minus_cT")
	if err != nil {
		return RoundedResidueVector{}, fmt.Errorf("subtract c*(2^nu_b*tAggRounded): %w", err)
	}
	uExact, err = VectorAddRq(pp.Ring, uExact, rB, "verify_add_rB")
	if err != nil {
		return RoundedResidueVector{}, fmt.Errorf("add rAgg*b: %w", err)
	}
	uExact, err = VectorAddRq(pp.Ring, uExact, bRBar, "verify_u_exact")
	if err != nil {
		return RoundedResidueVector{}, fmt.Errorf("add B*rBarAgg: %w", err)
	}

	uRounded, err := RoundRingVectorToResidue(pp.Ring, uExact, pp.NuW, pp.QNuWFloor, "u_round")
	if err != nil {
		return RoundedResidueVector{}, fmt.Errorf("round reconstruction target: %w", err)
	}
	return uRounded, nil
}

func validateVerifyCoreInputs(pp PublicParams, c Challenge, zAgg RingVector, rAgg RingElement, rBarAgg RingVector, aggPK AggPublicKey) error {
	if pp.Ring == nil {
		return fmt.Errorf("public params have nil ring")
	}
	if pp.QNuWFloor == 0 {
		return fmt.Errorf("q_nu_w floor is zero for parameter set %q; VerifyCore under the paper floor definition is not implemented for this profile", pp.Description)
	}
	if c.Value.Poly.N() == 0 {
		return fmt.Errorf("aggregate signature challenge is empty")
	}
	if len(aggPK.TildeTExact.Data) != pp.MDim {
		return fmt.Errorf("aggregate exact public key dimension is %d, want %d", len(aggPK.TildeTExact.Data), pp.MDim)
	}
	if len(aggPK.TildeTRounded.Data) != pp.MDim {
		return fmt.Errorf("aggregate rounded public key dimension is %d, want %d", len(aggPK.TildeTRounded.Data), pp.MDim)
	}
	if len(zAgg.Data) != pp.LDim {
		return fmt.Errorf("aggregate z dimension is %d, want %d", len(zAgg.Data), pp.LDim)
	}
	if rAgg.Poly.N() == 0 {
		return fmt.Errorf("aggregate r is empty")
	}
	if len(rBarAgg.Data) != pp.LDim {
		return fmt.Errorf("aggregate rbar dimension is %d, want %d", len(rBarAgg.Data), pp.LDim)
	}
	return nil
}
