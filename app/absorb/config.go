// Package absorb contains the configuration for record absorption/merging.
// This is a separate package to avoid import cycles between routes and hooks.
package absorb

import "fmt"

// Config defines the complete configuration for absorbing a collection.
// All absorb-related relationships are defined here in one place.
type Config struct {
	// RefConfigs defines which tables/columns get updated when absorbing this collection.
	// When records are absorbed, references in these tables are updated to point to the target.
	RefConfigs []RefConfig

	// ParentConstraint enforces that records being absorbed must share the same parent.
	// For example, client_contacts being absorbed must all belong to the same client.
	ParentConstraint *ParentConstraint

	// DependsOn specifies the parent collection in the absorb dependency graph.
	// If set:
	//   - This collection's COMMIT is blocked when the parent has a pending absorb
	//   - The parent's UNDO is blocked when this collection has a pending absorb
	// This prevents data inconsistencies where undoing a parent absorb would try to
	// restore references to records that were deleted by a committed child absorb.
	DependsOn string
}

type RefConfig struct {
	Table  string
	Column string
}

// ParentConstraint enforces data integrity during record absorption by ensuring
// that records being absorbed together must share the same parent relationship.
// This prevents merging records that don't logically belong together.
type ParentConstraint struct {
	Collection string // The parent collection that must match (e.g., "clients")
	Field      string // The field that must have matching values (e.g., "client")
}

// Configs is the single source of truth for all absorb configurations.
// Add new absorbable collections here.
var Configs = map[string]Config{
	"clients": {
		RefConfigs: []RefConfig{
			{"client_contacts", "client"},
			{"client_notes", "client"},
			{"jobs", "client"},
			{"jobs", "job_owner"},
		},
		// clients has no parent - it's at the top of the hierarchy
	},
	"client_contacts": {
		RefConfigs: []RefConfig{
			{"jobs", "contact"},
		},
		ParentConstraint: &ParentConstraint{
			Collection: "clients",
			Field:      "client",
		},
		// client_contacts depends on clients for absorb ordering:
		// - Can't commit client_contacts while clients absorb exists
		// - Can't undo clients while client_contacts absorb exists
		DependsOn: "clients",
	},
	"vendors": {
		RefConfigs: []RefConfig{
			{"purchase_orders", "vendor"},
			{"expenses", "vendor"},
		},
		// vendors is independent - no parent dependency
	},
}

// GetConfig returns the absorb configuration for a collection.
func GetConfig(collectionName string) (Config, error) {
	config, ok := Configs[collectionName]
	if !ok {
		return Config{}, fmt.Errorf("unknown collection: %s", collectionName)
	}
	return config, nil
}

// GetRefConfigs returns the reference configs and parent constraint for a collection.
func GetRefConfigs(collectionName string) ([]RefConfig, *ParentConstraint, error) {
	config, err := GetConfig(collectionName)
	if err != nil {
		return nil, nil, err
	}
	return config.RefConfigs, config.ParentConstraint, nil
}

// GetUndoBlockers returns collections that block undoing this collection's absorb.
// These are collections that have DependsOn pointing to this collection.
func GetUndoBlockers(collectionName string) []string {
	var blocks []string
	for name, config := range Configs {
		if config.DependsOn == collectionName {
			blocks = append(blocks, name)
		}
	}
	return blocks
}

// GetCommitBlockers returns collections that block committing this collection's absorb.
// This is simply the DependsOn field - if this collection depends on another, that other
// collection's absorb must be resolved before this one can be committed.
func GetCommitBlockers(collectionName string) []string {
	config, ok := Configs[collectionName]
	if !ok || config.DependsOn == "" {
		return nil
	}
	return []string{config.DependsOn}
}
