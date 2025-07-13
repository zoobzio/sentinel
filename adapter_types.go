package catalog

// Adapter-specific metadata types for pipz contracts

// CerealMetadata represents metadata extracted by cereal adapter
type CerealMetadata struct {
	Scopes              []string `json:"scopes,omitempty"`
	RedactionStrategy   string   `json:"redaction_strategy,omitempty"`
	RedactionValue      string   `json:"redaction_value,omitempty"`
	EncryptionType      string   `json:"encryption_type,omitempty"`
	EncryptionAlgorithm string   `json:"encryption_algorithm,omitempty"`
	DataResidency       []string `json:"data_residency,omitempty"`
}

// ValidatorMetadata represents metadata extracted by validator adapter
type ValidatorMetadata struct {
	Required    bool              `json:"required,omitempty"`
	CustomRules []string          `json:"custom_rules,omitempty"`
	Constraints map[string]string `json:"constraints,omitempty"`
}

// ZlogMetadata represents metadata extracted by zlog adapter
type ZlogMetadata struct {
	Sensitive bool   `json:"sensitive,omitempty"`
	Level     string `json:"level,omitempty"`
	Exclude   bool   `json:"exclude,omitempty"`
}

// RoccoMetadata represents metadata extracted by rocco adapter
type RoccoMetadata struct {
	AuthRequired bool     `json:"auth_required,omitempty"`
	Roles        []string `json:"roles,omitempty"`
	Permissions  []string `json:"permissions,omitempty"`
}

// DatabaseMetadata represents metadata extracted by database adapter
type DatabaseMetadata struct {
	Column      string `json:"column,omitempty"`
	PrimaryKey  bool   `json:"primary_key,omitempty"`
	ForeignKey  string `json:"foreign_key,omitempty"`
	Index       bool   `json:"index,omitempty"`
	Unique      bool   `json:"unique,omitempty"`
	Nullable    bool   `json:"nullable,omitempty"`
}