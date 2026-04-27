package core

import (
	"fmt"

	"github.com/tuneinsight/lattigo/v5/ring"
)

func Setup(cfg Config) (PublicParams, error) {
	r, err := ring.NewRing(1<<cfg.LogN, []uint64{cfg.Q})
	if err != nil {
		return PublicParams{}, fmt.Errorf("setup ring: %w", err)
	}
	if cfg.ChallengeKappa <= 0 || cfg.ChallengeKappa > r.N() {
		return PublicParams{}, fmt.Errorf("invalid challenge kappa: %d, want 1..%d", cfg.ChallengeKappa, r.N())
	}
	rejectionM, err := effectiveRejectionM(cfg)
	if err != nil {
		return PublicParams{}, err
	}

	samplingSeed := copySamplingSeed(cfg.SamplingSeed)

	a, err := SampleUniformMatrix(r, samplingSeed, "setup:A", cfg.MDim, cfg.LDim, "A")
	if err != nil {
		return PublicParams{}, fmt.Errorf("sample A: %w", err)
	}

	aBar, err := BuildABar(r, a, "[A|I]")
	if err != nil {
		return PublicParams{}, fmt.Errorf("build Abar: %w", err)
	}

	return PublicParams{
		SecurityBits:            cfg.SecurityBits,
		Description:             fmt.Sprintf("section3-rounding01-step0-pp-%dbit", cfg.SecurityBits),
		Ring:                    r,
		LogN:                    cfg.LogN,
		Q:                       cfg.Q,
		PaperQ:                  cfg.PaperQ,
		PaperQLogCeil:           cfg.PaperQLogCeil,
		LDim:                    cfg.LDim,
		MDim:                    cfg.MDim,
		NuB:                     cfg.NuB,
		NuW:                     cfg.NuW,
		QNuBFloor:               cfg.QNuBFloor,
		QNuBNearest:             cfg.QNuBNearest,
		QNuWFloor:               cfg.QNuWFloor,
		QNuWNearest:             cfg.QNuWNearest,
		Eta:                     cfg.Eta,
		ChallengeKappa:          cfg.ChallengeKappa,
		Sigma:                   cfg.Sigma,
		GaussianBound:           cfg.GaussianBound,
		RejectionM:              rejectionM,
		TargetRejectionMn:       cfg.TargetRejectionMn,
		SamplingSeed:            samplingSeed,
		ZHNormBoundSquared:      copyBigIntOrZero(cfg.ZHNormBoundSquared),
		RAggNormBoundSquared:    copyBigIntOrZero(cfg.RAggNormBoundSquared),
		RBarAggNormBoundSquared: copyBigIntOrZero(cfg.RBarAggNormBoundSquared),
		EnforceNormBounds:       cfg.EnforceNormBounds,
		A:                       a,
		ABar:                    aBar,
	}, nil
}
