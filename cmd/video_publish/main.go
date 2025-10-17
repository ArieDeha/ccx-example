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

package main

import (
	"context"
	"fmt"
	"time"

	ccx "github.com/ArieDeha/ccx"
	policy "github.com/ArieDeha/ccxpolicy"

	"github.com/ArieDeha/ccx-example/policies"
)

func registerPolicies() {
	// Register all application policies exactly once at startup.
	policy.RegisterPolicy(policies.SafetyStop{})
	policy.RegisterPolicy(policies.QualityCap{})
}

func main() {
	registerPolicies()

	// Root intent: PublishVideo
	root := ccx.Background()
	rootCons := ccx.Constraints{Deadline: time.Now().Add(6 * time.Second)}

	pubParams := map[string]any{
		"videoID":                 "VID-42",
		"targetQuality":           1080,
		"safety.block":            false, // flip to true to demo root cancel
		"transcode.targetQuality": 1440,  // deliberately above cap to trigger policy
	}
	publish, cancel := ccx.WithIntent(root, ccx.Intent{
		Name:   "PublishVideo",
		Params: pubParams,
	}, rootCons)
	defer cancel()

	// Level 1 children
	tx, _ := ccx.WithIntent(publish, ccx.Intent{
		Name: "Transcode",
		Params: map[string]any{
			"segmentCount":            3,
			"segmentMs":               250,
			"transcode.targetQuality": pubParams["transcode.targetQuality"], // inherit initial
		},
	}, ccx.Constraints{})

	th, _ := ccx.WithIntent(publish, ccx.Intent{
		Name: "Thumbnail",
		Params: map[string]any{
			"frames": 3,
			"sizes":  []int{120, 320},
		},
	}, ccx.Constraints{})

	cdn, _ := ccx.WithIntent(publish, ccx.Intent{Name: "CDNPush"}, ccx.Constraints{})

	// Immediately evaluate & enforce policies for the just-created L1 nodes
	enforcePolicies(tx, "after create: Transcode")
	enforcePolicies(th, "after create: Thumbnail")
	enforcePolicies(cdn, "after create: CDNPush")

	// Run handlers
	go handleTranscode(tx)
	go handleThumbnail(th)
	go handleCDN(cdn)

	// Wait for all L1 children, then finish root
	if err := ccx.WaitAll(context.Background(), th, tx, cdn); err != nil {
		fmt.Println("root children error:", err)
	}
	publish.Fulfill()
	fmt.Println("[root]", publish.State())
}

func enforcePolicies(n *ccx.Ctx, where string) {
	ds := ccx.EvaluatePolicies(n)
	if len(ds) > 0 {
		fmt.Printf("[policy] %s: %d decision(s)\n", where, len(ds))
	}
	ccx.EnforcePolicies(n, ds)
	// Optional peek: show effective params after enforcement
	fmt.Printf("[params] %s %s: %+v\n", n.ID(), n.Intent().Name, n.Intent().Params)
}

func handleTranscode(tx *ccx.Ctx) {
	// L2: Variants (1080, 720)
	v1, _ := ccx.WithIntent(tx, ccx.Intent{
		Name: "Variant",
		Params: map[string]any{
			"quality": 1080,
		},
	}, ccx.Constraints{})
	v2, _ := ccx.WithIntent(tx, ccx.Intent{
		Name: "Variant",
		Params: map[string]any{
			"quality": 720,
		},
	}, ccx.Constraints{})

	// Enforce policies at this level as well (e.g., safety may cancel)
	enforcePolicies(v1, "create: Variant-1080")
	enforcePolicies(v2, "create: Variant-720")

	// L3: Segments (simulate work)
	go doSegments(v1, 3, 220)
	go doSegments(v2, 3, 220)

	if err := ccx.WaitAll(context.Background(), v1, v2); err != nil {
		fmt.Println("[Transcode] error:", err)
	}
	tx.Fulfill()
	fmt.Println("[Transcode]", tx.State())
}

func doSegments(parent *ccx.Ctx, n, ms int) {
	children := make([]*ccx.Ctx, 0, n)
	for i := 0; i < n; i++ {
		ch, _ := ccx.WithIntent(parent, ccx.Intent{
			Name:   "Segment",
			Params: map[string]any{"idx": i},
		}, ccx.Constraints{})
		enforcePolicies(ch, fmt.Sprintf("create: Segment-%d", i))

		// Simulate segment work honoring cancellation.
		go func(c *ccx.Ctx) {
			select {
			case <-time.After(time.Duration(ms) * time.Millisecond):
				c.Fulfill()
				fmt.Printf("[Segment %d] done\n", c.Intent().Params["idx"])
			case <-c.DoneChan():
				fmt.Printf("[Segment %d] canceled: %v\n", c.Intent().Params["idx"], c.ErrState())
			}
		}(ch)
		children = append(children, ch)
	}
	_ = ccx.WaitAll(context.Background(), children...)
	parent.Fulfill()
	fmt.Println("[Variant]", parent.Intent().Params["quality"], "done")
}

func handleThumbnail(th *ccx.Ctx) {
	ex, _ := ccx.WithIntent(th, ccx.Intent{
		Name:   "ExtractFrame",
		Params: map[string]any{"frames": 3},
	}, ccx.Constraints{})
	enforcePolicies(ex, "create: ExtractFrame")

	r1, _ := ccx.WithIntent(th, ccx.Intent{
		Name:   "Resize",
		Params: map[string]any{"size": 120},
	}, ccx.Constraints{})
	enforcePolicies(r1, "create: Resize-120")

	r2, _ := ccx.WithIntent(th, ccx.Intent{
		Name:   "Resize",
		Params: map[string]any{"size": 320},
	}, ccx.Constraints{})
	enforcePolicies(r2, "create: Resize-320")

	go func() {
		select {
		case <-time.After(250 * time.Millisecond):
			ex.Fulfill()
			fmt.Println("[ExtractFrame] done")
		case <-ex.DoneChan():
		}
	}()
	go fulfillAfter(r1, 200)
	go fulfillAfter(r2, 200)

	if err := ccx.WaitAll(context.Background(), ex, r1, r2); err != nil {
		fmt.Println("[Thumbnail] error:", err)
	}
	th.Fulfill()
	fmt.Println("[Thumbnail]", th.State())
}

func handleCDN(c *ccx.Ctx) {
	enforcePolicies(c, "create: CDNPush")
	select {
	case <-time.After(600 * time.Millisecond):
		c.Fulfill()
		fmt.Println("[CDNPush] done")
	case <-c.DoneChan():
	}
}

func fulfillAfter(n *ccx.Ctx, ms int) {
	select {
	case <-time.After(time.Duration(ms) * time.Millisecond):
		n.Fulfill()
	default:
	}
}
