package local

import core "sec3rounding/internal/core"

type (
	Config       = core.Config
	PublicParams = core.PublicParams
	PublicKey    = core.PublicKey
	SecretKey    = core.SecretKey
	KeyAggResult = core.KeyAggResult
	Round1Msg    = core.Round1Msg
	Round1State  = core.Round1State
	PartialSig   = core.PartialSig
	AggSig       = core.AggSig
)

var (
	Setup                = core.Setup
	KeyGen               = core.KeyGen
	KeyAgg               = core.KeyAgg
	KeyAggWithListDigest = core.KeyAggWithListDigest
	HAggListDigest       = core.HAggListDigest
	Sign1                = core.Sign1
	Sign2                = core.Sign2
	SAgg                 = core.SAgg
	Verify               = core.Verify
)
