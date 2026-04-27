package local

import (
	"fmt"
	"time"
)

func LocalRun(cfg Config) error {
	fmt.Println("Setup...")
	pp, err := Setup(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("Public params ready: %s\n", pp.Description)

	signers := make([]*Signer, cfg.NumSigners)
	for i := 0; i < cfg.NumSigners; i++ {
		signers[i] = NewSigner(i)
	}
	fmt.Printf("Created %d signers\n", len(signers))

	fmt.Println("KeyGen...")
	keys := make([]PublicKey, 0, len(signers))
	keyGenStart := time.Now()
	for _, signer := range signers {
		pk, sk, err := KeyGen(pp, signer.ID)
		if err != nil {
			return err
		}
		signer.PK = pk
		signer.SK = sk
		keys = append(keys, pk)
	}
	keyGenDuration := time.Since(keyGenStart)
	fmt.Println("KeyGen done")

	fmt.Println("KeyAgg...")
	keyAggStart := time.Now()
	keyAgg, err := KeyAgg(pp, keys)
	if err != nil {
		return err
	}
	keyAggDuration := time.Since(keyAggStart)
	fmt.Printf("KeyAgg done: %s\n", keyAgg.AggPK.TildeTExact)

	if cfg.MaxSigningAttempts <= 0 {
		return fmt.Errorf("max signing attempts must be positive")
	}

	var (
		sig              AggSig
		verifyOK         bool
		successAttempt   int
		rejectedAttempts int
		totalSign1       time.Duration
		totalSign2       time.Duration
		sAggDuration     time.Duration
		verifyDuration   time.Duration
	)

	for attempt := 1; attempt <= cfg.MaxSigningAttempts; attempt++ {
		fmt.Printf("Signing attempt %d/%d...\n", attempt, cfg.MaxSigningAttempts)
		sessionID := fmt.Sprintf("rounding-demo-001-attempt-%d", attempt)

		fmt.Println("Sign1...")
		sign1Start := time.Now()
		allMsgs := make([]Round1Msg, 0, len(signers))
		states := make([]Round1State, 0, len(signers))
		for _, signer := range signers {
			msg, st, err := Sign1(pp, signer.SK, keyAgg, cfg.Message, sessionID)
			if err != nil {
				return err
			}
			allMsgs = append(allMsgs, msg)
			states = append(states, st)
		}
		totalSign1 += time.Since(sign1Start)
		fmt.Printf("Sign1 done: collected %d commitments\n", len(allMsgs))

		fmt.Println("Sign2...")
		sign2Start := time.Now()
		partials := make([]PartialSig, 0, len(signers))
		for _, st := range states {
			partial, err := Sign2(pp, st, allMsgs)
			if err != nil {
				return err
			}
			partials = append(partials, partial)
		}
		totalSign2 += time.Since(sign2Start)
		fmt.Printf("Sign2 done: collected %d partial signatures\n", len(partials))

		abortedSignerIDs := collectAbortedSignerIDs(partials)
		if len(abortedSignerIDs) > 0 {
			rejectedAttempts++
			fmt.Printf("Signing attempt %d rejected by signers: %v\n", attempt, abortedSignerIDs)
			continue
		}

		fmt.Println("SAgg...")
		sAggStart := time.Now()
		sig, err = SAgg(pp, cfg.Message, keyAgg.AggPK, keyAgg.SignerIDs, allMsgs, partials)
		if err != nil {
			return err
		}
		sAggDuration = time.Since(sAggStart)
		fmt.Printf("SAgg done: %d signers aggregated\n", len(sig.SignerIDs))

		fmt.Println("Verify...")
		verifyStart := time.Now()
		verifyOK, err = Verify(pp, cfg.Message, sig, keyAgg.AggPK)
		if err != nil {
			return err
		}
		verifyDuration = time.Since(verifyStart)
		fmt.Printf("Verify result: %v\n", verifyOK)
		successAttempt = attempt
		break
	}

	if successAttempt == 0 {
		return fmt.Errorf("rounding signing failed after %d attempts; every attempt had at least one rejected signer (target M_n=%.3f, signer M=%.6f)", cfg.MaxSigningAttempts, pp.TargetRejectionMn, pp.RejectionM)
	}

	fmt.Println("Timing summary:")
	fmt.Printf("Signing attempts: %d success attempt / %d max attempts, %d rejected attempts\n", successAttempt, cfg.MaxSigningAttempts, rejectedAttempts)
	fmt.Printf("KeyGen duration: %.3f ms\n", durationMillis(keyGenDuration))
	fmt.Printf("KeyAgg duration: %.3f ms\n", durationMillis(keyAggDuration))
	fmt.Printf("Sign1 total duration across attempts: %.3f ms\n", durationMillis(totalSign1))
	fmt.Printf("Sign2 total duration across attempts: %.3f ms\n", durationMillis(totalSign2))
	fmt.Printf("SAgg duration: %.3f ms\n", durationMillis(sAggDuration))
	fmt.Printf("Verify duration: %.3f ms\n", durationMillis(verifyDuration))
	return nil
}

type Signer struct {
	ID int
	PK PublicKey
	SK SecretKey
}

func NewSigner(id int) *Signer {
	return &Signer{ID: id}
}

func collectAbortedSignerIDs(partials []PartialSig) []int {
	ids := make([]int, 0)
	for _, partial := range partials {
		if partial.Aborted {
			ids = append(ids, partial.SignerID)
		}
	}
	return ids
}

func durationMillis(duration time.Duration) float64 {
	return float64(duration.Nanoseconds()) / 1e6
}
