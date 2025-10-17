package policies_test

import (
	"fmt"
	"testing"

	policy "github.com/ArieDeha/ccxpolicy"

	qcMain "github.com/ArieDeha/ccx-example/policies"
)

// minimal node adapter for tests
type qcTestNode struct {
	id     string
	name   string
	params map[string]any
	parent *qcTestNode
}

func (n *qcTestNode) ID() string             { return n.id }
func (n *qcTestNode) Name() string           { return n.name }
func (n *qcTestNode) Params() map[string]any { return n.params }
func (n *qcTestNode) Parent() policy.Node {
	if n.parent == nil {
		return nil
	}
	return n.parent
}
func (n *qcTestNode) Root() policy.Node {
	cur := n
	for cur.parent != nil {
		cur = cur.parent
	}
	return cur
}

func TestQualityCap_AdjustsWhenAboveCap(t *testing.T) {
	qc := qcMain.QualityCap{}
	node := &qcTestNode{
		id:   "n1",
		name: "Transcode",
		params: map[string]any{
			"transcode.targetQuality": 1440,
		},
	}
	if !qc.Match(node) {
		t.Fatalf("expected QualityCap to match Transcode")
	}
	ds := qc.Check(node)
	if len(ds) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(ds))
	}
	d := ds[0]
	if d.Action != policy.ActionAdjust || d.Scope != policy.ScopeSubtree {
		t.Fatalf("unexpected decision: %+v", d)
	}
	// Apply adjust and verify
	params := map[string]any{"transcode.targetQuality": 1440}
	if d.Adjust == nil {
		t.Fatalf("expected adjust function")
	}
	d.Adjust(params)
	if got := params["transcode.targetQuality"]; got != 1080 {
		t.Fatalf("expected 1080 after adjust, got %v", got)
	}
}

func TestQualityCap_NoDecisionWhenAtOrBelowCap(t *testing.T) {
	qc := qcMain.QualityCap{}
	for _, q := range []int{1080, 720} {
		node := &qcTestNode{
			id:     "n2",
			name:   "Transcode",
			params: map[string]any{"transcode.targetQuality": q},
		}
		ds := qc.Check(node)
		if len(ds) != 0 {
			t.Fatalf("q=%d expected 0 decisions, got %d", q, len(ds))
		}
	}
}

// ExampleQualityCap demonstrates capping quality to 1080 when above cap.
func ExampleQualityCap() {
	qc := qcMain.QualityCap{}
	node := &qcTestNode{
		id:   "n1",
		name: "Transcode",
		params: map[string]any{
			"transcode.targetQuality": 1440,
		},
	}
	ds := qc.Check(node)
	if len(ds) > 0 && ds[0].Adjust != nil {
		ds[0].Adjust(node.params)
	}
	fmt.Println(node.params["transcode.targetQuality"])
	// Output: 1080
}

// ExampleQualityCap_noop shows no decision when within cap.
func ExampleQualityCap_noop() {
	qc := qcMain.QualityCap{}
	node := &qcTestNode{
		id:     "n2",
		name:   "Transcode",
		params: map[string]any{"transcode.targetQuality": 720},
	}
	ds := qc.Check(node)
	fmt.Println(len(ds))
	// Output: 0
}
