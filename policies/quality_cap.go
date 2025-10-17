package policies

import (
	policy "github.com/ArieDeha/ccxpolicy"
)

// QualityCap enforces that "transcode.targetQuality" must not exceed 1080.
// If it does, it adjusts it to 1080 across the subtree.
type QualityCap struct{}

func (QualityCap) ID() string               { return "cap_quality" }
func (QualityCap) Priority() int            { return 10 } // runs after safety (lower number) if needed
func (QualityCap) Match(n policy.Node) bool { return n.Name() == "Transcode" }

func (QualityCap) Check(n policy.Node) []policy.Decision {
	q, _ := n.Params()["transcode.targetQuality"].(int)
	if q > 1080 {
		return []policy.Decision{{
			PolicyID: "cap_quality",
			Scope:    policy.ScopeSubtree,
			Action:   policy.ActionAdjust,
			Adjust: func(p map[string]any) {
				p["transcode.targetQuality"] = 1080
			},
			Reason: policy.Reason("quality above cap; forcing 1080"),
			Stop:   false,
		}}
	}
	return nil
}
