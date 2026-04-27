package network

import core "dualmspoc/internal/core"

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
	AggPublicKey = core.AggPublicKey
	Challenge    = core.Challenge
	RingElement  = core.RingElement
	RingVector   = core.RingVector
)

var (
	DefaultConfig = core.DefaultConfig
	Setup         = core.Setup
	NewSigner     = core.NewSigner
	KeyGen        = core.KeyGen
	KeyAgg        = core.KeyAgg
	Sign1         = core.Sign1
	Sign2         = core.Sign2
	SAgg          = core.SAgg
	Verify        = core.Verify
)
