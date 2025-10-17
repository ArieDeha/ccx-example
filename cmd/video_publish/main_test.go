package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	ccx "github.com/ArieDeha/ccx"
	policy "github.com/ArieDeha/ccxpolicy"

	"github.com/ArieDeha/ccx-example/policies"
)

// Ensure we only register policies once per test process.
var registerOnce sync.Once

func registerPoliciesOnce() {
	registerOnce.Do(func() {
		registerPolicies()
	})
}

func TestPublishFlow_Completes(t *testing.T) {
	registerPoliciesOnce()

	// Build the same high-level structure as main(), but assert completion.
	root := ccx.Background()
	publish, cancel := ccx.WithIntent(root, ccx.Intent{
		Name: "PublishVideo",
		Params: map[string]any{
			"videoID":                 "VID-42",
			"safety.block":            false,
			"transcode.targetQuality": 1440, // will be adjusted by policy
		},
	}, ccx.Constraints{Deadline: time.Now().Add(2 * time.Second)})
	defer cancel()

	// L1
	tx, _ := ccx.WithIntent(publish, ccx.Intent{
		Name: "Transcode",
		Params: map[string]any{
			"segmentCount":            3,
			"segmentMs":               200,
			"transcode.targetQuality": 1440,
		},
	}, ccx.Constraints{})
	th, _ := ccx.WithIntent(publish, ccx.Intent{
		Name:   "Thumbnail",
		Params: map[string]any{"frames": 2, "sizes": []int{120, 320}},
	}, ccx.Constraints{})
	cdn, _ := ccx.WithIntent(publish, ccx.Intent{Name: "CDNPush"}, ccx.Constraints{})

	// Apply policies deterministically (no logging in tests).
	for _, n := range []*ccx.Ctx{tx, th, cdn} {
		ds := ccx.EvaluatePolicies(n)
		ccx.EnforcePolicies(n, ds)
	}

	// Handlers (reuse app handlers for realism).
	go handleTranscode(tx)
	go handleThumbnail(th)
	go handleCDN(cdn)

	// Wait L1, then fulfill publish and assert 'done'.
	if err := ccx.WaitAll(context.Background(), th, tx, cdn); err != nil {
		t.Fatalf("children failed: %v", err)
	}
	publish.Fulfill()

	select {
	case <-publish.DoneChan():
		if st := publish.State(); st != "done" {
			t.Fatalf("publish not done, state=%s", st)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for publish completion")
	}
}

func TestSafetyPolicy_CancelsRoot(t *testing.T) {
	registerPoliciesOnce()

	root := ccx.Background()
	node, cancel := ccx.WithIntent(root, ccx.Intent{
		Name:   "Anything",
		Params: map[string]any{"safety.block": true},
	}, ccx.Constraints{Deadline: time.Now().Add(500 * time.Millisecond)})
	defer cancel()

	// Evaluate & enforce -> should cancel root due to SafetyStop policy.
	ds := ccx.EvaluatePolicies(node)
	ccx.EnforcePolicies(node, ds)

	select {
	case <-root.DoneChan():
		if st := root.State(); st != "aborted" {
			t.Fatalf("expected root aborted, got %s", st)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("root was not canceled by safety policy")
	}
}

// -------------------------
// Example functions (docs)
// -------------------------

// Example_registerPolicies_and_evaluate shows registering app policies
// and evaluating a node that triggers an adjustment.
func Example_registerPolicies_and_evaluate() {
	// Register app policies (idempotent for the example).
	policy.RegisterPolicy(policies.SafetyStop{})
	policy.RegisterPolicy(policies.QualityCap{})

	// Build a small node that should trigger QualityCap (1440 -> 1080).
	root := ccx.Background()
	tx, _ := ccx.WithIntent(root, ccx.Intent{
		Name:   "Transcode",
		Params: map[string]any{"transcode.targetQuality": 1440},
	}, ccx.Constraints{})

	ds := ccx.EvaluatePolicies(tx)
	fmt.Println(len(ds) >= 1) // at least the QualityCap decision
	// Output: true
}

// Example_smallFlow demonstrates a minimal flow that ends in "done".
func Example_smallFlow() {
	root := ccx.Background()
	child, cancel := ccx.WithIntent(
		root,
		ccx.Intent{Name: "Task"},
		ccx.Constraints{Deadline: time.Now().Add(200 * time.Millisecond)},
	)
	defer cancel()

	// Simulate finishing work.
	child.Fulfill()
	<-child.DoneChan()
	fmt.Println(child.State())
	// Output: done
}
