package core

import "math/big"

func verifyNormBounds(pp PublicParams, sig AggSig) (bool, error) {
	zHintNormSquared, err := ZHintNorm2(pp.Ring, sig.ZAgg, sig.H)
	if err != nil {
		return false, err
	}
	rAggNormSquared, err := RingElementNorm2(pp.Ring, sig.RAgg)
	if err != nil {
		return false, err
	}
	rBarAggNormSquared, err := RingVectorNorm2(pp.Ring, sig.RBarAgg)
	if err != nil {
		return false, err
	}

	if !pp.EnforceNormBounds {
		return true, nil
	}
	if zHintNormSquared.Cmp(normBoundOrZero(pp.ZHNormBoundSquared)) > 0 {
		return false, nil
	}
	if rAggNormSquared.Cmp(normBoundOrZero(pp.RAggNormBoundSquared)) > 0 {
		return false, nil
	}
	if rBarAggNormSquared.Cmp(normBoundOrZero(pp.RBarAggNormBoundSquared)) > 0 {
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
