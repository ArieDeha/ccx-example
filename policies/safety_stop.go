// Copyright 2025 Arieditya Pramadyana Deha <arieditya.prdh@live.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
