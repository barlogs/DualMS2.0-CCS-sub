package core

import (
	"fmt"
	"math"

	"github.com/tuneinsight/lattigo/v5/ring"
)

func Setup(cfg Config) (PublicParams, error) {
	if err := validateConfig(cfg); err != nil {
		return PublicParams{}, err
	}

	r, err := ring.NewRing(1<<cfg.LogN, []uint64{cfg.Q})
	if err != nil {
		return PublicParams{}, fmt.Errorf("setup ring: %w", err)
	}

	a, err := SampleUniformMatrix(r, cfg.SamplingSeed, "dualms:setup:A", cfg.KDim, cfg.LDim, "A")
	if err != nil {
		return PublicParams{}, fmt.Errorf("sample A: %w", err)
	}
	aBar, err := BuildABar(r, a, "[A|I]")
	if err != nil {
		return PublicParams{}, fmt.Errorf("build Abar: %w", err)
	}

	return PublicParams{
		ParameterSet:         cfg.ParameterSet,
		SecurityBits:         cfg.SecurityBits,
		Description:          fmt.Sprintf("dualms-figure3-poc-%s-%dbit", cfg.ParameterSet, cfg.SecurityBits),
		Ring:                 r,
		LogN:                 cfg.LogN,
		Q:                    cfg.Q,
		PaperQ:               cfg.PaperQ,
		PaperQLogCeil:        cfg.PaperQLogCeil,
		KDim:                 cfg.KDim,
		LDim:                 cfg.LDim,
		LPrimeDim:            cfg.LPrimeDim,
		Eta:                  cfg.Eta,
		EtaPrime:             cfg.EtaPrime,
		ChallengeKappa:       cfg.ChallengeKappa,
		Gamma:                cfg.Gamma,
		TParam:               cfg.TParam,
		Alpha:                cfg.Alpha,
		Sigma:                cfg.Sigma,
		SigmaPrime:           cfg.SigmaPrime,
		GaussianBound:        cfg.GaussianBound,
		GaussianPrimeBound:   cfg.GaussianPrimeBound,
		RejectionM:           cfg.EffectiveSingleSignerM(),
		TargetRejectionMn:    cfg.TargetRejectionMn,
		SamplingSeed:         copySamplingSeed(cfg.SamplingSeed),
		ZAggNormBoundSquared: copyBigIntOrZero(cfg.ZAggNormBoundSquared),
		RAggNormBoundSquared: copyBigIntOrZero(cfg.RAggNormBoundSquared),
		EAggNormBoundSquared: copyBigIntOrZero(cfg.EAggNormBoundSquared),
		EnforceNormBounds:    cfg.EnforceNormBounds,
		A:                    a,
		ABar:                 aBar,
	}, nil
}

func validateConfig(cfg Config) error {
	if cfg.NumSigners <= 0 {
		return fmt.Errorf("num signers must be positive")
	}
	if cfg.LogN <= 0 {
		return fmt.Errorf("logN must be positive")
	}
	if cfg.Q == 0 {
		return fmt.Errorf("q must be non-zero")
	}
	if cfg.KDim <= 0 || cfg.LDim <= 0 || cfg.LPrimeDim <= 0 {
		return fmt.Errorf("invalid dimensions: k=%d l=%d l'=%d", cfg.KDim, cfg.LDim, cfg.LPrimeDim)
	}
	if cfg.ChallengeKappa <= 0 || cfg.ChallengeKappa > 1<<cfg.LogN {
		return fmt.Errorf("invalid challenge kappa: %d", cfg.ChallengeKappa)
	}
	if cfg.Eta != 1 || cfg.EtaPrime != 1 {
		return fmt.Errorf("skeleton currently supports eta=eta'=1, got eta=%d eta'=%d", cfg.Eta, cfg.EtaPrime)
	}
	if cfg.Sigma <= 0 || cfg.SigmaPrime <= 0 {
		return fmt.Errorf("invalid sigma parameters: s=%f s'=%f", cfg.Sigma, cfg.SigmaPrime)
	}
	if cfg.GaussianBound <= 0 || cfg.GaussianPrimeBound <= 0 {
		return fmt.Errorf("invalid gaussian bounds")
	}
	if cfg.RejectionM < 0 || math.IsNaN(cfg.RejectionM) || math.IsInf(cfg.RejectionM, 0) {
		return fmt.Errorf("invalid rejection M: %v", cfg.RejectionM)
	}
	if cfg.RejectionM > 0 && cfg.RejectionM < 1 {
		return fmt.Errorf("invalid rejection M: %v, want 0 or >=1", cfg.RejectionM)
	}
	if cfg.TargetRejectionMn < 0 || math.IsNaN(cfg.TargetRejectionMn) || math.IsInf(cfg.TargetRejectionMn, 0) {
		return fmt.Errorf("invalid target rejection M_n: %v", cfg.TargetRejectionMn)
	}
	if cfg.TargetRejectionMn > 0 && cfg.TargetRejectionMn < 1 {
		return fmt.Errorf("invalid target rejection M_n: %v, want 0 or >=1", cfg.TargetRejectionMn)
	}
	return nil
}
