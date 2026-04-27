package network

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	networkRoleCoordinatorSigner = "coordinator_signer"
	networkRoleSigner            = "signer"
	defaultNetworkDialTimeoutMS  = 10000
)

type NetworkPartyConfig struct {
	ID   int    `json:"id"`
	Role string `json:"role"`
}

type NetworkConfig struct {
	NumSigners         int                  `json:"num_signers"`
	CoordinatorID      int                  `json:"coordinator_id"`
	ParameterProfile   int                  `json:"parameter_profile"`
	TargetRejectionMn  float64              `json:"target_rejection_mn"`
	MaxSigningAttempts int                  `json:"max_signing_attempts"`
	Message            string               `json:"message"`
	CoordinatorListen  string               `json:"coordinator_listen"`
	CoordinatorPublic  string               `json:"coordinator_public"`
	DialTimeoutMillis  int                  `json:"dial_timeout_millis"`
	Parties            []NetworkPartyConfig `json:"parties"`
}

func LoadNetworkConfig(path string) (NetworkConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return NetworkConfig{}, fmt.Errorf("read network config: %w", err)
	}

	var cfg NetworkConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return NetworkConfig{}, fmt.Errorf("parse network config: %w", err)
	}
	cfg.applyNetworkDefaults()
	if err := cfg.Validate(); err != nil {
		return NetworkConfig{}, err
	}
	return cfg, nil
}

func (cfg *NetworkConfig) applyNetworkDefaults() {
	if cfg.ParameterProfile == 0 {
		cfg.ParameterProfile = 32
	}
	if cfg.MaxSigningAttempts == 0 {
		cfg.MaxSigningAttempts = DefaultConfig().MaxSigningAttempts
	}
	if cfg.Message == "" {
		cfg.Message = fmt.Sprintf("dualms network demo active_n%d using n%d profile", cfg.NumSigners, cfg.ParameterProfile)
	}
	if cfg.DialTimeoutMillis == 0 {
		cfg.DialTimeoutMillis = defaultNetworkDialTimeoutMS
	}
}

func (cfg NetworkConfig) Validate() error {
	if cfg.NumSigners < 2 {
		return fmt.Errorf("network num_signers must be at least 2")
	}
	if cfg.CoordinatorID < 0 || cfg.CoordinatorID >= cfg.NumSigners {
		return fmt.Errorf("coordinator_id %d is outside signer range [0,%d)", cfg.CoordinatorID, cfg.NumSigners)
	}
	if cfg.ParameterProfile != 8 && cfg.ParameterProfile != 32 && cfg.ParameterProfile != 1024 {
		return fmt.Errorf("unsupported DualMS parameter_profile %d; want 8, 32, or 1024", cfg.ParameterProfile)
	}
	if cfg.ParameterProfile == 8 && cfg.NumSigners != 8 {
		return fmt.Errorf("DualMS parameter_profile 8 is reserved for the n=8 network experiment, but num_signers=%d", cfg.NumSigners)
	}
	if cfg.TargetRejectionMn < 0 {
		return fmt.Errorf("invalid target_rejection_mn: %v", cfg.TargetRejectionMn)
	}
	if cfg.MaxSigningAttempts <= 0 {
		return fmt.Errorf("max_signing_attempts must be positive")
	}
	if cfg.DialTimeoutMillis <= 0 {
		return fmt.Errorf("dial_timeout_millis must be positive")
	}
	if cfg.CoordinatorListen == "" {
		return fmt.Errorf("coordinator_listen must not be empty")
	}
	if cfg.CoordinatorPublic == "" {
		return fmt.Errorf("coordinator_public must not be empty")
	}
	if len(cfg.Parties) != cfg.NumSigners {
		return fmt.Errorf("parties length is %d, want num_signers=%d", len(cfg.Parties), cfg.NumSigners)
	}

	seen := make(map[int]string, len(cfg.Parties))
	for _, party := range cfg.Parties {
		if party.ID < 0 || party.ID >= cfg.NumSigners {
			return fmt.Errorf("party id %d is outside signer range [0,%d)", party.ID, cfg.NumSigners)
		}
		if _, ok := seen[party.ID]; ok {
			return fmt.Errorf("duplicate party id: %d", party.ID)
		}
		if party.Role != networkRoleCoordinatorSigner && party.Role != networkRoleSigner {
			return fmt.Errorf("party %d has unsupported role %q", party.ID, party.Role)
		}
		if party.ID == cfg.CoordinatorID && party.Role != networkRoleCoordinatorSigner {
			return fmt.Errorf("coordinator party %d must have role %q", party.ID, networkRoleCoordinatorSigner)
		}
		if party.ID != cfg.CoordinatorID && party.Role != networkRoleSigner {
			return fmt.Errorf("non-coordinator party %d must have role %q", party.ID, networkRoleSigner)
		}
		seen[party.ID] = party.Role
	}
	if _, ok := seen[cfg.CoordinatorID]; !ok {
		return fmt.Errorf("coordinator party %d is missing from parties", cfg.CoordinatorID)
	}
	return nil
}

func (cfg NetworkConfig) ProtocolConfig() (Config, error) {
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	protocolCfg := DefaultConfig()
	if err := protocolCfg.ApplyDualMSNetworkParameterProfile(cfg.ParameterProfile); err != nil {
		return Config{}, err
	}
	if cfg.TargetRejectionMn > 0 {
		protocolCfg.TargetRejectionMn = cfg.TargetRejectionMn
	}
	protocolCfg.MaxSigningAttempts = cfg.MaxSigningAttempts
	protocolCfg.Message = cfg.Message
	protocolCfg.RefreshDerived()
	return protocolCfg, nil
}

func (cfg NetworkConfig) PartyRole(id int) (string, bool) {
	for _, party := range cfg.Parties {
		if party.ID == id {
			return party.Role, true
		}
	}
	return "", false
}
