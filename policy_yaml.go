package sentinel

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Note: Policy loading functions are available but with singleton pattern,
// policies would need to be applied differently to the global instance.

// LoadPolicyFile loads a policy from a YAML file.
func LoadPolicyFile(path string) (Policy, error) {
	file, err := os.Open(path)
	if err != nil {
		return Policy{}, fmt.Errorf("failed to open policy file: %w", err)
	}
	defer file.Close()

	return LoadPolicy(file)
}

// LoadPolicyDir loads all YAML policy files from a directory.
func LoadPolicyDir(dir string) ([]Policy, error) {
	policies := make([]Policy, 0)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .yaml and .yml files
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		policy, err := LoadPolicyFile(path)
		if err != nil {
			// Log but continue with other files
			continue
		}

		policies = append(policies, policy)
	}

	return policies, nil
}

// LoadPolicy loads a policy from a reader.
func LoadPolicy(r io.Reader) (Policy, error) {
	var policy Policy

	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&policy); err != nil {
		return Policy{}, fmt.Errorf("failed to decode policy: %w", err)
	}

	// Validate the loaded policy
	if err := ValidatePolicy(policy); err != nil {
		return Policy{}, fmt.Errorf("invalid policy: %w", err)
	}

	return policy, nil
}

// ValidatePolicy checks if a policy is well-formed.
func ValidatePolicy(policy Policy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy must have a name")
	}

	if len(policy.Policies) == 0 {
		return fmt.Errorf("policy must have at least one type policy")
	}

	for i, tp := range policy.Policies {
		if tp.Match == "" {
			return fmt.Errorf("type policy %d must have a match pattern", i)
		}

		// Validate field policies
		for j, fp := range tp.Fields {
			if fp.Match == "" {
				return fmt.Errorf("field policy %d.%d must have a match pattern", i, j)
			}

			// At least one of Require or Apply should be set
			if len(fp.Require) == 0 && len(fp.Apply) == 0 {
				return fmt.Errorf("field policy %d.%d must have either require or apply rules", i, j)
			}
		}
	}

	return nil
}

// MarshalPolicy converts a policy to YAML.
func MarshalPolicy(policy Policy) ([]byte, error) {
	return yaml.Marshal(policy)
}
