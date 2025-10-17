# ccxpolicy — Minimal, domain-neutral policy engine (stdlib-only)

`ccxpolicy` is a tiny, dependency-free module for declaring and running **threshold/governance policies** over arbitrary “nodes” in your runtime. It is **not** tied to any orchestration library. Consumers (e.g., `ccx`) adapt:

* a **Node** view (read-only name/params/lineage),
* **Decisions** (adjust/cancel/warn),
* and an **Enforcer** that applies the decisions in the host runtime.

Use it when you want declarative rules like:

> “If `quality > 1080`, **adjust** it to `1080`; if `safety.block` is true, **cancel root**.”

---

## Install

```bash
go get github.com/yourorg/ccxpolicy@latest
```

---

## Core Concepts

### Node (read-only view)

Your runtime adapts its unit of work to this interface.

```go
type Node interface {
    ID() string
    Name() string
    Params() map[string]any
    Parent() Node   // may be nil
    Root() Node
}
```

### Policy

A small object that can (a) decide applicability and (b) emit **Decisions**.

```go
type Policy interface {
    ID() string
    Priority() int        // lower runs earlier
    Match(n Node) bool    // quick applicability probe
    Check(n Node) []Decision
}
```

### Scope & Action

Where to apply, and what to do.

```go
type Scope int
const (
    ScopeNode Scope = iota
    ScopeSubtree
    ScopeRoot
)

type Action int
const (
    ActionNoop Action = iota
    ActionWarn
    ActionAdjust
    ActionCancelNode
    ActionCancelSubtree
    ActionCancelRoot
)
```

### Decision

The output of a policy check.

```go
type Decision struct {
    PolicyID string
    Scope    Scope
    Action   Action
    Adjust   func(params map[string]any) // used when ActionAdjust
    Reason   error
    Stop     bool // short-circuit evaluation when true
}
```

---

## Quick Start

### 1) Implement a Policy

```go
package main

import (
    policy "github.com/yourorg/ccxpolicy"
)

type QualityCap struct{}

func (QualityCap) ID() string               { return "cap_quality" }
func (QualityCap) Priority() int            { return 10 }
func (QualityCap) Match(n policy.Node) bool { return n.Name() == "Transcode" }

func (QualityCap) Check(n policy.Node) []policy.Decision {
    q, _ := n.Params()["transcode.targetQuality"].(int)
    if q > 1080 {
        return []policy.Decision{{
            PolicyID: "cap_quality",
            Scope:    policy.ScopeSubtree,
            Action:   policy.ActionAdjust,
            Adjust:   func(p map[string]any){ p["transcode.targetQuality"] = 1080 },
            Reason:   policy.Reason("quality above cap"),
        }}
    }
    return nil
}
```

### 2) Register at startup

```go
func init() {
    policy.RegisterPolicy(QualityCap{})
}
```

### 3) Evaluate + Enforce at runtime

You provide the **Node** (your adapter) and the **Enforcer** (how to apply).

```go
// Evaluate runs all matching policies in priority order and returns decisions.
ds := policy.Evaluate(myNode)

// Enforcer is your bridge to the host runtime.
type myEnforcer struct{ /* refs to your runtime */ }

func (e myEnforcer) Adjust(s policy.Scope, fn func(map[string]any)) { /* mutate params */ }
func (e myEnforcer) Cancel(s policy.Scope, reason error)            { /* cancel node/subtree/root */ }
func (e myEnforcer) Warn(id string, reason error)                   { /* log/metrics */ }

// Apply decisions deterministically.
policy.Enforce(myEnforcer{/* ... */}, ds)
```

---

## Adapting to Your Runtime

Most users pair `ccxpolicy` with an orchestration layer (e.g., `ccx`).
In that case, the orchestration library typically provides:

* a **Node adapter** (turns its node into `ccxpolicy.Node`), and
* an **Enforcer** that maps decisions to its own primitives (adjust/cancel/warn).

If you don’t use an orchestration library, you can still:

* implement `Node` over your own task/graph model, and
* write a tiny `Enforcer` to mutate state and cancel work.

---

## Determinism & Ordering

* Policies run in **ascending Priority**.
* A `Decision` with `Stop: true` **short-circuits** further evaluation.
* Multiple `ActionAdjust` decisions apply in order; last writer wins.

---

## Thread-Safety

* `Evaluate` does **no mutation** and may be run concurrently.
* `Enforce` calls your `Enforcer`; make it thread-safe if your runtime is concurrent.
* If `Adjust` is used, ensure your parameter store is protected (mutex/CAS) in your runtime.

---

## Minimal JSON Example (build your own adapter)

This module intentionally **does not** include JSON parsing—keep it in your app, or build a small adapter that turns declarative rules into `Policy` implementations. A simple schema:

```jsonc
{
  "policies": [
    {
      "id": "safety_stop",
      "priority": 5,
      "match": { "intent": "*" },
      "rules": [
        {
          "path": "safety.block",
          "op": "==",
          "value": true,
          "on_violation": { "action": "cancel_root", "reason": "Safety override", "stop": true }
        }
      ]
    }
  ]
}
```

Write a loader that:

* matches on `intent` (or other fields),
* reads `params[path]`,
* compares with `op`,
* and emits `Decision{Scope, Action, Adjust, Reason, Stop}`.

---

## FAQ

**Q: Does `ccxpolicy` know how to cancel or adjust tasks?**
*A:* No. It only **decides**. Your **Enforcer** applies those decisions in your runtime.

**Q: Is there a default logger or metrics?**
*A:* No. Use `Enforcer.Warn` to hook into your logging/metrics.

**Q: Can I hot-reload policies?**
*A:* Yes. Build your own loader to re-`RegisterPolicy` (you control lifecycle).
Note: `RegisterPolicy` appends to a process-global list; design your reload accordingly.

---

## API Reference (selected)

```go
// Registration and evaluation
func RegisterPolicy(p Policy)
func Evaluate(n Node) []Decision

// Enforcement
type Enforcer interface {
    Adjust(scope Scope, fn func(map[string]any))
    Cancel(scope Scope, reason error)
    Warn(policyID string, reason error)
}
func Enforce(e Enforcer, ds []Decision)

// Helpers
func Reason(msg string) error
```

---

## Files Structure

```text
ccxpolicy/
├─ go.mod
├─ README.md
├─ policy.go
└─ registry.go
```

---

## License

Apache-2.0 (or your preference).
