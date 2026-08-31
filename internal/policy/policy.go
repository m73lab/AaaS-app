package policy

import "strings"

// Action is the transformation applied to a detected entity.
type Action string

const (
	// ActionReversible replaces the value with a token stored in the vault,
	// allowing the response to be de-anonymized back to the original.
	ActionReversible Action = "reversible"
	// ActionFPE replaces the value with a deterministic, format-preserving
	// token computed without any stored state (vault-less pseudonymization).
	ActionFPE Action = "fpe"
	// ActionRedact replaces the value with a fixed mask (e.g. "****").
	ActionRedact Action = "redact"
	// ActionHash replaces the value with a deterministic keyed hash.
	ActionHash Action = "hash"
	// ActionBlock rejects the request entirely (fail-closed / never egress).
	ActionBlock Action = "block"
)

// Rule maps a set of entity categories to an action.
type Rule struct {
	Categories []string
	Action     Action
}

// Policy decides which action applies to each detected entity category.
type Policy struct {
	Default Action
	Rules   []Rule
}

// ActionFor returns the configured action for a category, falling back to the
// default when no rule matches.
func (p *Policy) ActionFor(category string) Action {
	for _, r := range p.Rules {
		for _, c := range r.Categories {
			if strings.EqualFold(c, category) {
				return r.Action
			}
		}
	}
	return p.Default
}
