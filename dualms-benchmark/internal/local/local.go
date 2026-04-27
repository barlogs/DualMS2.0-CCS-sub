package local

import (
	"fmt"
	"math"
)

func LocalRun(cfg Config) error {
	if cfg.MaxSigningAttempts <= 0 {
		return fmt.Errorf("max signing attempts must be positive")
	}

	fmt.Println("DualMS Setup...")
	pp, err := Setup(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("Public params ready: %s\n", pp.Description)
	fmt.Printf("Paper params: set=%s n=%d N=%d ceil(log q)=%d paper q=%d k=%d l=%d l'=%d eta=%d kappa=%d log(s)=%.2f log(s')=%.2f\n",
		cfg.ParameterSet,
		cfg.NumSigners,
		1<<cfg.LogN,
		cfg.PaperQLogCeil,
		cfg.PaperQ,
		cfg.KDim,
		cfg.LDim,
		cfg.LPrimeDim,
		cfg.Eta,
		cfg.ChallengeKappa,
		math.Log2(paperGaussianWidth(cfg.Sigma)),
		math.Log2(paperGaussianWidth(cfg.SigmaPrime)),
	)
	fmt.Printf("Implementation q: %d (0x%x), signer M=%.8f target M_n=%.3f\n",
		cfg.Q,
		cfg.Q,
		pp.RejectionM,
		pp.TargetRejectionMn,
	)

	signers := make([]*Signer, cfg.NumSigners)
	keys := make([]PublicKey, 0, cfg.NumSigners)
	for i := 0; i < cfg.NumSigners; i++ {
		signers[i] = NewSigner(i)
		pk, sk, err := KeyGen(pp, signers[i].ID)
		if err != nil {
			return err
		}
		signers[i].PK = pk
		signers[i].SK = sk
		keys = append(keys, pk)
	}
	fmt.Printf("KeyGen done: %d signers\n", len(signers))

	keyAgg, err := KeyAgg(pp, keys)
	if err != nil {
		return err
	}
	fmt.Printf("KeyAgg done: %s with %d signers\n", keyAgg.AggPK.TildeT, len(keyAgg.SignerIDs))

	for attempt := 1; attempt <= cfg.MaxSigningAttempts; attempt++ {
		fmt.Printf("Signing attempt %d/%d...\n", attempt, cfg.MaxSigningAttempts)

		sessionID := fmt.Sprintf("dualms-demo-attempt-%03d", attempt)
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
		fmt.Printf("Sign1 done: collected %d commitments and %d local states\n", len(allMsgs), len(states))

		partials := make([]PartialSig, 0, len(states))
		abortedSignerIDs := make([]int, 0)
		for _, st := range states {
			partial, err := Sign2(pp, st, allMsgs)
			if err != nil {
				return err
			}
			if partial.Aborted {
				abortedSignerIDs = append(abortedSignerIDs, partial.SignerID)
			}
			partials = append(partials, partial)
		}
		fmt.Printf("Sign2 done: collected %d partial signatures\n", len(partials))
		if len(abortedSignerIDs) > 0 {
			fmt.Printf("Signing attempt %d rejected by signers: %v\n", attempt, abortedSignerIDs)
			continue
		}

		sig, err := SAgg(pp, partials)
		if err != nil {
			return err
		}
		fmt.Printf("SAgg done: aggregated %d partial signatures into (c, z_tilde[%d], r_tilde[%d], e_tilde[%d])\n",
			len(sig.SignerIDs),
			len(sig.ZAgg.Data),
			len(sig.RAgg.Data),
			len(sig.EAgg.Data),
		)

		fmt.Println("Verify...")
		verifyOK := Verify(pp, cfg.Message, sig, keyAgg.AggPK)
		fmt.Printf("Verify result: %t\n", verifyOK)
		if !verifyOK {
			return fmt.Errorf("verify rejected aggregate signature")
		}
		fmt.Printf("Signing succeeded on attempt %d/%d\n", attempt, cfg.MaxSigningAttempts)
		return nil
	}

	return fmt.Errorf("signing failed after %d attempts due to rejection sampling", cfg.MaxSigningAttempts)
}
