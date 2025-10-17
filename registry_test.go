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

package ccxpolicy_test

import (
	"fmt"
	"reflect"
	"testing"

	policy "github.com/ArieDeha/ccxpolicy"
)

type testNode struct {
	id, name string
	params   map[string]any
	parent   *testNode
}

func (n *testNode) ID() string             { return n.id }
func (n *testNode) Name() string           { return n.name }
func (n *testNode) Params() map[string]any { return n.params }
func (n *testNode) Parent() policy.Node {
	if n.parent == nil {
		return nil
	}
	return n.parent
}
func (n *testNode) Root() policy.Node {
	cur := n
	for cur.parent != nil {
		cur = cur.parent
	}
	return cur
}

type policyA struct{}

func (policyA) ID() string             { return "A" }
func (policyA) Priority() int          { return 10 }
func (policyA) Match(policy.Node) bool { return true }
func (policyA) Check(policy.Node) []policy.Decision {
	return []policy.Decision{{
		PolicyID: "A",
		Scope:    policy.ScopeNode,
		Action:   policy.ActionWarn,
		Reason:   policy.Reason("warn A"),
	}}
}

type policyBStop struct{}

func (policyBStop) ID() string             { return "B" }
func (policyBStop) Priority() int          { return 5 } // runs before A
func (policyBStop) Match(policy.Node) bool { return true }
func (policyBStop) Check(policy.Node) []policy.Decision {
	return []policy.Decision{{
		PolicyID: "B",
		Scope:    policy.ScopeRoot,
		Action:   policy.ActionAdjust,
		Adjust:   func(m map[string]any) { m["x"] = 1 },
		Stop:     true, // short-circuit
		Reason:   policy.Reason("stop"),
	}}
}

func TestRegisterAndEvaluateOrderAndStop(t *testing.T) {
	// NOTE: registry is process-global; tests assume fresh process.
	policy.RegisterPolicy(policyA{})
	policy.RegisterPolicy(policyBStop{})

	n := &testNode{id: "n1", name: "N", params: map[string]any{}}
	ds := policy.Evaluate(n)

	if len(ds) != 1 {
		t.Fatalf("expected 1 decision due to Stop, got %d", len(ds))
	}
	if ds[0].PolicyID != "B" || ds[0].Action != policy.ActionAdjust {
		t.Fatalf("unexpected decision %+v", ds[0])
	}
}

type recEnforcer struct {
	adjusts []policy.Scope
	cancels []struct {
		s policy.Scope
	}
	warns []string
}

func (r *recEnforcer) Adjust(s policy.Scope, fn func(map[string]any)) {
	r.adjusts = append(r.adjusts, s)
	m := map[string]any{}
	fn(m) // ensure callable
}
func (r *recEnforcer) Cancel(s policy.Scope, _ error) {
	r.cancels = append(r.cancels, struct{ s policy.Scope }{s})
}
func (r *recEnforcer) Warn(id string, _ error) {
	r.warns = append(r.warns, id)
}

func TestEnforceMapping(t *testing.T) {
	e := &recEnforcer{}
	ds := []policy.Decision{
		{PolicyID: "W", Action: policy.ActionWarn},
		{PolicyID: "A", Action: policy.ActionAdjust, Scope: policy.ScopeSubtree, Adjust: func(m map[string]any) { m["y"] = 2 }},
		{PolicyID: "C", Action: policy.ActionCancelRoot, Scope: policy.ScopeRoot},
	}
	policy.Enforce(e, ds)

	if !reflect.DeepEqual(e.warns, []string{"W"}) {
		t.Fatalf("warns not recorded: %v", e.warns)
	}
	if len(e.adjusts) != 1 || e.adjusts[0] != policy.ScopeSubtree {
		t.Fatalf("adjust not recorded with correct scope: %v", e.adjusts)
	}
	if len(e.cancels) != 1 || e.cancels[0].s != policy.ScopeRoot {
		t.Fatalf("cancel not recorded with correct scope: %+v", e.cancels)
	}
}

// --- Examples ---

type demoEnf struct{}

func (demoEnf) Adjust(s policy.Scope, fn func(map[string]any)) {
	p := map[string]any{"q": 1440}
	fn(p)
	fmt.Println("adjust", s, p["q"])
}
func (demoEnf) Cancel(s policy.Scope, _ error) { fmt.Println("cancel", s) }
func (demoEnf) Warn(id string, _ error)        { fmt.Println("warn", id) }

// ExampleEnforce demonstrates applying a set of decisions with a custom Enforcer.
func ExampleEnforce() {

	ds := []policy.Decision{
		{PolicyID: "W", Action: policy.ActionWarn},
		{PolicyID: "A", Action: policy.ActionAdjust, Scope: policy.ScopeNode, Adjust: func(m map[string]any) { m["q"] = 1080 }},
		{PolicyID: "C", Action: policy.ActionCancelSubtree, Scope: policy.ScopeSubtree},
	}

	policy.Enforce(demoEnf{}, ds)
	// Output:
	// warn W
	// adjust 0 1080
	// cancel 1
}
