package local

import (
	"math/big"

	core "dualmspoc/internal/core"
)

type (
	Config       = core.Config
	PublicParams = core.PublicParams
	PublicKey    = core.PublicKey
	SecretKey    = core.SecretKey
	Signer       = core.Signer
	KeyAggResult = core.KeyAggResult
	Round1Msg    = core.Round1Msg
	Round1State  = core.Round1State
	PartialSig   = core.PartialSig
	AggSig       = core.AggSig
	RingVector   = core.RingVector
)

var (
	Setup                = core.Setup
	NewSigner            = core.NewSigner
	KeyGen               = core.KeyGen
	KeyAgg               = core.KeyAgg
	KeyAggWithListDigest = core.KeyAggWithListDigest
	HAggListDigest       = core.HAggListDigest
	Sign1                = core.Sign1
	Sign2                = core.Sign2
	SAgg                 = core.SAgg
	Verify               = core.Verify
	RingVectorNorm2      = core.RingVectorNorm2
)

func paperGaussianWidth(stdDev float64) float64 {
	return core.PaperGaussianWidth(stdDev)
}

func copyBigIntOrZero(value *big.Int) *big.Int {
	return core.CopyBigIntOrZero(value)
}
