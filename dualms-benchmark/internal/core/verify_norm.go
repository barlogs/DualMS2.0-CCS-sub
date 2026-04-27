package core

import "math/big"

func verifyNormBounds(pp PublicParams, sig AggSig) (bool, error) {
	zAggNormSquared, err := RingVectorNorm2(pp.Ring, sig.ZAgg)
	if err != nil {
		return false, err
	}
	rAggNormSquared, err := RingVectorNorm2(pp.Ring, sig.RAgg)
	if err != nil {
		return false, err
	}
	eAggNormSquared, err := RingVectorNorm2(pp.Ring, sig.EAgg)
	if err != nil {
		return false, err
	}

	if !pp.EnforceNormBounds {
		return true, nil
	}
	if zAggNormSquared.Cmp(normBoundOrZero(pp.ZAggNormBoundSquared)) > 0 {
		return false, nil
	}
	if rAggNormSquared.Cmp(normBoundOrZero(pp.RAggNormBoundSquared)) > 0 {
		return false, nil
	}
	if eAggNormSquared.Cmp(normBoundOrZero(pp.EAggNormBoundSquared)) > 0 {
		return false, nil
	}

	return true, nil
}

func normBoundOrZero(bound *big.Int) *big.Int {
	if bound == nil {
		return big.NewInt(0)
	}
	return bound
}
