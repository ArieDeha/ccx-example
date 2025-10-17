package policies_test

import (
	"fmt"
	"testing"

	stMain "github.com/ArieDeha/ccx-example/policies"

	policy "github.com/ArieDeha/ccxpolicy"
)

// minimal node adapter for tests
type ssTestNode struct {
	id     string
	name   string
	params map[string]any
	parent *ssTestNode
}

func (n *ssTestNode) ID() string             { return n.id }
func (n *ssTestNode) Name() string           { return n.name }
func (n *ssTestNode) Params() map[string]any { return n.params }
func (n *ssTestNode) Parent() policy.Node {
	if n.parent == nil {
		return nil
	}
	return n.parent
}
func (n *ssTestNode) Root() policy.Node {
	cur := n
	for cur.parent != nil {
		cur = cur.parent
	}
	return cur
}

func TestSafetyStop_CancelsRootWhenBlocked(t *testing.T) {
	ss := stMain.SafetyStop{}
	root := &ssTestNode{
		id:     "root",
		name:   "Any",
		params: map[string]any{"safety.block": true},
	}
	if !ss.Match(root) {
		t.Fatalf("expected SafetyStop to match all nodes")
	}
	ds := ss.Check(root)
	if len(ds) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(ds))
	}
	d := ds[0]
	if d.Action != policy.ActionCancelRoot || d.Scope != policy.ScopeRoot || !d.Stop {
		t.Fatalf("unexpected decision: %+v", d)
	}
}

func TestSafetyStop_NoDecisionWhenNotBlocked(t *testing.T) {
	ss := stMain.SafetyStop{}
	node := &ssTestNode{
		id:     "n",
		name:   "Any",
		params: map[string]any{"safety.block": false},
	}
	ds := ss.Check(node)
	if len(ds) != 0 {
		t.Fatalf("expected 0 decisions, got %d", len(ds))
	}
}

// ExampleSafetyStop demonstrates emitting a cancel-root decision when blocked.
func ExampleSafetyStop() {
	ss := stMain.SafetyStop{}
	node := &ssTestNode{
		id:     "n",
		name:   "Any",
		params: map[string]any{"safety.block": true},
	}
	ds := ss.Check(node)
	// Print a short summary: count, action, scope, stop
	if len(ds) > 0 {
		fmt.Println(len(ds), int(ds[0].Action), int(ds[0].Scope), ds[0].Stop)
	} else {
		fmt.Println(0)
	}
	// Output: 1 5 2 true
}

// ExampleSafetyStop_noop shows no decisions when not blocked.
func ExampleSafetyStop_noop() {
	ss := stMain.SafetyStop{}
	node := &ssTestNode{
		id:     "n",
		name:   "Any",
		params: map[string]any{"safety.block": false},
	}
	ds := ss.Check(node)
	fmt.Println(len(ds))
	// Output: 0
}
