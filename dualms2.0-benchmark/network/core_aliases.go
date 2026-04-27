package network

import core "sec3rounding/internal/core"

type (
	Config               = core.Config
	PublicParams         = core.PublicParams
	PublicKey            = core.PublicKey
	SecretKey            = core.SecretKey
	KeyAggResult         = core.KeyAggResult
	Round1Msg            = core.Round1Msg
	Round1State          = core.Round1State
	AggregatedRound1     = core.AggregatedRound1
	PartialSig           = core.PartialSig
	AggSig               = core.AggSig
	AggPublicKey         = core.AggPublicKey
	Challenge            = core.Challenge
	RingElement          = core.RingElement
	RingVector           = core.RingVector
	RoundedVector        = core.RoundedVector
	RoundedResidueVector = core.RoundedResidueVector
)

var (
	DefaultConfig           = core.DefaultConfig
	Setup                   = core.Setup
	KeyGen                  = core.KeyGen
	KeyAgg                  = core.KeyAgg
	Sign1                   = core.Sign1
	Sign2                   = core.Sign2
	SAgg                    = core.SAgg
	Verify                  = core.Verify
	AggregateRound1Messages = core.AggregateRound1Messages
	HSigRounded             = core.HSigRounded
)
