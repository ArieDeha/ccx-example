package policies

import (
	policy "github.com/ArieDeha/ccxpolicy"
)

// SafetyStop cancels the root if a global "safety.block" flag is set in params.
// Demonstrates a cross-cutting policy that applies to all intents.
type SafetyStop struct{}

func (SafetyStop) ID() string               { return "safety_stop" }
func (SafetyStop) Priority() int            { return 5 }    // runs before QualityCap
func (SafetyStop) Match(n policy.Node) bool { return true } // applies to all nodes

func (SafetyStop) Check(n policy.Node) []policy.Decision {
	if v, ok := n.Params()["safety.block"].(bool); ok && v {
		return []policy.Decision{{
			PolicyID: "safety_stop",
			Scope:    policy.ScopeRoot,
			Action:   policy.ActionCancelRoot,
			Reason:   policy.Reason("safety override engaged"),
			Stop:     true, // stop after this
		}}
	}
	return nil
}
