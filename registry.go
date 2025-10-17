// Package ccxpolicy provides registration, evaluation, and enforcement helpers
// for the minimal, domain-neutral policy engine. This file implements the
// process-level policy registry, the evaluation routine that emits Decisions,
// and the Enforcer interface used to apply those Decisions in a host runtime.
package ccxpolicy

import (
	"sort"
	"sync"
)

// registry holds process-wide policy instances in deterministic priority order.
// It is safe for concurrent reads after initialization. Registration is
// typically performed at process startup (e.g., in init()).
var registry = struct {
	mu       sync.RWMutex
	policies []Policy
}{}

// RegisterPolicy adds a policy to the global registry.
//
// Notes:
//   - Registration order does not matter; policies are kept sorted by
//     Policy.Priority() (ascending) to ensure deterministic evaluation.
//   - Call this at process startup (e.g., in init()). If you hot-reload,
//     coordinate external synchronization to avoid racing with Evaluate.
func RegisterPolicy(p Policy) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.policies = append(registry.policies, p)
	sort.Slice(registry.policies, func(i, j int) bool {
		return registry.policies[i].Priority() < registry.policies[j].Priority()
	})
}

// Evaluate runs all registered policies that Match(n) in ascending Priority and
// returns the emitted Decisions in the order they should be enforced.
//
// Behavior:
//   - For each matching policy, all Decisions returned by Check(n) are appended.
//   - If any Decision has Stop == true, evaluation short-circuits immediately
//     and returns the decisions collected so far.
//   - Evaluate itself is read-only and does not mutate the node.
func Evaluate(n Node) []Decision {
	registry.mu.RLock()
	pols := append([]Policy(nil), registry.policies...) // snapshot under lock
	registry.mu.RUnlock()

	out := make([]Decision, 0, 4)
	for _, p := range pols {
		if !p.Match(n) {
			continue
		}
		ds := p.Check(n)
		for _, d := range ds {
			out = append(out, d)
			if d.Stop {
				return out
			}
		}
	}
	return out
}

// Enforcer is implemented by the host runtime to *apply* Decisions produced by
// Evaluate. The engine is runtime-agnostic: it does not know how to cancel or
// adjust anything—your Enforcer provides those effects.
type Enforcer interface {
	// Adjust applies a parameter mutation function at the specified Scope.
	Adjust(scope Scope, fn func(map[string]any))
	// Cancel aborts work at the specified Scope with a reason suitable for logs.
	Cancel(scope Scope, reason error)
	// Warn records an advisory signal for observability (logs/metrics/tracing).
	Warn(policyID string, reason error)
}

// Enforce applies the provided Decisions against the given Enforcer,
// deterministically and in order.
//
// Mapping of Actions:
//   - ActionNoop:         no effect
//   - ActionWarn:         e.Warn(policyID, reason)
//   - ActionAdjust:       e.Adjust(scope, adjustFn)  (no-op if adjustFn is nil)
//   - ActionCancelNode:   e.Cancel(ScopeNode, reason)
//   - ActionCancelSubtree:e.Cancel(ScopeSubtree, reason)
//   - ActionCancelRoot:   e.Cancel(ScopeRoot, reason)
//
// Short-circuiting:
//   - If a Decision has Stop == true, Enforce stops after applying it.
func Enforce(e Enforcer, ds []Decision) {
	for _, d := range ds {
		switch d.Action {
		case ActionNoop:
			// no-op
		case ActionWarn:
			e.Warn(d.PolicyID, d.Reason)
		case ActionAdjust:
			if d.Adjust != nil {
				e.Adjust(d.Scope, d.Adjust)
			}
		case ActionCancelNode:
			e.Cancel(ScopeNode, d.Reason)
		case ActionCancelSubtree:
			e.Cancel(ScopeSubtree, d.Reason)
		case ActionCancelRoot:
			e.Cancel(ScopeRoot, d.Reason)
		}
		if d.Stop {
			return
		}
	}
}
