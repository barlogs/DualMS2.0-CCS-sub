package network

import (
	"encoding/json"
	"fmt"
	"strings"
)

type networkCoordinatorSummary struct {
	Role                       string  `json:"role"`
	CoordinatorID              int     `json:"coordinator_id"`
	SignerIDs                  []int   `json:"signer_ids"`
	ActiveSigners              int     `json:"active_signers"`
	Message                    string  `json:"message"`
	ParameterSet               string  `json:"parameter_set"`
	Q                          uint64  `json:"q"`
	SignerM                    float64 `json:"signer_m"`
	TargetRejectionMn          float64 `json:"target_rejection_mn"`
	SuccessAttempt             int     `json:"success_attempt"`
	RejectedAttempts           int     `json:"rejected_attempts"`
	BootstrapMS                float64 `json:"bootstrap_ms"`
	OnlineMS                   float64 `json:"online_ms"`
	WallMS                     float64 `json:"wall_ms"`
	KeyAggMS                   float64 `json:"keyagg_ms"`
	CoordinatorSign1Success    float64 `json:"coordinator_sign1_success_ms"`
	CoordinatorSign1Total      float64 `json:"coordinator_sign1_total_ms"`
	CoordinatorSign2Success    float64 `json:"coordinator_sign2_success_ms"`
	CoordinatorSign2Total      float64 `json:"coordinator_sign2_total_ms"`
	Round1CollectSuccessMS     float64 `json:"round1_collect_success_ms"`
	Round1BroadcastSuccessMS   float64 `json:"round1_broadcast_success_ms"`
	Round1CollectTotalMS       float64 `json:"round1_collect_total_ms"`
	Round1BroadcastTotalMS     float64 `json:"round1_broadcast_total_ms"`
	PartialCollectSuccessMS    float64 `json:"partial_collect_success_ms"`
	DecisionBroadcastSuccessMS float64 `json:"decision_broadcast_success_ms"`
	PartialCollectTotalMS      float64 `json:"partial_collect_total_ms"`
	DecisionBroadcastTotalMS   float64 `json:"decision_broadcast_total_ms"`
	Sign1NetworkSuccessMS      float64 `json:"sign1_network_success_ms"`
	Sign2NetworkSuccessMS      float64 `json:"sign2_network_success_ms"`
	Sign1E2ESuccessMS          float64 `json:"sign1_e2e_success_ms"`
	Sign2E2ESuccessMS          float64 `json:"sign2_e2e_success_ms"`
	SAggMS                     float64 `json:"sagg_ms"`
	VerifyMS                   float64 `json:"verify_ms"`
	BytesIn                    int     `json:"bytes_in"`
	BytesOut                   int     `json:"bytes_out"`
}

type networkSignerSummary struct {
	Role              string  `json:"role"`
	SignerID          int     `json:"signer_id"`
	CoordinatorPublic string  `json:"coordinator_public"`
	Message           string  `json:"message"`
	ParameterSet      string  `json:"parameter_set"`
	Q                 uint64  `json:"q"`
	SignerM           float64 `json:"signer_m"`
	TargetRejectionMn float64 `json:"target_rejection_mn"`
	SuccessAttempt    int     `json:"success_attempt"`
	RejectedAttempts  int     `json:"rejected_attempts"`
	PreSigningMS      float64 `json:"pre_signing_ms"`
	OnlineMS          float64 `json:"online_ms"`
	WallMS            float64 `json:"wall_ms"`
	KeyGenMS          float64 `json:"keygen_ms"`
	KeyAggMS          float64 `json:"keyagg_ms"`
	LocalSign1TotalMS float64 `json:"local_sign1_total_ms"`
	LocalSign2TotalMS float64 `json:"local_sign2_total_ms"`
	BytesIn           int     `json:"bytes_in"`
	BytesOut          int     `json:"bytes_out"`
}

func printNetworkSummary(summary any) {
	data, err := json.Marshal(summary)
	if err != nil {
		fmt.Printf("NETWORK_SUMMARY_MARSHAL_ERROR %v\n", err)
		return
	}
	fmt.Printf("NETWORK_SUMMARY %s\n", data)
}

func printNetworkCoordinatorSummaryTable(summary networkCoordinatorSummary) {
	rows := []networkSummaryRow{
		{"role", summary.Role},
		{"signers", fmt.Sprintf("%v", summary.SignerIDs)},
		{"parameter_set", summary.ParameterSet},
		{"q", fmt.Sprintf("%d", summary.Q)},
		{"signer_m", fmt.Sprintf("%.6f", summary.SignerM)},
		{"target_rejection_mn", fmt.Sprintf("%.6f", summary.TargetRejectionMn)},
		{"success_attempt", fmt.Sprintf("%d", summary.SuccessAttempt)},
		{"rejected_attempts", fmt.Sprintf("%d", summary.RejectedAttempts)},
		{"bootstrap_ms", fmt.Sprintf("%.3f", summary.BootstrapMS)},
		{"online_ms", fmt.Sprintf("%.3f", summary.OnlineMS)},
		{"wall_ms", fmt.Sprintf("%.3f", summary.WallMS)},
		{"keyagg_ms", fmt.Sprintf("%.3f", summary.KeyAggMS)},
		{"coord_sign1_success_ms", fmt.Sprintf("%.3f", summary.CoordinatorSign1Success)},
		{"coord_sign1_total_ms", fmt.Sprintf("%.3f", summary.CoordinatorSign1Total)},
		{"coord_sign2_success_ms", fmt.Sprintf("%.3f", summary.CoordinatorSign2Success)},
		{"coord_sign2_total_ms", fmt.Sprintf("%.3f", summary.CoordinatorSign2Total)},
		{"round1_collect_success_ms", fmt.Sprintf("%.3f", summary.Round1CollectSuccessMS)},
		{"round1_broadcast_success_ms", fmt.Sprintf("%.3f", summary.Round1BroadcastSuccessMS)},
		{"round1_collect_total_ms", fmt.Sprintf("%.3f", summary.Round1CollectTotalMS)},
		{"round1_broadcast_total_ms", fmt.Sprintf("%.3f", summary.Round1BroadcastTotalMS)},
		{"partial_collect_success_ms", fmt.Sprintf("%.3f", summary.PartialCollectSuccessMS)},
		{"decision_broadcast_success_ms", fmt.Sprintf("%.3f", summary.DecisionBroadcastSuccessMS)},
		{"partial_collect_total_ms", fmt.Sprintf("%.3f", summary.PartialCollectTotalMS)},
		{"decision_broadcast_total_ms", fmt.Sprintf("%.3f", summary.DecisionBroadcastTotalMS)},
		{"sign1_network_success_ms", fmt.Sprintf("%.3f", summary.Sign1NetworkSuccessMS)},
		{"sign2_network_success_ms", fmt.Sprintf("%.3f", summary.Sign2NetworkSuccessMS)},
		{"sign1_e2e_success_ms", fmt.Sprintf("%.3f", summary.Sign1E2ESuccessMS)},
		{"sign2_e2e_success_ms", fmt.Sprintf("%.3f", summary.Sign2E2ESuccessMS)},
		{"sagg_ms", fmt.Sprintf("%.3f", summary.SAggMS)},
		{"verify_ms", fmt.Sprintf("%.3f", summary.VerifyMS)},
		{"bytes_in", fmt.Sprintf("%d", summary.BytesIn)},
		{"bytes_out", fmt.Sprintf("%d", summary.BytesOut)},
	}
	printNetworkTable("Coordinator Summary", rows)
}

func printNetworkSignerSummaryTable(summary networkSignerSummary) {
	rows := []networkSummaryRow{
		{"role", summary.Role},
		{"signer_id", fmt.Sprintf("%d", summary.SignerID)},
		{"coordinator_public", summary.CoordinatorPublic},
		{"parameter_set", summary.ParameterSet},
		{"q", fmt.Sprintf("%d", summary.Q)},
		{"signer_m", fmt.Sprintf("%.6f", summary.SignerM)},
		{"target_rejection_mn", fmt.Sprintf("%.6f", summary.TargetRejectionMn)},
		{"success_attempt", fmt.Sprintf("%d", summary.SuccessAttempt)},
		{"rejected_attempts", fmt.Sprintf("%d", summary.RejectedAttempts)},
		{"pre_signing_ms", fmt.Sprintf("%.3f", summary.PreSigningMS)},
		{"online_ms", fmt.Sprintf("%.3f", summary.OnlineMS)},
		{"wall_ms", fmt.Sprintf("%.3f", summary.WallMS)},
		{"keygen_ms", fmt.Sprintf("%.3f", summary.KeyGenMS)},
		{"keyagg_ms", fmt.Sprintf("%.3f", summary.KeyAggMS)},
		{"local_sign1_total_ms", fmt.Sprintf("%.3f", summary.LocalSign1TotalMS)},
		{"local_sign2_total_ms", fmt.Sprintf("%.3f", summary.LocalSign2TotalMS)},
		{"bytes_in", fmt.Sprintf("%d", summary.BytesIn)},
		{"bytes_out", fmt.Sprintf("%d", summary.BytesOut)},
	}
	printNetworkTable("Signer Summary", rows)
}

type networkSummaryRow struct {
	key   string
	value string
}

func printNetworkTable(title string, rows []networkSummaryRow) {
	keyWidth := len("Metric")
	valueWidth := len("Value")
	for _, row := range rows {
		if len(row.key) > keyWidth {
			keyWidth = len(row.key)
		}
		if len(row.value) > valueWidth {
			valueWidth = len(row.value)
		}
	}

	border := fmt.Sprintf("+-%s-+-%s-+", strings.Repeat("-", keyWidth), strings.Repeat("-", valueWidth))
	fmt.Println(border)
	fmt.Printf("| %-*s | %-*s |\n", keyWidth, title, valueWidth, "")
	fmt.Println(border)
	fmt.Printf("| %-*s | %-*s |\n", keyWidth, "Metric", valueWidth, "Value")
	fmt.Println(border)
	for _, row := range rows {
		fmt.Printf("| %-*s | %-*s |\n", keyWidth, row.key, valueWidth, row.value)
	}
	fmt.Println(border)
}
