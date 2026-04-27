package network

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const networkMaxFrameBytes = 64 << 20

type networkEnvelope struct {
	Type     string
	SignerID int
	Payload  []byte
}

func writeNetworkJSON(w io.Writer, msgType string, signerID int, payload any) (int, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal network payload: %w", err)
	}
	return writeNetworkBinary(w, msgType, signerID, payloadBytes)
}

func writeNetworkBinary(w io.Writer, msgType string, signerID int, payload []byte) (int, error) {
	if msgType == "" {
		return 0, fmt.Errorf("network message type must not be empty")
	}
	if len(payload) > networkMaxFrameBytes {
		return 0, fmt.Errorf("network payload is %d bytes, max %d", len(payload), networkMaxFrameBytes)
	}

	var body bytes.Buffer
	if err := writeString(&body, msgType); err != nil {
		return 0, fmt.Errorf("write network message type: %w", err)
	}
	if err := writeInt32(&body, signerID); err != nil {
		return 0, fmt.Errorf("write network signer id: %w", err)
	}
	if err := binary.Write(&body, binary.BigEndian, uint32(len(payload))); err != nil {
		return 0, fmt.Errorf("write network payload length: %w", err)
	}
	if _, err := writeFull(&body, payload); err != nil {
		return 0, fmt.Errorf("write network payload: %w", err)
	}
	if body.Len() > networkMaxFrameBytes {
		return 0, fmt.Errorf("network frame is %d bytes, max %d", body.Len(), networkMaxFrameBytes)
	}

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(body.Len()))
	headerBytes, err := writeFull(w, header)
	if err != nil {
		return headerBytes, err
	}
	bodyBytes, err := writeFull(w, body.Bytes())
	return headerBytes + bodyBytes, err
}

func readNetworkEnvelope(r io.Reader) (networkEnvelope, int, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return networkEnvelope{}, 0, fmt.Errorf("read network frame header: %w", err)
	}
	frameLen := binary.BigEndian.Uint32(header)
	if frameLen == 0 {
		return networkEnvelope{}, 4, fmt.Errorf("network frame is empty")
	}
	if frameLen > networkMaxFrameBytes {
		return networkEnvelope{}, 4, fmt.Errorf("network frame is %d bytes, max %d", frameLen, networkMaxFrameBytes)
	}

	bodyBytes := make([]byte, frameLen)
	if _, err := io.ReadFull(r, bodyBytes); err != nil {
		return networkEnvelope{}, 4, fmt.Errorf("read network frame body: %w", err)
	}

	body := bytes.NewReader(bodyBytes)
	msgType, err := readString(body)
	if err != nil {
		return networkEnvelope{}, 4 + int(frameLen), fmt.Errorf("read network message type: %w", err)
	}
	if msgType == "" {
		return networkEnvelope{}, 4 + int(frameLen), fmt.Errorf("network envelope has empty type")
	}
	signerID, err := readInt32(body)
	if err != nil {
		return networkEnvelope{}, 4 + int(frameLen), fmt.Errorf("read network signer id: %w", err)
	}
	var payloadLen uint32
	if err := binary.Read(body, binary.BigEndian, &payloadLen); err != nil {
		return networkEnvelope{}, 4 + int(frameLen), fmt.Errorf("read network payload length: %w", err)
	}
	if int(payloadLen) > body.Len() {
		return networkEnvelope{}, 4 + int(frameLen), fmt.Errorf("network payload length %d exceeds remaining frame bytes %d", payloadLen, body.Len())
	}
	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(body, payload); err != nil {
		return networkEnvelope{}, 4 + int(frameLen), fmt.Errorf("read network payload: %w", err)
	}
	if body.Len() != 0 {
		return networkEnvelope{}, 4 + int(frameLen), fmt.Errorf("network frame has %d trailing bytes", body.Len())
	}

	return networkEnvelope{
		Type:     msgType,
		SignerID: signerID,
		Payload:  payload,
	}, 4 + int(frameLen), nil
}

func decodeNetworkPayload(envelope networkEnvelope, dst any) error {
	if len(envelope.Payload) == 0 {
		return fmt.Errorf("network envelope %q has empty payload", envelope.Type)
	}
	if err := json.Unmarshal(envelope.Payload, dst); err != nil {
		return fmt.Errorf("decode network payload %q: %w", envelope.Type, err)
	}
	return nil
}

func writeFull(w io.Writer, data []byte) (int, error) {
	total := 0
	for len(data) > 0 {
		n, err := w.Write(data)
		total += n
		if err != nil {
			return total, err
		}
		if n == 0 {
			return total, io.ErrShortWrite
		}
		data = data[n:]
	}
	return total, nil
}
