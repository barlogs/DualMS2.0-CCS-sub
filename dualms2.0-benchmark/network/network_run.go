package network

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

func RunNetworkCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf(networkUsage())
	}

	switch args[0] {
	case "coordinator":
		if len(args) != 2 {
			return fmt.Errorf(networkUsage())
		}
		return RunNetworkCoordinator(args[1])
	case "signer":
		if len(args) != 3 {
			return fmt.Errorf(networkUsage())
		}
		signerID, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid signer id: %w", err)
		}
		return RunNetworkSigner(signerID, args[2])
	default:
		return fmt.Errorf("unknown network mode %q\n%s", args[0], networkUsage())
	}
}

func networkUsage() string {
	return "usage: go run . net coordinator <net-config.json> | go run . net signer <signer-id> <net-config.json>"
}

type networkRemoteSigner struct {
	ID       int
	Conn     net.Conn
	Hello    NetworkHello
	PK       PublicKey
	BytesIn  int
	BytesOut int
}

func RunNetworkCoordinator(configPath string) error {
	netCfg, err := LoadNetworkConfig(configPath)
	if err != nil {
		return err
	}
	protocolCfg, err := netCfg.ProtocolConfig()
	if err != nil {
		return err
	}
	pp, err := Setup(protocolCfg)
	if err != nil {
		return fmt.Errorf("network coordinator setup: %w", err)
	}
	localPK, localSK, err := KeyGen(pp, netCfg.CoordinatorID)
	if err != nil {
		return fmt.Errorf("network coordinator keygen: %w", err)
	}

	listener, err := net.Listen("tcp", netCfg.CoordinatorListen)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", netCfg.CoordinatorListen, err)
	}
	defer listener.Close()

	expectedRemoteSigners := netCfg.NumSigners - 1
	fmt.Printf("Rounding01 network coordinator ready: listen=%s active_signers=%d profile=%s q=%d signer_M=%.6g\n",
		netCfg.CoordinatorListen,
		netCfg.NumSigners,
		protocolCfg.ParameterSet,
		pp.Q,
		pp.RejectionM,
	)
	fmt.Printf("Waiting for %d remote signer(s)...\n", expectedRemoteSigners)

	seen := make(map[int]NetworkHello, expectedRemoteSigners)
	remotes := make(map[int]*networkRemoteSigner, expectedRemoteSigners)
	publicKeys := map[int]PublicKey{
		netCfg.CoordinatorID: localPK,
	}
	totalBytesIn := 0
	totalBytesOut := 0
	wallStart := time.Now()
	for len(seen) < expectedRemoteSigners {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept signer connection: %w", err)
		}

		hello, pk, bytesIn, bytesOut, err := handleNetworkSignerBootstrap(conn, netCfg, pp)
		totalBytesIn += bytesIn
		totalBytesOut += bytesOut
		if err != nil {
			conn.Close()
			return err
		}
		if err := conn.SetDeadline(time.Time{}); err != nil {
			conn.Close()
			return fmt.Errorf("clear signer %d connection deadline: %w", hello.SignerID, err)
		}

		if _, ok := seen[hello.SignerID]; ok {
			conn.Close()
			return fmt.Errorf("duplicate signer connection: %d", hello.SignerID)
		}
		seen[hello.SignerID] = hello
		publicKeys[hello.SignerID] = pk
		remotes[hello.SignerID] = &networkRemoteSigner{
			ID:       hello.SignerID,
			Conn:     conn,
			Hello:    hello,
			PK:       pk,
			BytesIn:  bytesIn,
			BytesOut: bytesOut,
		}
		defer conn.Close()
		fmt.Printf("Accepted signer %d bootstrap: bytes_in=%d bytes_out=%d\n", hello.SignerID, bytesIn, bytesOut)
	}

	orderedKeys, err := orderedNetworkPublicKeys(netCfg, publicKeys)
	if err != nil {
		return err
	}
	keyAggStart := time.Now()
	keyAgg, err := KeyAgg(pp, orderedKeys)
	if err != nil {
		return fmt.Errorf("network coordinator keyagg: %w", err)
	}
	keyAggDuration := time.Since(keyAggStart)

	ctx := NetworkKeyAggContext{
		PublicKeys: orderedKeys,
		Message:    protocolCfg.Message,
		SessionID:  networkSessionID(1),
	}
	broadcastBytes, err := broadcastNetworkKeyAggContext(netCfg, remotes, ctx)
	totalBytesOut += broadcastBytes
	if err != nil {
		return err
	}
	bootstrapDuration := time.Since(wallStart)
	onlineStart := time.Now()

	var successAttempt int
	var rejectedAttempts int
	var coordinatorSign1Total time.Duration
	var coordinatorSign2Total time.Duration
	var coordinatorSign1Success time.Duration
	var coordinatorSign2Success time.Duration
	var round1CollectTotal time.Duration
	var round1BroadcastTotal time.Duration
	var round1CollectSuccess time.Duration
	var round1BroadcastSuccess time.Duration
	var partialCollectTotal time.Duration
	var decisionBroadcastTotal time.Duration
	var partialCollectSuccess time.Duration
	var decisionBroadcastSuccess time.Duration
	var sAggDuration time.Duration
	var verifyDuration time.Duration

	for attempt := 1; attempt <= netCfg.MaxSigningAttempts; attempt++ {
		sessionID := networkSessionID(attempt)

		localSign1Start := time.Now()
		localMsg, localState, err := Sign1(pp, localSK, keyAgg, protocolCfg.Message, sessionID)
		if err != nil {
			return fmt.Errorf("network coordinator sign1 attempt %d: %w", attempt, err)
		}
		localSign1Duration := time.Since(localSign1Start)
		coordinatorSign1Total += localSign1Duration

		allMsgs := map[int]Round1Msg{
			netCfg.CoordinatorID: localMsg,
		}
		round1CollectStart := time.Now()
		remoteRound1BytesIn, remoteRound1BytesOut, err := collectNetworkRound1Messages(netCfg, pp, remotes, allMsgs)
		round1CollectDuration := time.Since(round1CollectStart)
		round1CollectTotal += round1CollectDuration
		totalBytesIn += remoteRound1BytesIn
		totalBytesOut += remoteRound1BytesOut
		if err != nil {
			return err
		}
		orderedMsgs, err := orderedNetworkRound1Msgs(netCfg, allMsgs)
		if err != nil {
			return err
		}
		if _, err := AggregateRound1Messages(pp, localState, orderedMsgs); err != nil {
			return fmt.Errorf("network coordinator validate collected round1 messages on attempt %d: %w", attempt, err)
		}

		round1BroadcastStart := time.Now()
		broadcastRound1Bytes, err := broadcastNetworkRound1Msgs(netCfg, remotes, orderedMsgs)
		round1BroadcastDuration := time.Since(round1BroadcastStart)
		round1BroadcastTotal += round1BroadcastDuration
		totalBytesOut += broadcastRound1Bytes
		if err != nil {
			return err
		}

		localSign2Start := time.Now()
		localPartial, err := Sign2(pp, localState, orderedMsgs)
		if err != nil {
			return fmt.Errorf("network coordinator sign2 attempt %d: %w", attempt, err)
		}
		localSign2Duration := time.Since(localSign2Start)
		coordinatorSign2Total += localSign2Duration

		partials := map[int]PartialSig{
			netCfg.CoordinatorID: localPartial,
		}
		partialCollectStart := time.Now()
		partialBytesIn, partialBytesOut, err := collectNetworkPartialSignatures(netCfg, remotes, partials)
		partialCollectDuration := time.Since(partialCollectStart)
		partialCollectTotal += partialCollectDuration
		totalBytesIn += partialBytesIn
		totalBytesOut += partialBytesOut
		if err != nil {
			return err
		}
		orderedPartials, err := orderedNetworkPartials(netCfg, partials)
		if err != nil {
			return err
		}
		abortedSignerIDs := collectAbortedSignerIDs(orderedPartials)
		if len(abortedSignerIDs) > 0 {
			rejectedAttempts++
			if attempt == netCfg.MaxSigningAttempts {
				errorBytes, broadcastErr := broadcastNetworkError(netCfg, remotes, fmt.Sprintf("signing failed after %d attempts; aborted signers on last attempt: %v", attempt, abortedSignerIDs))
				totalBytesOut += errorBytes
				if broadcastErr != nil {
					return broadcastErr
				}
				return fmt.Errorf("network signing failed after %d attempts; last aborted signers: %v", attempt, abortedSignerIDs)
			}

			retry := NetworkRetry{
				Attempt:           attempt + 1,
				SessionID:         networkSessionID(attempt + 1),
				RejectedSignerIDs: abortedSignerIDs,
				Message:           fmt.Sprintf("retry attempt %d", attempt+1),
			}
			decisionBroadcastStart := time.Now()
			retryBytes, err := broadcastNetworkRetry(netCfg, remotes, retry)
			decisionBroadcastDuration := time.Since(decisionBroadcastStart)
			decisionBroadcastTotal += decisionBroadcastDuration
			totalBytesOut += retryBytes
			if err != nil {
				return err
			}
			fmt.Printf("Network attempt %d rejected by signers %v; retrying with session %s\n", attempt, abortedSignerIDs, retry.SessionID)
			continue
		}

		sAggStart := time.Now()
		sig, err := SAgg(pp, protocolCfg.Message, keyAgg.AggPK, keyAgg.SignerIDs, orderedMsgs, orderedPartials)
		if err != nil {
			errorBytes, broadcastErr := broadcastNetworkError(netCfg, remotes, fmt.Sprintf("network coordinator sagg attempt %d: %v", attempt, err))
			totalBytesOut += errorBytes
			if broadcastErr != nil {
				return broadcastErr
			}
			return fmt.Errorf("network coordinator sagg attempt %d: %w", attempt, err)
		}
		sAggAttemptDuration := time.Since(sAggStart)

		verifyStart := time.Now()
		verifyOK, err := Verify(pp, protocolCfg.Message, sig, keyAgg.AggPK)
		verifyAttemptDuration := time.Since(verifyStart)
		if err != nil {
			errorBytes, broadcastErr := broadcastNetworkError(netCfg, remotes, fmt.Sprintf("network coordinator verify attempt %d: %v", attempt, err))
			totalBytesOut += errorBytes
			if broadcastErr != nil {
				return broadcastErr
			}
			return fmt.Errorf("network verify attempt %d: %w", attempt, err)
		}
		if !verifyOK {
			errorBytes, broadcastErr := broadcastNetworkError(netCfg, remotes, fmt.Sprintf("verify rejected aggregate signature on attempt %d", attempt))
			totalBytesOut += errorBytes
			if broadcastErr != nil {
				return broadcastErr
			}
			return fmt.Errorf("network verify rejected aggregate signature on attempt %d", attempt)
		}

		coordinatorSign1Success = localSign1Duration
		coordinatorSign2Success = localSign2Duration
		round1CollectSuccess = round1CollectDuration
		round1BroadcastSuccess = round1BroadcastDuration
		partialCollectSuccess = partialCollectDuration
		sAggDuration = sAggAttemptDuration
		verifyDuration = verifyAttemptDuration
		successAttempt = attempt

		success := NetworkSuccess{
			Attempt:  attempt,
			VerifyOK: true,
			Message:  fmt.Sprintf("signature finalized on attempt %d", attempt),
		}
		decisionBroadcastStart := time.Now()
		successBytes, err := broadcastNetworkSuccess(netCfg, remotes, success)
		decisionBroadcastDuration := time.Since(decisionBroadcastStart)
		decisionBroadcastSuccess = decisionBroadcastDuration
		decisionBroadcastTotal += decisionBroadcastDuration
		totalBytesOut += successBytes
		if err != nil {
			return err
		}
		onlineDuration := time.Since(onlineStart)
		wallDuration := time.Since(wallStart)
		signerIDs := allNetworkSignerIDs(netCfg)
		sign1NetworkSuccess := round1CollectSuccess + round1BroadcastSuccess
		sign2NetworkSuccess := partialCollectSuccess + decisionBroadcastSuccess
		sign1E2ESuccess := coordinatorSign1Success + sign1NetworkSuccess
		sign2E2ESuccess := coordinatorSign2Success + sign2NetworkSuccess + sAggDuration

		fmt.Printf("Rounding01 network signing complete: signers=%v keyagg_duration=%.3f ms success_attempt=%d rejected_attempts=%d bootstrap=%.3f ms online=%.3f ms wall=%.3f ms coordinator_sign1_success=%.3f ms coordinator_sign1_total=%.3f ms coordinator_sign2_success=%.3f ms coordinator_sign2_total=%.3f ms round1_collect_success=%.3f ms round1_broadcast_success=%.3f ms round1_collect_total=%.3f ms round1_broadcast_total=%.3f ms partial_collect_success=%.3f ms decision_broadcast_success=%.3f ms partial_collect_total=%.3f ms decision_broadcast_total=%.3f ms sign1_network_success=%.3f ms sign2_network_success=%.3f ms sign1_e2e_success=%.3f ms sign2_e2e_success=%.3f ms sagg=%.3f ms verify=%.3f ms bytes_in=%d bytes_out=%d\n",
			signerIDs,
			durationMillis(keyAggDuration),
			successAttempt,
			rejectedAttempts,
			durationMillis(bootstrapDuration),
			durationMillis(onlineDuration),
			durationMillis(wallDuration),
			durationMillis(coordinatorSign1Success),
			durationMillis(coordinatorSign1Total),
			durationMillis(coordinatorSign2Success),
			durationMillis(coordinatorSign2Total),
			durationMillis(round1CollectSuccess),
			durationMillis(round1BroadcastSuccess),
			durationMillis(round1CollectTotal),
			durationMillis(round1BroadcastTotal),
			durationMillis(partialCollectSuccess),
			durationMillis(decisionBroadcastSuccess),
			durationMillis(partialCollectTotal),
			durationMillis(decisionBroadcastTotal),
			durationMillis(sign1NetworkSuccess),
			durationMillis(sign2NetworkSuccess),
			durationMillis(sign1E2ESuccess),
			durationMillis(sign2E2ESuccess),
			durationMillis(sAggDuration),
			durationMillis(verifyDuration),
			totalBytesIn,
			totalBytesOut,
		)
		coordinatorSummary := networkCoordinatorSummary{
			Role:                       "coordinator",
			CoordinatorID:              netCfg.CoordinatorID,
			SignerIDs:                  signerIDs,
			ActiveSigners:              netCfg.NumSigners,
			Message:                    protocolCfg.Message,
			ParameterSet:               protocolCfg.ParameterSet,
			Q:                          pp.Q,
			SignerM:                    pp.RejectionM,
			TargetRejectionMn:          pp.TargetRejectionMn,
			SuccessAttempt:             successAttempt,
			RejectedAttempts:           rejectedAttempts,
			BootstrapMS:                durationMillis(bootstrapDuration),
			OnlineMS:                   durationMillis(onlineDuration),
			WallMS:                     durationMillis(wallDuration),
			KeyAggMS:                   durationMillis(keyAggDuration),
			CoordinatorSign1Success:    durationMillis(coordinatorSign1Success),
			CoordinatorSign1Total:      durationMillis(coordinatorSign1Total),
			CoordinatorSign2Success:    durationMillis(coordinatorSign2Success),
			CoordinatorSign2Total:      durationMillis(coordinatorSign2Total),
			Round1CollectSuccessMS:     durationMillis(round1CollectSuccess),
			Round1BroadcastSuccessMS:   durationMillis(round1BroadcastSuccess),
			Round1CollectTotalMS:       durationMillis(round1CollectTotal),
			Round1BroadcastTotalMS:     durationMillis(round1BroadcastTotal),
			PartialCollectSuccessMS:    durationMillis(partialCollectSuccess),
			DecisionBroadcastSuccessMS: durationMillis(decisionBroadcastSuccess),
			PartialCollectTotalMS:      durationMillis(partialCollectTotal),
			DecisionBroadcastTotalMS:   durationMillis(decisionBroadcastTotal),
			Sign1NetworkSuccessMS:      durationMillis(sign1NetworkSuccess),
			Sign2NetworkSuccessMS:      durationMillis(sign2NetworkSuccess),
			Sign1E2ESuccessMS:          durationMillis(sign1E2ESuccess),
			Sign2E2ESuccessMS:          durationMillis(sign2E2ESuccess),
			SAggMS:                     durationMillis(sAggDuration),
			VerifyMS:                   durationMillis(verifyDuration),
			BytesIn:                    totalBytesIn,
			BytesOut:                   totalBytesOut,
		}
		printNetworkCoordinatorSummaryTable(coordinatorSummary)
		printNetworkSummary(coordinatorSummary)
		return nil
	}

	return fmt.Errorf("network signing failed after %d attempts", netCfg.MaxSigningAttempts)
}

func handleNetworkSignerBootstrap(conn net.Conn, netCfg NetworkConfig, pp PublicParams) (NetworkHello, PublicKey, int, int, error) {
	hello, bytesIn, bytesOut, err := handleNetworkHello(conn, netCfg)
	if err != nil {
		return NetworkHello{}, PublicKey{}, bytesIn, bytesOut, err
	}

	pk, keyBytesIn, keyBytesOut, err := handleNetworkPublicKey(conn, netCfg, pp, hello.SignerID)
	bytesIn += keyBytesIn
	bytesOut += keyBytesOut
	if err != nil {
		return NetworkHello{}, PublicKey{}, bytesIn, bytesOut, err
	}
	return hello, pk, bytesIn, bytesOut, nil
}

func handleNetworkHello(conn net.Conn, netCfg NetworkConfig) (NetworkHello, int, int, error) {
	deadline := time.Now().Add(time.Duration(netCfg.DialTimeoutMillis) * time.Millisecond)
	if err := conn.SetDeadline(deadline); err != nil {
		return NetworkHello{}, 0, 0, fmt.Errorf("set connection deadline: %w", err)
	}

	envelope, bytesIn, err := readNetworkEnvelope(conn)
	if err != nil {
		return NetworkHello{}, bytesIn, 0, err
	}
	if envelope.Type != networkMsgHello {
		return NetworkHello{}, bytesIn, 0, fmt.Errorf("expected %q, got %q", networkMsgHello, envelope.Type)
	}

	var hello NetworkHello
	if err := decodeNetworkPayload(envelope, &hello); err != nil {
		return NetworkHello{}, bytesIn, 0, err
	}
	if err := validateNetworkHello(netCfg, envelope.SignerID, hello); err != nil {
		bytesOut, writeErr := writeNetworkJSON(conn, networkMsgError, netCfg.CoordinatorID, NetworkError{Message: err.Error()})
		if writeErr != nil {
			return NetworkHello{}, bytesIn, bytesOut, fmt.Errorf("send error after invalid hello: %w", writeErr)
		}
		return NetworkHello{}, bytesIn, bytesOut, err
	}

	ack := NetworkHelloAck{
		CoordinatorID: netCfg.CoordinatorID,
		Accepted:      true,
		Message:       "hello accepted",
	}
	bytesOut, err := writeNetworkJSON(conn, networkMsgHelloAck, netCfg.CoordinatorID, ack)
	if err != nil {
		return NetworkHello{}, bytesIn, bytesOut, fmt.Errorf("send hello ack: %w", err)
	}
	return hello, bytesIn, bytesOut, nil
}

func handleNetworkPublicKey(conn net.Conn, netCfg NetworkConfig, pp PublicParams, signerID int) (PublicKey, int, int, error) {
	envelope, bytesIn, err := readNetworkEnvelope(conn)
	if err != nil {
		return PublicKey{}, bytesIn, 0, err
	}
	if envelope.Type != networkMsgPublicKey {
		return PublicKey{}, bytesIn, 0, fmt.Errorf("expected %q from signer %d, got %q", networkMsgPublicKey, signerID, envelope.Type)
	}
	if envelope.SignerID != signerID {
		return PublicKey{}, bytesIn, 0, fmt.Errorf("public key envelope signer_id=%d, want %d", envelope.SignerID, signerID)
	}
	pk, err := decodePublicKey(envelope.Payload)
	if err != nil {
		return PublicKey{}, bytesIn, 0, fmt.Errorf("decode public key from signer %d: %w", signerID, err)
	}
	if pk.SignerID != signerID {
		return PublicKey{}, bytesIn, 0, fmt.Errorf("public key signer_id=%d, want %d", pk.SignerID, signerID)
	}
	if len(pk.T.Data) != pp.MDim {
		return PublicKey{}, bytesIn, 0, fmt.Errorf("public key from signer %d has dimension %d, want %d", signerID, len(pk.T.Data), pp.MDim)
	}

	ack := NetworkPublicKeyAck{
		SignerID: signerID,
		Accepted: true,
		Message:  "public key accepted",
	}
	bytesOut, err := writeNetworkJSON(conn, networkMsgPublicKeyAck, netCfg.CoordinatorID, ack)
	if err != nil {
		return PublicKey{}, bytesIn, bytesOut, fmt.Errorf("send public key ack: %w", err)
	}
	return pk, bytesIn, bytesOut, nil
}

func broadcastNetworkKeyAggContext(netCfg NetworkConfig, remotes map[int]*networkRemoteSigner, ctx NetworkKeyAggContext) (int, error) {
	payload, err := encodeKeyAggContext(ctx)
	if err != nil {
		return 0, fmt.Errorf("encode key aggregation context: %w", err)
	}

	totalBytesOut := 0
	for _, signerID := range sortedRemoteSignerIDs(remotes) {
		remote := remotes[signerID]
		bytesOut, err := writeNetworkBinary(remote.Conn, networkMsgKeyAggCtx, netCfg.CoordinatorID, payload)
		totalBytesOut += bytesOut
		remote.BytesOut += bytesOut
		if err != nil {
			return totalBytesOut, fmt.Errorf("send key aggregation context to signer %d: %w", signerID, err)
		}
	}
	return totalBytesOut, nil
}

func collectNetworkRound1Messages(netCfg NetworkConfig, pp PublicParams, remotes map[int]*networkRemoteSigner, allMsgs map[int]Round1Msg) (int, int, error) {
	totalBytesIn := 0
	totalBytesOut := 0
	for _, signerID := range sortedRemoteSignerIDs(remotes) {
		remote := remotes[signerID]
		msg, bytesIn, err := readNetworkRound1Msg(remote.Conn, pp, signerID)
		totalBytesIn += bytesIn
		remote.BytesIn += bytesIn
		if err != nil {
			return totalBytesIn, totalBytesOut, err
		}
		if _, ok := allMsgs[msg.SignerID]; ok {
			return totalBytesIn, totalBytesOut, fmt.Errorf("duplicate round1 message for signer %d", msg.SignerID)
		}
		allMsgs[msg.SignerID] = msg

		ack := NetworkRound1Ack{
			SignerID: signerID,
			Accepted: true,
			Message:  "round1 accepted",
		}
		bytesOut, err := writeNetworkJSON(remote.Conn, networkMsgRound1Ack, netCfg.CoordinatorID, ack)
		totalBytesOut += bytesOut
		remote.BytesOut += bytesOut
		if err != nil {
			return totalBytesIn, totalBytesOut, fmt.Errorf("send round1 ack to signer %d: %w", signerID, err)
		}
	}
	return totalBytesIn, totalBytesOut, nil
}

func readNetworkRound1Msg(conn net.Conn, pp PublicParams, signerID int) (Round1Msg, int, error) {
	envelope, bytesIn, err := readNetworkEnvelope(conn)
	if err != nil {
		return Round1Msg{}, bytesIn, err
	}
	if envelope.Type != networkMsgRound1 {
		return Round1Msg{}, bytesIn, fmt.Errorf("expected %q from signer %d, got %q", networkMsgRound1, signerID, envelope.Type)
	}
	if envelope.SignerID != signerID {
		return Round1Msg{}, bytesIn, fmt.Errorf("round1 envelope signer_id=%d, want %d", envelope.SignerID, signerID)
	}
	msg, err := decodeRound1Msg(envelope.Payload)
	if err != nil {
		return Round1Msg{}, bytesIn, fmt.Errorf("decode round1 message from signer %d: %w", signerID, err)
	}
	if msg.SignerID != signerID {
		return Round1Msg{}, bytesIn, fmt.Errorf("round1 message signer_id=%d, want %d", msg.SignerID, signerID)
	}
	if len(msg.W.Data) != pp.MDim {
		return Round1Msg{}, bytesIn, fmt.Errorf("round1 message from signer %d has w dimension %d, want %d", signerID, len(msg.W.Data), pp.MDim)
	}
	return msg, bytesIn, nil
}

func broadcastNetworkRound1Msgs(netCfg NetworkConfig, remotes map[int]*networkRemoteSigner, msgs []Round1Msg) (int, error) {
	payload, err := encodeRound1MsgList(msgs)
	if err != nil {
		return 0, fmt.Errorf("encode all round1 messages: %w", err)
	}

	totalBytesOut := 0
	for _, signerID := range sortedRemoteSignerIDs(remotes) {
		remote := remotes[signerID]
		bytesOut, err := writeNetworkBinary(remote.Conn, networkMsgAllRound1, netCfg.CoordinatorID, payload)
		totalBytesOut += bytesOut
		remote.BytesOut += bytesOut
		if err != nil {
			return totalBytesOut, fmt.Errorf("send all round1 messages to signer %d: %w", signerID, err)
		}
	}
	return totalBytesOut, nil
}

func collectNetworkPartialSignatures(netCfg NetworkConfig, remotes map[int]*networkRemoteSigner, partials map[int]PartialSig) (int, int, error) {
	totalBytesIn := 0
	totalBytesOut := 0
	for _, signerID := range sortedRemoteSignerIDs(remotes) {
		remote := remotes[signerID]
		partial, bytesIn, err := readNetworkPartialSig(remote.Conn, signerID)
		totalBytesIn += bytesIn
		remote.BytesIn += bytesIn
		if err != nil {
			return totalBytesIn, totalBytesOut, err
		}
		if _, ok := partials[partial.SignerID]; ok {
			return totalBytesIn, totalBytesOut, fmt.Errorf("duplicate partial signature for signer %d", partial.SignerID)
		}
		partials[partial.SignerID] = partial

		ack := NetworkPartialSigAck{
			SignerID: signerID,
			Accepted: true,
			Aborted:  partial.Aborted,
			Message:  "partial accepted",
		}
		bytesOut, err := writeNetworkJSON(remote.Conn, networkMsgPartialAck, netCfg.CoordinatorID, ack)
		totalBytesOut += bytesOut
		remote.BytesOut += bytesOut
		if err != nil {
			return totalBytesIn, totalBytesOut, fmt.Errorf("send partial ack to signer %d: %w", signerID, err)
		}
	}
	return totalBytesIn, totalBytesOut, nil
}

func readNetworkPartialSig(conn net.Conn, signerID int) (PartialSig, int, error) {
	envelope, bytesIn, err := readNetworkEnvelope(conn)
	if err != nil {
		return PartialSig{}, bytesIn, err
	}
	if envelope.Type != networkMsgPartialSig {
		return PartialSig{}, bytesIn, fmt.Errorf("expected %q from signer %d, got %q", networkMsgPartialSig, signerID, envelope.Type)
	}
	if envelope.SignerID != signerID {
		return PartialSig{}, bytesIn, fmt.Errorf("partial signature envelope signer_id=%d, want %d", envelope.SignerID, signerID)
	}
	partial, err := decodePartialSig(envelope.Payload)
	if err != nil {
		return PartialSig{}, bytesIn, fmt.Errorf("decode partial signature from signer %d: %w", signerID, err)
	}
	if partial.SignerID != signerID {
		return PartialSig{}, bytesIn, fmt.Errorf("partial signature signer_id=%d, want %d", partial.SignerID, signerID)
	}
	return partial, bytesIn, nil
}

func broadcastNetworkRetry(netCfg NetworkConfig, remotes map[int]*networkRemoteSigner, retry NetworkRetry) (int, error) {
	totalBytesOut := 0
	for _, signerID := range sortedRemoteSignerIDs(remotes) {
		remote := remotes[signerID]
		bytesOut, err := writeNetworkJSON(remote.Conn, networkMsgRetry, netCfg.CoordinatorID, retry)
		totalBytesOut += bytesOut
		remote.BytesOut += bytesOut
		if err != nil {
			return totalBytesOut, fmt.Errorf("send retry to signer %d: %w", signerID, err)
		}
	}
	return totalBytesOut, nil
}

func broadcastNetworkSuccess(netCfg NetworkConfig, remotes map[int]*networkRemoteSigner, success NetworkSuccess) (int, error) {
	totalBytesOut := 0
	for _, signerID := range sortedRemoteSignerIDs(remotes) {
		remote := remotes[signerID]
		bytesOut, err := writeNetworkJSON(remote.Conn, networkMsgSuccess, netCfg.CoordinatorID, success)
		totalBytesOut += bytesOut
		remote.BytesOut += bytesOut
		if err != nil {
			return totalBytesOut, fmt.Errorf("send success to signer %d: %w", signerID, err)
		}
	}
	return totalBytesOut, nil
}

func broadcastNetworkError(netCfg NetworkConfig, remotes map[int]*networkRemoteSigner, message string) (int, error) {
	totalBytesOut := 0
	payload := NetworkError{Message: message}
	for _, signerID := range sortedRemoteSignerIDs(remotes) {
		remote := remotes[signerID]
		bytesOut, err := writeNetworkJSON(remote.Conn, networkMsgError, netCfg.CoordinatorID, payload)
		totalBytesOut += bytesOut
		remote.BytesOut += bytesOut
		if err != nil {
			return totalBytesOut, fmt.Errorf("send error to signer %d: %w", signerID, err)
		}
	}
	return totalBytesOut, nil
}

func validateNetworkHello(netCfg NetworkConfig, envelopeSignerID int, hello NetworkHello) error {
	if hello.SignerID != envelopeSignerID {
		return fmt.Errorf("hello signer_id %d does not match envelope signer_id %d", hello.SignerID, envelopeSignerID)
	}
	if hello.SignerID == netCfg.CoordinatorID {
		return fmt.Errorf("remote hello cannot use coordinator id %d", netCfg.CoordinatorID)
	}
	role, ok := netCfg.PartyRole(hello.SignerID)
	if !ok {
		return fmt.Errorf("unknown signer id %d", hello.SignerID)
	}
	if role != networkRoleSigner || hello.Role != networkRoleSigner {
		return fmt.Errorf("signer %d role mismatch: config=%q hello=%q", hello.SignerID, role, hello.Role)
	}
	if hello.NumSigners != netCfg.NumSigners {
		return fmt.Errorf("hello num_signers=%d, want %d", hello.NumSigners, netCfg.NumSigners)
	}
	if hello.ParameterProfile != netCfg.ParameterProfile {
		return fmt.Errorf("hello parameter_profile=%d, want %d", hello.ParameterProfile, netCfg.ParameterProfile)
	}
	if hello.TargetRejectionMn != netCfg.TargetRejectionMn {
		return fmt.Errorf("hello target_rejection_mn=%.6f, want %.6f", hello.TargetRejectionMn, netCfg.TargetRejectionMn)
	}
	if hello.Message != netCfg.Message {
		return fmt.Errorf("hello message does not match coordinator message")
	}
	return nil
}

func RunNetworkSigner(signerID int, configPath string) error {
	netCfg, err := LoadNetworkConfig(configPath)
	if err != nil {
		return err
	}
	role, ok := netCfg.PartyRole(signerID)
	if !ok {
		return fmt.Errorf("signer %d is not present in network config", signerID)
	}
	if signerID == netCfg.CoordinatorID {
		return fmt.Errorf("signer mode cannot use coordinator id %d", signerID)
	}
	if role != networkRoleSigner {
		return fmt.Errorf("signer %d has role %q, want %q", signerID, role, networkRoleSigner)
	}

	protocolCfg, err := netCfg.ProtocolConfig()
	if err != nil {
		return err
	}
	pp, err := Setup(protocolCfg)
	if err != nil {
		return fmt.Errorf("network signer setup: %w", err)
	}
	keyGenStart := time.Now()
	pk, sk, err := KeyGen(pp, signerID)
	if err != nil {
		return fmt.Errorf("network signer keygen: %w", err)
	}
	keyGenDuration := time.Since(keyGenStart)

	dialer := net.Dialer{Timeout: time.Duration(netCfg.DialTimeoutMillis) * time.Millisecond}
	start := time.Now()
	conn, err := dialer.Dial("tcp", netCfg.CoordinatorPublic)
	if err != nil {
		return fmt.Errorf("dial coordinator %s: %w", netCfg.CoordinatorPublic, err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(time.Duration(netCfg.DialTimeoutMillis) * time.Millisecond)); err != nil {
		return fmt.Errorf("set connection deadline: %w", err)
	}

	hello := NetworkHello{
		SignerID:          signerID,
		Role:              networkRoleSigner,
		NumSigners:        netCfg.NumSigners,
		ParameterProfile:  netCfg.ParameterProfile,
		TargetRejectionMn: netCfg.TargetRejectionMn,
		Message:           netCfg.Message,
	}
	totalBytesOut, err := writeNetworkJSON(conn, networkMsgHello, signerID, hello)
	if err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	envelope, totalBytesIn, err := readNetworkEnvelope(conn)
	if err != nil {
		return fmt.Errorf("read hello ack: %w", err)
	}
	if envelope.Type == networkMsgError {
		var networkErr NetworkError
		if decodeErr := decodeNetworkPayload(envelope, &networkErr); decodeErr != nil {
			return decodeErr
		}
		return fmt.Errorf("coordinator rejected hello: %s", networkErr.Message)
	}
	if envelope.Type != networkMsgHelloAck {
		return fmt.Errorf("expected %q, got %q", networkMsgHelloAck, envelope.Type)
	}

	var ack NetworkHelloAck
	if err := decodeNetworkPayload(envelope, &ack); err != nil {
		return err
	}
	if !ack.Accepted {
		return fmt.Errorf("coordinator did not accept hello: %s", ack.Message)
	}
	if ack.CoordinatorID != netCfg.CoordinatorID {
		return fmt.Errorf("ack coordinator_id=%d, want %d", ack.CoordinatorID, netCfg.CoordinatorID)
	}

	encodedPK, err := encodePublicKey(pk)
	if err != nil {
		return fmt.Errorf("encode public key: %w", err)
	}
	bytesOut, err := writeNetworkBinary(conn, networkMsgPublicKey, signerID, encodedPK)
	totalBytesOut += bytesOut
	if err != nil {
		return fmt.Errorf("send public key: %w", err)
	}

	envelope, bytesIn, err := readNetworkEnvelope(conn)
	totalBytesIn += bytesIn
	if err != nil {
		return fmt.Errorf("read public key ack: %w", err)
	}
	if envelope.Type == networkMsgError {
		var networkErr NetworkError
		if decodeErr := decodeNetworkPayload(envelope, &networkErr); decodeErr != nil {
			return decodeErr
		}
		return fmt.Errorf("coordinator rejected public key: %s", networkErr.Message)
	}
	if envelope.Type != networkMsgPublicKeyAck {
		return fmt.Errorf("expected %q, got %q", networkMsgPublicKeyAck, envelope.Type)
	}
	var pkAck NetworkPublicKeyAck
	if err := decodeNetworkPayload(envelope, &pkAck); err != nil {
		return err
	}
	if !pkAck.Accepted || pkAck.SignerID != signerID {
		return fmt.Errorf("invalid public key ack: accepted=%v signer_id=%d", pkAck.Accepted, pkAck.SignerID)
	}

	if err := conn.SetDeadline(time.Time{}); err != nil {
		return fmt.Errorf("clear coordinator connection deadline: %w", err)
	}

	ctxEnvelope, bytesIn, err := readNetworkEnvelope(conn)
	totalBytesIn += bytesIn
	if err != nil {
		return fmt.Errorf("read key aggregation context: %w", err)
	}
	if ctxEnvelope.Type != networkMsgKeyAggCtx {
		return fmt.Errorf("expected %q, got %q", networkMsgKeyAggCtx, ctxEnvelope.Type)
	}
	if ctxEnvelope.SignerID != netCfg.CoordinatorID {
		return fmt.Errorf("key aggregation context signer_id=%d, want coordinator %d", ctxEnvelope.SignerID, netCfg.CoordinatorID)
	}
	ctx, err := decodeKeyAggContext(ctxEnvelope.Payload)
	if err != nil {
		return fmt.Errorf("decode key aggregation context: %w", err)
	}
	if err := validateNetworkKeyAggContext(pp, netCfg, signerID, pk, ctx); err != nil {
		return err
	}

	keyAggStart := time.Now()
	keyAgg, err := KeyAgg(pp, ctx.PublicKeys)
	if err != nil {
		return fmt.Errorf("network signer keyagg: %w", err)
	}
	keyAggDuration := time.Since(keyAggStart)
	preSigningDuration := time.Since(start)
	onlineStart := time.Now()

	currentAttempt := 1
	currentSessionID := ctx.SessionID
	var rejectedAttempts int
	var totalSign1Duration time.Duration
	var totalSign2Duration time.Duration

	for {
		sign1Start := time.Now()
		msg, st, err := Sign1(pp, sk, keyAgg, ctx.Message, currentSessionID)
		if err != nil {
			return fmt.Errorf("network signer sign1 attempt %d: %w", currentAttempt, err)
		}
		sign1Duration := time.Since(sign1Start)
		totalSign1Duration += sign1Duration
		encodedRound1, err := encodeRound1Msg(msg)
		if err != nil {
			return fmt.Errorf("encode round1 message: %w", err)
		}
		bytesOut, err = writeNetworkBinary(conn, networkMsgRound1, signerID, encodedRound1)
		totalBytesOut += bytesOut
		if err != nil {
			return fmt.Errorf("send round1 message: %w", err)
		}

		envelope, bytesIn, err = readNetworkEnvelope(conn)
		totalBytesIn += bytesIn
		if err != nil {
			return fmt.Errorf("read round1 ack: %w", err)
		}
		if envelope.Type != networkMsgRound1Ack {
			return fmt.Errorf("expected %q, got %q", networkMsgRound1Ack, envelope.Type)
		}
		var round1Ack NetworkRound1Ack
		if err := decodeNetworkPayload(envelope, &round1Ack); err != nil {
			return err
		}
		if !round1Ack.Accepted || round1Ack.SignerID != signerID {
			return fmt.Errorf("invalid round1 ack: accepted=%v signer_id=%d", round1Ack.Accepted, round1Ack.SignerID)
		}

		envelope, bytesIn, err = readNetworkEnvelope(conn)
		totalBytesIn += bytesIn
		if err != nil {
			return fmt.Errorf("read all round1 messages: %w", err)
		}
		if envelope.Type != networkMsgAllRound1 {
			return fmt.Errorf("expected %q, got %q", networkMsgAllRound1, envelope.Type)
		}
		allMsgs, err := decodeRound1MsgList(envelope.Payload)
		if err != nil {
			return fmt.Errorf("decode all round1 messages: %w", err)
		}

		sign2Start := time.Now()
		partial, err := Sign2(pp, st, allMsgs)
		if err != nil {
			return fmt.Errorf("network signer sign2 attempt %d: %w", currentAttempt, err)
		}
		sign2Duration := time.Since(sign2Start)
		totalSign2Duration += sign2Duration
		encodedPartial, err := encodePartialSig(partial)
		if err != nil {
			return fmt.Errorf("encode partial signature: %w", err)
		}
		bytesOut, err = writeNetworkBinary(conn, networkMsgPartialSig, signerID, encodedPartial)
		totalBytesOut += bytesOut
		if err != nil {
			return fmt.Errorf("send partial signature: %w", err)
		}

		envelope, bytesIn, err = readNetworkEnvelope(conn)
		totalBytesIn += bytesIn
		if err != nil {
			return fmt.Errorf("read partial ack: %w", err)
		}
		if envelope.Type != networkMsgPartialAck {
			return fmt.Errorf("expected %q, got %q", networkMsgPartialAck, envelope.Type)
		}
		var partialAck NetworkPartialSigAck
		if err := decodeNetworkPayload(envelope, &partialAck); err != nil {
			return err
		}
		if !partialAck.Accepted || partialAck.SignerID != signerID {
			return fmt.Errorf("invalid partial ack: accepted=%v signer_id=%d", partialAck.Accepted, partialAck.SignerID)
		}

		envelope, bytesIn, err = readNetworkEnvelope(conn)
		totalBytesIn += bytesIn
		if err != nil {
			return fmt.Errorf("read coordinator decision: %w", err)
		}
		switch envelope.Type {
		case networkMsgRetry:
			var retry NetworkRetry
			if err := decodeNetworkPayload(envelope, &retry); err != nil {
				return err
			}
			if retry.Attempt <= currentAttempt {
				return fmt.Errorf("invalid retry attempt %d after current attempt %d", retry.Attempt, currentAttempt)
			}
			if retry.SessionID == "" {
				return fmt.Errorf("retry session id must not be empty")
			}
			rejectedAttempts++
			currentAttempt = retry.Attempt
			currentSessionID = retry.SessionID
			fmt.Printf("Network signer %d received retry: next_attempt=%d rejected_signers=%v session=%s\n", signerID, retry.Attempt, retry.RejectedSignerIDs, retry.SessionID)
			continue
		case networkMsgSuccess:
			var success NetworkSuccess
			if err := decodeNetworkPayload(envelope, &success); err != nil {
				return err
			}
			if !success.VerifyOK {
				return fmt.Errorf("coordinator reported unsuccessful final status on attempt %d", success.Attempt)
			}
			if success.Attempt != currentAttempt {
				return fmt.Errorf("coordinator success attempt %d does not match local current attempt %d", success.Attempt, currentAttempt)
			}
			onlineDuration := time.Since(onlineStart)
			wallDuration := time.Since(start)
			fmt.Printf("Rounding01 network signer %d complete: coordinator=%s profile=%s q=%d signer_M=%.6g success_attempt=%d rejected_attempts=%d pre_signing=%.3f ms online=%.3f ms wall=%.3f ms keygen=%.3f ms keyagg=%.3f ms local_sign1_total=%.3f ms local_sign2_total=%.3f ms bytes_out=%d bytes_in=%d\n",
				signerID,
				netCfg.CoordinatorPublic,
				protocolCfg.ParameterSet,
				pp.Q,
				pp.RejectionM,
				success.Attempt,
				rejectedAttempts,
				durationMillis(preSigningDuration),
				durationMillis(onlineDuration),
				durationMillis(wallDuration),
				durationMillis(keyGenDuration),
				durationMillis(keyAggDuration),
				durationMillis(totalSign1Duration),
				durationMillis(totalSign2Duration),
				totalBytesOut,
				totalBytesIn,
			)
			signerSummary := networkSignerSummary{
				Role:              "signer",
				SignerID:          signerID,
				CoordinatorPublic: netCfg.CoordinatorPublic,
				Message:           ctx.Message,
				ParameterSet:      protocolCfg.ParameterSet,
				Q:                 pp.Q,
				SignerM:           pp.RejectionM,
				TargetRejectionMn: pp.TargetRejectionMn,
				SuccessAttempt:    success.Attempt,
				RejectedAttempts:  rejectedAttempts,
				PreSigningMS:      durationMillis(preSigningDuration),
				OnlineMS:          durationMillis(onlineDuration),
				WallMS:            durationMillis(wallDuration),
				KeyGenMS:          durationMillis(keyGenDuration),
				KeyAggMS:          durationMillis(keyAggDuration),
				LocalSign1TotalMS: durationMillis(totalSign1Duration),
				LocalSign2TotalMS: durationMillis(totalSign2Duration),
				BytesIn:           totalBytesIn,
				BytesOut:          totalBytesOut,
			}
			printNetworkSignerSummaryTable(signerSummary)
			printNetworkSummary(signerSummary)
			return nil
		case networkMsgError:
			var networkErr NetworkError
			if err := decodeNetworkPayload(envelope, &networkErr); err != nil {
				return err
			}
			return fmt.Errorf("coordinator reported fatal error: %s", networkErr.Message)
		default:
			return fmt.Errorf("expected %q, %q, or %q; got %q", networkMsgRetry, networkMsgSuccess, networkMsgError, envelope.Type)
		}
	}
}

func networkSessionID(attempt int) string {
	return fmt.Sprintf("network-session-001-attempt-%03d", attempt)
}

func validateNetworkKeyAggContext(pp PublicParams, netCfg NetworkConfig, signerID int, localPK PublicKey, ctx NetworkKeyAggContext) error {
	if ctx.Message != netCfg.Message {
		return fmt.Errorf("key aggregation context message does not match network config")
	}
	if ctx.SessionID == "" {
		return fmt.Errorf("key aggregation context session id must not be empty")
	}
	if len(ctx.PublicKeys) != netCfg.NumSigners {
		return fmt.Errorf("key aggregation context has %d public keys, want %d", len(ctx.PublicKeys), netCfg.NumSigners)
	}
	seen := make(map[int]struct{}, len(ctx.PublicKeys))
	hasLocalKey := false
	for _, key := range ctx.PublicKeys {
		if key.SignerID < 0 {
			return fmt.Errorf("key aggregation context contains negative signer id %d", key.SignerID)
		}
		if _, ok := netCfg.PartyRole(key.SignerID); !ok {
			return fmt.Errorf("key aggregation context contains unknown signer id %d", key.SignerID)
		}
		if _, ok := seen[key.SignerID]; ok {
			return fmt.Errorf("key aggregation context contains duplicate signer id %d", key.SignerID)
		}
		if len(key.T.Data) != pp.MDim {
			return fmt.Errorf("public key for signer %d has dimension %d, want %d", key.SignerID, len(key.T.Data), pp.MDim)
		}
		if key.SignerID == signerID {
			hasLocalKey = true
			if !ringVectorsEqual(pp.Ring, key.T, localPK.T) {
				return fmt.Errorf("key aggregation context public key for local signer %d does not match local keygen output", signerID)
			}
		}
		seen[key.SignerID] = struct{}{}
	}
	if !hasLocalKey {
		return fmt.Errorf("key aggregation context is missing local signer %d public key", signerID)
	}
	return nil
}

func orderedNetworkPublicKeys(netCfg NetworkConfig, keys map[int]PublicKey) ([]PublicKey, error) {
	if len(keys) != netCfg.NumSigners {
		return nil, fmt.Errorf("have %d public keys, want %d", len(keys), netCfg.NumSigners)
	}
	ids := make([]int, 0, len(keys))
	for signerID := range keys {
		ids = append(ids, signerID)
	}
	sortInts(ids)

	ordered := make([]PublicKey, 0, len(ids))
	for _, signerID := range ids {
		if _, ok := netCfg.PartyRole(signerID); !ok {
			return nil, fmt.Errorf("public key from unknown signer %d", signerID)
		}
		ordered = append(ordered, keys[signerID])
	}
	return ordered, nil
}

func orderedNetworkRound1Msgs(netCfg NetworkConfig, msgs map[int]Round1Msg) ([]Round1Msg, error) {
	if len(msgs) != netCfg.NumSigners {
		return nil, fmt.Errorf("have %d round1 messages, want %d", len(msgs), netCfg.NumSigners)
	}
	ids := make([]int, 0, len(msgs))
	for signerID := range msgs {
		ids = append(ids, signerID)
	}
	sortInts(ids)

	ordered := make([]Round1Msg, 0, len(ids))
	for _, signerID := range ids {
		if _, ok := netCfg.PartyRole(signerID); !ok {
			return nil, fmt.Errorf("round1 message from unknown signer %d", signerID)
		}
		ordered = append(ordered, msgs[signerID])
	}
	return ordered, nil
}

func orderedNetworkPartials(netCfg NetworkConfig, partials map[int]PartialSig) ([]PartialSig, error) {
	if len(partials) != netCfg.NumSigners {
		return nil, fmt.Errorf("have %d partial signatures, want %d", len(partials), netCfg.NumSigners)
	}
	ids := make([]int, 0, len(partials))
	for signerID := range partials {
		ids = append(ids, signerID)
	}
	sortInts(ids)

	ordered := make([]PartialSig, 0, len(ids))
	for _, signerID := range ids {
		if _, ok := netCfg.PartyRole(signerID); !ok {
			return nil, fmt.Errorf("partial signature from unknown signer %d", signerID)
		}
		ordered = append(ordered, partials[signerID])
	}
	return ordered, nil
}

func sortedRemoteSignerIDs(remotes map[int]*networkRemoteSigner) []int {
	ids := make([]int, 0, len(remotes))
	for signerID := range remotes {
		ids = append(ids, signerID)
	}
	sortInts(ids)
	return ids
}

func allNetworkSignerIDs(netCfg NetworkConfig) []int {
	ids := make([]int, 0, len(netCfg.Parties))
	for _, party := range netCfg.Parties {
		ids = append(ids, party.ID)
	}
	sortInts(ids)
	return ids
}

func sortInts(values []int) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
