package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/tuneinsight/lattigo/v5/ring"
	"github.com/tuneinsight/lattigo/v5/utils/buffer"
	"github.com/tuneinsight/lattigo/v5/utils/structs"
)

const networkMaxStringBytes = 1 << 20

const (
	minNetworkInt32 = -1 << 31
	maxNetworkInt32 = 1<<31 - 1
)

func encodePublicKey(pk PublicKey) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeInt32(&buf, pk.SignerID); err != nil {
		return nil, err
	}
	if err := writeRingVectorBinary(&buf, pk.T); err != nil {
		return nil, fmt.Errorf("write public key vector: %w", err)
	}
	return buf.Bytes(), nil
}

func decodePublicKey(data []byte) (PublicKey, error) {
	reader := buffer.NewBuffer(data)
	signerID, err := readInt32(reader)
	if err != nil {
		return PublicKey{}, err
	}
	t, err := readRingVectorBinary(reader)
	if err != nil {
		return PublicKey{}, fmt.Errorf("read public key vector: %w", err)
	}
	if err := requireNoTrailingBytes(reader); err != nil {
		return PublicKey{}, err
	}
	return PublicKey{
		SignerID: signerID,
		T:        t,
	}, nil
}

func encodeKeyAggContext(ctx NetworkKeyAggContext) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeString(&buf, ctx.Message); err != nil {
		return nil, fmt.Errorf("write keyagg message: %w", err)
	}
	if err := writeString(&buf, ctx.SessionID); err != nil {
		return nil, fmt.Errorf("write keyagg session id: %w", err)
	}
	if err := writePublicKeyList(&buf, ctx.PublicKeys); err != nil {
		return nil, fmt.Errorf("write keyagg public keys: %w", err)
	}
	return buf.Bytes(), nil
}

func decodeKeyAggContext(data []byte) (NetworkKeyAggContext, error) {
	reader := buffer.NewBuffer(data)
	message, err := readString(reader)
	if err != nil {
		return NetworkKeyAggContext{}, fmt.Errorf("read keyagg message: %w", err)
	}
	sessionID, err := readString(reader)
	if err != nil {
		return NetworkKeyAggContext{}, fmt.Errorf("read keyagg session id: %w", err)
	}
	keys, err := readPublicKeyList(reader)
	if err != nil {
		return NetworkKeyAggContext{}, fmt.Errorf("read keyagg public keys: %w", err)
	}
	if err := requireNoTrailingBytes(reader); err != nil {
		return NetworkKeyAggContext{}, err
	}
	return NetworkKeyAggContext{
		PublicKeys: keys,
		Message:    message,
		SessionID:  sessionID,
	}, nil
}

func encodeRound1Msg(msg Round1Msg) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeInt32(&buf, msg.SignerID); err != nil {
		return nil, err
	}
	if err := writeRingVectorBinary(&buf, msg.W); err != nil {
		return nil, fmt.Errorf("write round1 w: %w", err)
	}
	return buf.Bytes(), nil
}

func decodeRound1Msg(data []byte) (Round1Msg, error) {
	reader := buffer.NewBuffer(data)
	signerID, err := readInt32(reader)
	if err != nil {
		return Round1Msg{}, err
	}
	w, err := readRingVectorBinary(reader)
	if err != nil {
		return Round1Msg{}, fmt.Errorf("read round1 w: %w", err)
	}
	if err := requireNoTrailingBytes(reader); err != nil {
		return Round1Msg{}, err
	}
	return Round1Msg{
		SignerID: signerID,
		W:        w,
	}, nil
}

func encodeRound1MsgList(msgs []Round1Msg) ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(msgs))); err != nil {
		return nil, err
	}
	for _, msg := range msgs {
		encoded, err := encodeRound1Msg(msg)
		if err != nil {
			return nil, err
		}
		if err := writeBytes(&buf, encoded); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func decodeRound1MsgList(data []byte) ([]Round1Msg, error) {
	reader := buffer.NewBuffer(data)
	var size uint32
	if err := binary.Read(reader, binary.BigEndian, &size); err != nil {
		return nil, err
	}
	msgs := make([]Round1Msg, size)
	for i := range msgs {
		encoded, err := readBytes(reader)
		if err != nil {
			return nil, err
		}
		msg, err := decodeRound1Msg(encoded)
		if err != nil {
			return nil, err
		}
		msgs[i] = msg
	}
	if err := requireNoTrailingBytes(reader); err != nil {
		return nil, err
	}
	return msgs, nil
}

func encodePartialSig(sig PartialSig) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeInt32(&buf, sig.SignerID); err != nil {
		return nil, err
	}
	if err := writeBool(&buf, sig.Aborted); err != nil {
		return nil, err
	}
	if err := writeRingElementBinary(&buf, sig.C.Value); err != nil {
		return nil, fmt.Errorf("write partial challenge: %w", err)
	}
	if err := writeRingVectorBinary(&buf, sig.Z); err != nil {
		return nil, fmt.Errorf("write partial z: %w", err)
	}
	if err := writeRingVectorBinary(&buf, sig.R); err != nil {
		return nil, fmt.Errorf("write partial r: %w", err)
	}
	return buf.Bytes(), nil
}

func decodePartialSig(data []byte) (PartialSig, error) {
	reader := buffer.NewBuffer(data)
	signerID, err := readInt32(reader)
	if err != nil {
		return PartialSig{}, err
	}
	aborted, err := readBool(reader)
	if err != nil {
		return PartialSig{}, err
	}
	c, err := readRingElementBinary(reader)
	if err != nil {
		return PartialSig{}, fmt.Errorf("read partial challenge: %w", err)
	}
	z, err := readRingVectorBinary(reader)
	if err != nil {
		return PartialSig{}, fmt.Errorf("read partial z: %w", err)
	}
	r, err := readRingVectorBinary(reader)
	if err != nil {
		return PartialSig{}, fmt.Errorf("read partial r: %w", err)
	}
	if err := requireNoTrailingBytes(reader); err != nil {
		return PartialSig{}, err
	}
	return PartialSig{
		SignerID: signerID,
		C:        Challenge{Value: c},
		Z:        z,
		R:        r,
		Aborted:  aborted,
	}, nil
}

func writeRingElementBinary(w io.Writer, element RingElement) error {
	if err := writeString(w, element.Label); err != nil {
		return err
	}
	if _, err := element.Poly.WriteTo(w); err != nil {
		return err
	}
	return nil
}

func readRingElementBinary(r io.Reader) (RingElement, error) {
	label, err := readString(r)
	if err != nil {
		return RingElement{}, err
	}
	var poly ring.Poly
	if _, err := poly.ReadFrom(r); err != nil {
		return RingElement{}, err
	}
	return RingElement{
		Poly:  poly,
		Label: label,
	}, nil
}

func writeRingVectorBinary(w io.Writer, vector RingVector) error {
	if err := writeString(w, vector.Label); err != nil {
		return err
	}
	if _, err := vector.Data.WriteTo(w); err != nil {
		return err
	}
	return nil
}

func readRingVectorBinary(r io.Reader) (RingVector, error) {
	label, err := readString(r)
	if err != nil {
		return RingVector{}, err
	}
	var data structs.Vector[ring.Poly]
	if _, err := data.ReadFrom(r); err != nil {
		return RingVector{}, err
	}
	return RingVector{
		Data:  data,
		Label: label,
	}, nil
}

func writeString(w io.Writer, value string) error {
	if len(value) > networkMaxStringBytes {
		return fmt.Errorf("string is %d bytes, max %d", len(value), networkMaxStringBytes)
	}
	if err := binary.Write(w, binary.BigEndian, uint32(len(value))); err != nil {
		return err
	}
	if len(value) == 0 {
		return nil
	}
	_, err := writeFull(w, []byte(value))
	return err
}

func readString(r io.Reader) (string, error) {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return "", err
	}
	if size > networkMaxStringBytes {
		return "", fmt.Errorf("string is %d bytes, max %d", size, networkMaxStringBytes)
	}
	if size == 0 {
		return "", nil
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return "", err
	}
	return string(data), nil
}

func writeInt32(w io.Writer, value int) error {
	if value < minNetworkInt32 || value > maxNetworkInt32 {
		return fmt.Errorf("int value %d does not fit int32", value)
	}
	return binary.Write(w, binary.BigEndian, int32(value))
}

func readInt32(r io.Reader) (int, error) {
	var value int32
	if err := binary.Read(r, binary.BigEndian, &value); err != nil {
		return 0, err
	}
	return int(value), nil
}

func writeBool(w io.Writer, value bool) error {
	var encoded byte
	if value {
		encoded = 1
	}
	_, err := writeFull(w, []byte{encoded})
	return err
}

func readBool(r io.Reader) (bool, error) {
	var encoded [1]byte
	if _, err := io.ReadFull(r, encoded[:]); err != nil {
		return false, err
	}
	switch encoded[0] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("invalid bool encoding: %d", encoded[0])
	}
}

func writeIntSlice(w io.Writer, values []int) error {
	if err := binary.Write(w, binary.BigEndian, uint32(len(values))); err != nil {
		return err
	}
	for _, value := range values {
		if err := writeInt32(w, value); err != nil {
			return err
		}
	}
	return nil
}

func writePublicKeyList(w io.Writer, keys []PublicKey) error {
	if err := binary.Write(w, binary.BigEndian, uint32(len(keys))); err != nil {
		return err
	}
	for _, key := range keys {
		encoded, err := encodePublicKey(key)
		if err != nil {
			return err
		}
		if err := writeBytes(w, encoded); err != nil {
			return err
		}
	}
	return nil
}

func readPublicKeyList(r io.Reader) ([]PublicKey, error) {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return nil, err
	}
	keys := make([]PublicKey, size)
	for i := range keys {
		encoded, err := readBytes(r)
		if err != nil {
			return nil, err
		}
		key, err := decodePublicKey(encoded)
		if err != nil {
			return nil, err
		}
		keys[i] = key
	}
	return keys, nil
}

func writeBytes(w io.Writer, data []byte) error {
	if len(data) > networkMaxFrameBytes {
		return fmt.Errorf("byte slice is %d bytes, max %d", len(data), networkMaxFrameBytes)
	}
	if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	_, err := writeFull(w, data)
	return err
}

func readBytes(r io.Reader) ([]byte, error) {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return nil, err
	}
	if size > networkMaxFrameBytes {
		return nil, fmt.Errorf("byte slice is %d bytes, max %d", size, networkMaxFrameBytes)
	}
	data := make([]byte, size)
	if len(data) == 0 {
		return data, nil
	}
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

func readIntSlice(r io.Reader) ([]int, error) {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return nil, err
	}
	values := make([]int, size)
	for i := range values {
		value, err := readInt32(r)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	return values, nil
}

func requireNoTrailingBytes(reader *buffer.Buffer) error {
	if reader.Size() != 0 {
		return fmt.Errorf("payload has %d trailing bytes", reader.Size())
	}
	return nil
}
