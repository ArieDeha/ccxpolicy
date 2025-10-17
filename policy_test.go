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
	"testing"

	policy "github.com/ArieDeha/ccxpolicy"
)

func TestReason(t *testing.T) {
	err := policy.Reason("hello")
	if err == nil || err.Error() != "hello" {
		t.Fatalf("expected error 'hello', got: %v", err)
	}
}

// Compile-time interface checks via dummy implementations.

type _dummyNode struct{}

func (_dummyNode) ID() string             { return "id" }
func (_dummyNode) Name() string           { return "name" }
func (_dummyNode) Params() map[string]any { return map[string]any{"k": "v"} }
func (_dummyNode) Parent() policy.Node    { return nil }
func (_dummyNode) Root() policy.Node      { return _dummyNode{} }

type _dummyPolicy struct{}

func (_dummyPolicy) ID() string                          { return "p" }
func (_dummyPolicy) Priority() int                       { return 0 }
func (_dummyPolicy) Match(policy.Node) bool              { return true }
func (_dummyPolicy) Check(policy.Node) []policy.Decision { return nil }

func TestTypesCompile(t *testing.T) {
	var _ policy.Node = _dummyNode{}
	var _ policy.Policy = _dummyPolicy{}
}

// --- Examples ---

// ExampleReason shows creating a Reason error.
func ExampleReason() {
	err := policy.Reason("quality above cap")
	fmt.Println(err.Error())
	// Output: quality above cap
}

// ExampleDecision_adjust demonstrates invoking a Decision's Adjust function.
func ExampleDecision_adjust() {
	params := map[string]any{"q": 1440}
	d := policy.Decision{
		Action: policy.ActionAdjust,
		Adjust: func(p map[string]any) { p["q"] = 1080 },
	}
	if d.Adjust != nil {
		d.Adjust(params)
	}
	fmt.Println(params["q"])
	// Output: 1080
}
