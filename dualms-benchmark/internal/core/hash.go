package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/tuneinsight/lattigo/v5/ring"
)

func HAgg(pp PublicParams, publicKeys []PublicKey, key PublicKey) Challenge {
	return HAggFromDigest(pp, HAggListDigest(pp, publicKeys), key)
}

func HAggListDigest(pp PublicParams, publicKeys []PublicKey) []byte {
	return hashPayload(pp, "DualMS/Hagg/list", func(input *hashInput) {
		input.writePublicKeys(publicKeys)
	})
}

func HAggFromDigest(pp PublicParams, listDigest []byte, key PublicKey) Challenge {
	payload := hashPayload(pp, "DualMS/Hagg/final", func(input *hashInput) {
		input.writeBytes(listDigest)
		input.writePublicKey(key)
	})
	return mustHashToChallenge(pp, payload, "DualMS/Hagg/challenge", fmt.Sprintf("a_%d", key.SignerID))
}

func HCom(pp PublicParams, aggPK AggPublicKey, message string) HComResult {
	payload := hashPayload(pp, "DualMS/HCom", func(input *hashInput) {
		input.writeRingVector(aggPK.TildeT)
		input.writeString(message)
	})

	b := mustHashToUniformMatrix(pp, payload, "DualMS/HCom/B", pp.KDim, pp.LPrimeDim, "B")
	bBar, err := BuildABar(pp.Ring, b, "[B|I]")
	if err != nil {
		panic(fmt.Sprintf("build HCom Bbar: %v", err))
	}

	return HComResult{B: b, BBar: bBar}
}

func HSig(pp PublicParams, aggPK AggPublicKey, message string, wAgg RingVector) Challenge {
	payload := hashPayload(pp, "DualMS/HSig", func(input *hashInput) {
		input.writeRingVector(aggPK.TildeT)
		input.writeString(message)
		input.writeRingVector(wAgg)
	})
	return mustHashToChallenge(pp, payload, "DualMS/HSig/challenge", "c")
}

func hashPayload(pp PublicParams, domain string, writeInput func(*hashInput)) []byte {
	input := &hashInput{}
	input.writeString(domain)
	input.writePublicParameterFingerprint(pp)
	writeInput(input)

	digest := sha256.Sum256(input.Bytes())
	payload := make([]byte, len(digest))
	copy(payload, digest[:])
	return payload
}

func mustHashToChallenge(pp PublicParams, payload []byte, domain string, label string) Challenge {
	if err := validateHashParams(pp); err != nil {
		panic(err)
	}

	prng, err := newFixedPRNG(payload, domain)
	if err != nil {
		panic(fmt.Sprintf("create challenge prng: %v", err))
	}

	sampler, err := ring.NewTernarySampler(prng, pp.Ring, ring.Ternary{H: pp.ChallengeKappa}, false)
	if err != nil {
		panic(fmt.Sprintf("create challenge sampler: %v", err))
	}

	return Challenge{Value: RingElement{Poly: sampler.ReadNew(), Label: label}}
}

func mustHashToUniformMatrix(pp PublicParams, payload []byte, domain string, rows int, cols int, label string) RingMatrix {
	if err := validateHashParams(pp); err != nil {
		panic(err)
	}

	matrix, err := SampleUniformMatrix(pp.Ring, payload, domain, rows, cols, label)
	if err != nil {
		panic(fmt.Sprintf("hash to uniform matrix: %v", err))
	}
	return matrix
}

func validateHashParams(pp PublicParams) error {
	if pp.Ring == nil {
		return fmt.Errorf("public params have nil ring")
	}
	if pp.ChallengeKappa <= 0 || pp.ChallengeKappa > pp.Ring.N() {
		return fmt.Errorf("invalid challenge kappa: %d, want 1..%d", pp.ChallengeKappa, pp.Ring.N())
	}
	return nil
}

type hashInput struct {
	bytes.Buffer
}

func (input *hashInput) writePublicParameterFingerprint(pp PublicParams) {
	input.writeString(pp.ParameterSet)
	input.writeUint64(uint64(pp.SecurityBits))
	input.writeUint64(uint64(pp.LogN))
	input.writeUint64(pp.Q)
	input.writeUint64(pp.PaperQ)
	input.writeUint64(uint64(pp.PaperQLogCeil))
	input.writeUint64(uint64(pp.KDim))
	input.writeUint64(uint64(pp.LDim))
	input.writeUint64(uint64(pp.LPrimeDim))
	input.writeUint64(uint64(pp.Eta))
	input.writeUint64(uint64(pp.EtaPrime))
	input.writeUint64(uint64(pp.ChallengeKappa))
}

func (input *hashInput) writePublicKeys(keys []PublicKey) {
	sortedKeys := append([]PublicKey(nil), keys...)
	sort.Slice(sortedKeys, func(i int, j int) bool {
		return sortedKeys[i].SignerID < sortedKeys[j].SignerID
	})

	input.writeUint64(uint64(len(sortedKeys)))
	for _, key := range sortedKeys {
		input.writePublicKey(key)
	}
}

func (input *hashInput) writePublicKey(key PublicKey) {
	input.writeInt(key.SignerID)
	input.writeRingVector(key.T)
}

func (input *hashInput) writeRingVector(vector RingVector) {
	input.writeUint64(uint64(len(vector.Data)))
	for _, poly := range vector.Data {
		input.writePoly(poly)
	}
}

func (input *hashInput) writePoly(poly ring.Poly) {
	input.writeUint64(uint64(len(poly.Coeffs)))
	for _, levelCoeffs := range poly.Coeffs {
		input.writeUint64(uint64(len(levelCoeffs)))
		for _, coeff := range levelCoeffs {
			input.writeUint64(coeff)
		}
	}
}

func (input *hashInput) writeBytes(value []byte) {
	input.writeUint64(uint64(len(value)))
	input.Write(value)
}

func (input *hashInput) writeString(value string) {
	input.writeUint64(uint64(len(value)))
	input.WriteString(value)
}

func (input *hashInput) writeInt(value int) {
	input.writeUint64(uint64(value))
}

func (input *hashInput) writeUint64(value uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], value)
	input.Write(buf[:])
}
