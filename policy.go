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

// Package ccxpolicy defines a minimal, domain-neutral policy engine that can be
// used by any host runtime. Policies inspect a read-only Node view and emit
// Decisions (e.g., Adjust/Cancel/Warn). The host integrates via two roles:
//
//   - Node:    your runtime's read-only adapter (ID/Name/Params/lineage)
//   - Enforcer:your runtime's executor that applies Decisions elsewhere
//
// Registration, evaluation, and enforcement are provided in other files
// (e.g., registry.go). This file contains the core interfaces and types.
package ccxpolicy

import "errors"

// Scope indicates where a Decision should be applied within the host runtime's
// execution tree. The actual meaning of Node/Subtree/Root is defined by the
// host (e.g., a workflow or context hierarchy).
type Scope int

const (
	// ScopeNode applies to the single target node only.
	ScopeNode Scope = iota
	// ScopeSubtree applies to the target node and all of its descendants.
	ScopeSubtree
	// ScopeRoot applies to the root of the tree that contains the target node.
	ScopeRoot
)

// Action represents the operation to perform when a policy rule triggers.
// The host runtime decides how to realize these actions (typically via an
// Enforcer): e.g., adjust parameters, cancel work, or just warn/log.
type Action int

const (
	// ActionNoop produces no effect (useful for dry-runs or placeholders).
	ActionNoop Action = iota
	// ActionWarn records an advisory signal (e.g., for logs/metrics).
	ActionWarn
	// ActionAdjust mutates the node's parameter map using the provided Adjust fn.
	ActionAdjust
	// ActionCancelNode cancels/aborts only the target node.
	ActionCancelNode
	// ActionCancelSubtree cancels/aborts the target node and all descendants.
	ActionCancelSubtree
	// ActionCancelRoot cancels/aborts the root of the target's tree.
	ActionCancelRoot
)

// Decision is the unit result emitted by a Policy's Check. A policy may return
// zero or more Decisions. The host is responsible for applying them deterministically.
//
//   - PolicyID: identifies the policy that produced this decision.
//   - Scope:    where to apply the decision (Node/Subtree/Root).
//   - Action:   what to do (Warn/Adjust/Cancel*).
//   - Adjust:   functional update applied to Params when ActionAdjust.
//   - Reason:   operator-friendly message explaining why the decision fired.
//   - Stop:     if true, short-circuit evaluation of lower-priority policies.
type Decision struct {
	PolicyID string
	Scope    Scope
	Action   Action
	Adjust   func(params map[string]any) // used only with ActionAdjust
	Reason   error                       // explanatory message for operators
	Stop     bool                        // short-circuit further policy evaluation
}

// Node describes the read-only view of a runtime element that policies inspect.
// Hosts adapt their internal node/task/context objects to this interface.
// Params should represent the current effective parameter map for the node.
type Node interface {
	// ID returns a stable identifier of the node in the host runtime.
	ID() string
	// Name returns a semantic label (e.g., intent name/type).
	Name() string
	// Params returns the node's current parameters (may be a shallow copy).
	Params() map[string]any
	// Parent returns the logical parent node, or nil if this is the root.
	Parent() Node
	// Root returns the root ancestor for this node.
	Root() Node
}

// Policy defines a policy object that can match nodes and emit Decisions.
// Policies run in priority order (ascending Priority). Implementations should
// make Match cheap (fast prefilter) and put heavier logic in Check.
type Policy interface {
	// ID returns a unique identifier for diagnostics and auditing.
	ID() string
	// Priority controls evaluation order; lower values run earlier.
	Priority() int
	// Match quickly determines whether this policy applies to the node.
	Match(n Node) bool
	// Check examines the node and returns zero or more Decisions.
	Check(n Node) []Decision
}

// Reason constructs a simple error value for use as Decision.Reason.
// It is a convenience helper to avoid importing errors at call sites.
func Reason(msg string) error { return errors.New(msg) }
