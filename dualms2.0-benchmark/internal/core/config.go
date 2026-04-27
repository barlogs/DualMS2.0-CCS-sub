package core

import (
	"fmt"
	"math"
	"math/big"
)

const (
	defaultSecurityBits            = 128
	defaultLogN                    = 8
	defaultEta                     = 1
	defaultChallengeKappa          = 60
	defaultAlphaCoeff              = 4.917
	defaultRejectionM              = 0
	defaultTargetMn                = 5
	defaultMaxAttempts             = 100
	defaultSeedSize                = 32
	rounding01N8PaperQ             = 4261411837
	rounding01N8ImplementationQ    = 4261376513
	rounding01N32PaperQ            = 2113928077
	rounding01N32ImplementationQ   = 2113927681
	rounding01N1024PaperQ          = 4160748533
	rounding01N1024ImplementationQ = 4160727041
	defaultQ                       = rounding01N32ImplementationQ
	rounding01ParameterSet8        = "sec3-rounding01-n8"
	rounding01ParameterSet32       = "sec3-rounding01-n32"
	rounding01ParameterSet1024     = "sec3-rounding01-n1024"
	unsupportedParameterSetMsg     = "unsupported sec3 rounding01 parameter set for %d signers; supported sets are n=32 and n=1024"
	unsupportedNetworkParameterSet = "unsupported sec3 rounding01 network parameter set for %d signers; supported profiles are n=8 and n=32"
)

type Config struct {
	ParameterSet            string
	SecurityBits            int
	NumSigners              int
	Message                 string
	UseSmoothingSigma       bool
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
	AlphaCoeff              float64
	Alpha                   float64
	Sigma                   float64
	GaussianBound           float64
	RejectionM              float64
	TargetRejectionMn       float64
	MaxSigningAttempts      int
	SamplingSeed            []byte
	ZHNormBoundSquared      *big.Int
	RAggNormBoundSquared    *big.Int
	RBarAggNormBoundSquared *big.Int
	EnforceNormBounds       bool
}

func DefaultConfig() Config {
	cfg := Config{
		SecurityBits:       defaultSecurityBits,
		Message:            "rounding01-demo-message",
		LogN:               defaultLogN,
		Eta:                defaultEta,
		ChallengeKappa:     defaultChallengeKappa,
		RejectionM:         defaultRejectionM,
		TargetRejectionMn:  defaultTargetMn,
		MaxSigningAttempts: defaultMaxAttempts,
		SamplingSeed:       make([]byte, defaultSeedSize),
		EnforceNormBounds:  true,
	}
	if err := cfg.ApplyRounding01ParameterSet(32); err != nil {
		panic(err)
	}
	return cfg
}

func (cfg *Config) ApplyRounding01ParameterSet(numSigners int) error {
	switch numSigners {
	case 32, 1024:
		return cfg.applyRounding01ParameterProfile(numSigners, false)
	default:
		return fmt.Errorf(unsupportedParameterSetMsg, numSigners)
	}
}

func (cfg *Config) ApplyRounding01NetworkParameterProfile(numSigners int) error {
	switch numSigners {
	case 8, 32:
		return cfg.applyRounding01ParameterProfile(numSigners, true)
	default:
		return fmt.Errorf(unsupportedNetworkParameterSet, numSigners)
	}
}

func (cfg *Config) applyRounding01ParameterProfile(numSigners int, useSmoothingSigma bool) error {
	switch numSigners {
	case 8:
		cfg.ParameterSet = rounding01ParameterSet8
		cfg.NumSigners = 8
		cfg.PaperQLogCeil = 32
		cfg.PaperQ = rounding01N8PaperQ
		cfg.Q = rounding01N8ImplementationQ
		cfg.MDim = 6
		cfg.LDim = 6
		cfg.NuB = 11
		cfg.NuW = 26
	case 32:
		cfg.ParameterSet = rounding01ParameterSet32
		cfg.NumSigners = 32
		cfg.PaperQLogCeil = 31
		cfg.PaperQ = rounding01N32PaperQ
		cfg.Q = rounding01N32ImplementationQ
		cfg.MDim = 7
		cfg.LDim = 6
		cfg.NuB = 11
		cfg.NuW = 26
	case 1024:
		cfg.ParameterSet = rounding01ParameterSet1024
		cfg.NumSigners = 1024
		cfg.PaperQLogCeil = 32
		cfg.PaperQ = rounding01N1024PaperQ
		cfg.Q = rounding01N1024ImplementationQ
		cfg.MDim = 10
		cfg.LDim = 6
		cfg.NuB = 11
		cfg.NuW = 28
	default:
		return fmt.Errorf(unsupportedNetworkParameterSet, numSigners)
	}

	cfg.LogN = defaultLogN
	cfg.Eta = defaultEta
	cfg.ChallengeKappa = defaultChallengeKappa
	cfg.AlphaCoeff = defaultAlphaCoeff
	cfg.TargetRejectionMn = defaultTargetMn
	cfg.UseSmoothingSigma = useSmoothingSigma
	cfg.RefreshDerived()
	return nil
}

func (cfg *Config) RefreshDerived() {
	cfg.Alpha = cfg.AlphaCoeff * float64(cfg.NumSigners)
	cfg.Sigma = rounding01GaussianStdDev(*cfg)
	cfg.GaussianBound = 2 * cfg.Sigma
	cfg.QNuBFloor = floorDivPow2(cfg.Q, cfg.NuB)
	cfg.QNuBNearest = nearestDivPow2(cfg.Q, cfg.NuB)
	cfg.QNuWFloor = floorDivPow2(cfg.Q, cfg.NuW)
	cfg.QNuWNearest = nearestDivPow2(cfg.Q, cfg.NuW)
	cfg.ZHNormBoundSquared = rounding01ZHNormBoundSquared(*cfg)
	cfg.RAggNormBoundSquared = rounding01RAggNormBoundSquared(*cfg)
	cfg.RBarAggNormBoundSquared = rounding01RBarAggNormBoundSquared(*cfg)
}

func (cfg Config) EffectiveSingleSignerM() float64 {
	if cfg.RejectionM >= 1 {
		return cfg.RejectionM
	}
	if cfg.TargetRejectionMn <= 0 || cfg.NumSigners <= 0 {
		return 0
	}
	return math.Pow(cfg.TargetRejectionMn, 1/float64(cfg.NumSigners))
}

func rounding01GaussianStdDev(cfg Config) float64 {
	ringDegree := float64(uint64(1) << cfg.LogN)
	tBound := float64(cfg.ChallengeKappa*cfg.ChallengeKappa*cfg.Eta) *
		math.Sqrt(ringDegree*float64(cfg.LDim+cfg.MDim+1))
	alphaT := cfg.Alpha * tBound
	if !cfg.UseSmoothingSigma {
		return alphaT
	}
	mDim := float64(cfg.MDim)
	lDim := float64(cfg.LDim)
	smoothing := 2 * ringDegree *
		math.Pow(float64(cfg.Q), mDim/(mDim+lDim)+2/(ringDegree*(mDim+lDim))) /
		math.Sqrt(2*math.Pi)
	if smoothing > alphaT {
		return smoothing
	}
	return alphaT
}

func rounding01BzNorm(cfg Config) float64 {
	ringDegree := float64(uint64(1) << cfg.LogN)
	return math.Sqrt(float64(cfg.NumSigners)) * cfg.Sigma *
		math.Sqrt(2*float64(cfg.LDim+cfg.MDim)*ringDegree)
}

func rounding01ZHNormBoundSquared(cfg Config) *big.Int {
	ringDegree := float64(uint64(1) << cfg.LogN)
	zPrefixBound := cfg.Sigma * math.Sqrt(2*float64(cfg.NumSigners)*float64(cfg.LDim)*ringDegree)
	scaledHintBound := math.Sqrt(float64(cfg.MDim)*ringDegree) *
		(cfg.Sigma*math.Sqrt(2*float64(cfg.NumSigners)) +
			float64(cfg.Eta) +
			float64(cfg.ChallengeKappa)*math.Ldexp(1, cfg.NuB-1) +
			math.Ldexp(1, cfg.NuW))
	return squareCeilFloat64(zPrefixBound + scaledHintBound)
}

func rounding01RAggNormBoundSquared(cfg Config) *big.Int {
	ringDegree := float64(uint64(1) << cfg.LogN)
	normBound := math.Sqrt(float64(cfg.NumSigners)) * math.Sqrt(2) * cfg.Sigma * math.Sqrt(ringDegree)
	return squareCeilFloat64(normBound)
}

func rounding01RBarAggNormBoundSquared(cfg Config) *big.Int {
	ringDegree := float64(uint64(1) << cfg.LogN)
	normBound := float64(cfg.Eta) *
		math.Sqrt(float64(cfg.LDim)*ringDegree*float64(cfg.NumSigners))
	return squareCeilFloat64(normBound)
}

func squareCeilFloat64(value float64) *big.Int {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return big.NewInt(0)
	}
	ceilValue := uint64(math.Ceil(value))
	asBig := new(big.Int).SetUint64(ceilValue)
	return new(big.Int).Mul(asBig, asBig)
}

func floorDivPow2(value uint64, shift int) uint64 {
	if shift <= 0 {
		return value
	}
	if shift >= 64 {
		return 0
	}
	return value / (uint64(1) << shift)
}

func nearestDivPow2(value uint64, shift int) uint64 {
	if shift <= 0 {
		return value
	}
	if shift >= 64 {
		return 0
	}
	denom := uint64(1) << shift
	half := denom >> 1
	return (value + half) / denom
}

func effectiveRejectionM(cfg Config) (float64, error) {
	if cfg.RejectionM < 0 || math.IsNaN(cfg.RejectionM) || math.IsInf(cfg.RejectionM, 0) {
		return 0, fmt.Errorf("invalid rejection M: %v", cfg.RejectionM)
	}
	if cfg.RejectionM > 0 && cfg.RejectionM < 1 {
		return 0, fmt.Errorf("invalid rejection M: %v, want 0 or >= 1", cfg.RejectionM)
	}
	if cfg.RejectionM >= 1 {
		return cfg.RejectionM, nil
	}

	if cfg.TargetRejectionMn < 0 || math.IsNaN(cfg.TargetRejectionMn) || math.IsInf(cfg.TargetRejectionMn, 0) {
		return 0, fmt.Errorf("invalid target rejection M_n: %v", cfg.TargetRejectionMn)
	}
	if cfg.TargetRejectionMn == 0 {
		return 0, nil
	}
	if cfg.TargetRejectionMn < 1 {
		return 0, fmt.Errorf("invalid target rejection M_n: %v, want 0 or >= 1", cfg.TargetRejectionMn)
	}
	if cfg.NumSigners <= 0 {
		return 0, fmt.Errorf("num signers must be positive when target rejection M_n is enabled")
	}
	return math.Pow(cfg.TargetRejectionMn, 1/float64(cfg.NumSigners)), nil
}

func copyBigIntOrZero(value *big.Int) *big.Int {
	if value == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(value)
}
