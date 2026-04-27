package core

import (
	"fmt"
	"math"
	"math/big"
)

const (
	defaultSecurityBits                   = 128
	defaultLogN                           = 8
	defaultEta                            = 1
	defaultEtaPrime                       = 1
	defaultChallengeKappa                 = 60
	defaultGamma                          = 1.1
	defaultTParam                         = 13.5
	defaultRejectionM                     = 0
	defaultTargetRejectionMn              = 5
	defaultMaxAttempts                    = 100
	defaultSeedSize                       = 32
	dualMS8PaperQ                         = 137438953349
	dualMS8ImplementationQ                = 0x1ffffff201
	dualMS32PaperQ                        = 137438953349
	dualMS32ImplementationQ               = 0x1ffffff201
	dualMS1024PaperQ                      = 281474976710597
	dualMS1024ImplementationQ             = 0xffffffffc001
	dualMSParameterSet8                   = "dualms-n8"
	dualMSParameterSet32                  = "dualms-n32"
	dualMSParameterSet1024                = "dualms-n1024"
	unsupportedParameterSetMsg            = "unsupported DualMS parameter set for %d signers; supported sets are n=32 and n=1024"
	unsupportedNetworkParameterProfileMsg = "unsupported DualMS network parameter profile %d; supported profiles are n=8, n=32, and n=1024"
)

type Config struct {
	ParameterSet         string
	SecurityBits         int
	NumSigners           int
	Message              string
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
	MaxSigningAttempts   int
	SamplingSeed         []byte
	ZAggNormBoundSquared *big.Int
	RAggNormBoundSquared *big.Int
	EAggNormBoundSquared *big.Int
	EnforceNormBounds    bool
}

func DefaultConfig() Config {
	cfg := Config{
		SecurityBits:       defaultSecurityBits,
		Message:            "dualms-demo-message",
		LogN:               defaultLogN,
		Eta:                defaultEta,
		EtaPrime:           defaultEtaPrime,
		ChallengeKappa:     defaultChallengeKappa,
		Gamma:              defaultGamma,
		TParam:             defaultTParam,
		RejectionM:         defaultRejectionM,
		TargetRejectionMn:  defaultTargetRejectionMn,
		MaxSigningAttempts: defaultMaxAttempts,
		SamplingSeed:       make([]byte, defaultSeedSize),
		EnforceNormBounds:  false,
	}
	if err := cfg.ApplyDualMSParameterSet(32); err != nil {
		panic(err)
	}
	cfg.RefreshDerived()
	return cfg
}

func (cfg *Config) ApplyDualMSParameterSet(numSigners int) error {
	switch numSigners {
	case 32, 1024:
		return cfg.applyDualMSParameterProfile(numSigners)
	default:
		return fmt.Errorf(unsupportedParameterSetMsg, numSigners)
	}
}

func (cfg *Config) ApplyDualMSNetworkParameterProfile(numSigners int) error {
	switch numSigners {
	case 8, 32, 1024:
		return cfg.applyDualMSParameterProfile(numSigners)
	default:
		return fmt.Errorf(unsupportedNetworkParameterProfileMsg, numSigners)
	}
}

func (cfg *Config) applyDualMSParameterProfile(numSigners int) error {
	switch numSigners {
	case 8:
		cfg.ParameterSet = dualMSParameterSet8
		cfg.NumSigners = 8
		cfg.PaperQLogCeil = 37
		cfg.PaperQ = dualMS8PaperQ
		cfg.Q = dualMS8ImplementationQ
		cfg.KDim = 5
		cfg.LDim = 7
	case 32:
		cfg.ParameterSet = dualMSParameterSet32
		cfg.NumSigners = 32
		cfg.PaperQLogCeil = 37
		cfg.PaperQ = dualMS32PaperQ
		cfg.Q = dualMS32ImplementationQ
		cfg.KDim = 6
		cfg.LDim = 7
	case 1024:
		cfg.ParameterSet = dualMSParameterSet1024
		cfg.NumSigners = 1024
		cfg.PaperQLogCeil = 48
		cfg.PaperQ = dualMS1024PaperQ
		cfg.Q = dualMS1024ImplementationQ
		cfg.KDim = 7
		cfg.LDim = 9
	default:
		return fmt.Errorf(unsupportedNetworkParameterProfileMsg, numSigners)
	}

	cfg.LogN = defaultLogN
	cfg.LPrimeDim = cfg.LDim + 1
	cfg.Eta = defaultEta
	cfg.EtaPrime = defaultEtaPrime
	cfg.ChallengeKappa = defaultChallengeKappa
	cfg.Gamma = defaultGamma
	cfg.TParam = defaultTParam
	cfg.Sigma = dualMSGaussianStdDev(cfg.NumSigners, cfg.KDim, cfg.LDim, cfg.Eta, cfg.ChallengeKappa)
	cfg.SigmaPrime = dualMSGaussianStdDev(cfg.NumSigners, cfg.KDim, cfg.LPrimeDim, cfg.EtaPrime, cfg.ChallengeKappa)
	cfg.TargetRejectionMn = dualMSPaperTargetRejectionMn(cfg.NumSigners, cfg.TParam)
	cfg.RefreshDerived()
	return nil
}

func (cfg *Config) RefreshDerived() {
	cfg.Alpha = 8.5 * float64(cfg.NumSigners)
	cfg.GaussianBound = 2 * cfg.Sigma
	cfg.GaussianPrimeBound = 2 * cfg.SigmaPrime
	cfg.ZAggNormBoundSquared = defaultZAggNormBoundSquared(*cfg)
	cfg.RAggNormBoundSquared = defaultRAggNormBoundSquared(*cfg)
	cfg.EAggNormBoundSquared = defaultEAggNormBoundSquared(*cfg)
}

func (cfg Config) SecretDim() int {
	return cfg.LDim + cfg.KDim
}

func (cfg Config) CommitmentRandomDim() int {
	return cfg.LPrimeDim + cfg.KDim
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

func dualMSGaussianStdDev(numSigners int, kDim int, responseDim int, eta int, kappa int) float64 {
	alpha := 8.5 * float64(numSigners)
	ringDegree := float64(uint64(1) << defaultLogN)
	tBound := float64(kappa*kappa*eta) * math.Sqrt(ringDegree*float64(responseDim+kDim))
	return alpha * tBound
}

func dualMSPaperSingleSignerM(numSigners int, tParam float64) float64 {
	alpha := 8.5 * float64(numSigners)
	return math.Exp(tParam/alpha + 1/(2*alpha*alpha))
}

func dualMSPaperTargetRejectionMn(numSigners int, tParam float64) float64 {
	return math.Pow(dualMSPaperSingleSignerM(numSigners, tParam), float64(numSigners))
}

func paperGaussianWidth(stdDev float64) float64 {
	return stdDev * math.Sqrt(2*math.Pi)
}

func PaperGaussianWidth(stdDev float64) float64 {
	return paperGaussianWidth(stdDev)
}

func copyBigIntOrZero(value *big.Int) *big.Int {
	if value == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(value)
}

func CopyBigIntOrZero(value *big.Int) *big.Int {
	return copyBigIntOrZero(value)
}

func defaultZAggNormBoundSquared(cfg Config) *big.Int {
	normBound := cfg.Gamma * cfg.Sigma *
		math.Sqrt(float64(cfg.NumSigners)*float64(uint64(1)<<cfg.LogN)*float64(cfg.LDim))
	return squareCeilFloat64(normBound)
}

func defaultRAggNormBoundSquared(cfg Config) *big.Int {
	normBound := cfg.Gamma * cfg.SigmaPrime *
		math.Sqrt(float64(cfg.NumSigners)*float64(uint64(1)<<cfg.LogN)*float64(cfg.LPrimeDim))
	return squareCeilFloat64(normBound)
}

func defaultEAggNormBoundSquared(cfg Config) *big.Int {
	normBound := cfg.Gamma * math.Sqrt(cfg.Sigma*cfg.Sigma+cfg.SigmaPrime*cfg.SigmaPrime) *
		math.Sqrt(float64(cfg.NumSigners)*float64(uint64(1)<<cfg.LogN)*float64(cfg.KDim))
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
