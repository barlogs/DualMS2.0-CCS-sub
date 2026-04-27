package network

const (
	networkMsgHello        = "hello"
	networkMsgHelloAck     = "hello_ack"
	networkMsgPublicKey    = "public_key"
	networkMsgPublicKeyAck = "public_key_ack"
	networkMsgKeyAggCtx    = "keyagg_context"
	networkMsgRound1       = "round1_msg"
	networkMsgRound1Ack    = "round1_ack"
	networkMsgAllRound1    = "all_round1_msgs"
	networkMsgPartialSig   = "partial_sig"
	networkMsgPartialAck   = "partial_sig_ack"
	networkMsgRetry        = "retry"
	networkMsgSuccess      = "success"
	networkMsgError        = "error"
)

type NetworkHello struct {
	SignerID         int    `json:"signer_id"`
	Role             string `json:"role"`
	NumSigners       int    `json:"num_signers"`
	ParameterProfile int    `json:"parameter_profile"`
	Message          string `json:"message"`
}

type NetworkHelloAck struct {
	CoordinatorID int    `json:"coordinator_id"`
	Accepted      bool   `json:"accepted"`
	Message       string `json:"message"`
}

type NetworkError struct {
	Message string `json:"message"`
}

type NetworkPublicKeyAck struct {
	SignerID int    `json:"signer_id"`
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}

type NetworkKeyAggContext struct {
	PublicKeys []PublicKey
	Message    string
	SessionID  string
}

type NetworkRound1Ack struct {
	SignerID int    `json:"signer_id"`
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}

type NetworkPartialSigAck struct {
	SignerID int    `json:"signer_id"`
	Accepted bool   `json:"accepted"`
	Aborted  bool   `json:"aborted"`
	Message  string `json:"message"`
}

type NetworkRetry struct {
	Attempt           int    `json:"attempt"`
	SessionID         string `json:"session_id"`
	RejectedSignerIDs []int  `json:"rejected_signer_ids"`
	Message           string `json:"message"`
}

type NetworkSuccess struct {
	Attempt  int    `json:"attempt"`
	VerifyOK bool   `json:"verify_ok"`
	Message  string `json:"message"`
}
