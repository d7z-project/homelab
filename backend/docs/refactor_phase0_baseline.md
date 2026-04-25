# Refactor Phase 0 Baseline

## Purpose

This document freezes the backend refactor baseline described in [TODO.md](/home/dragon/Documents/IDEA/homelab/TODO.md). It is the working contract for the destructive migration from the current `models + repositories + services + controllers` layout to the new `apis + store + modules + runtime` layout.

## Frozen Scope

### Core modules

- `auth`
- `audit`
- `dns`
- `rules`

### Optional modules

- `workflow`
- `intelligence`

### Out of scope for the first migration wave

- Multi-version API compatibility
- Legacy route compatibility for `/api/v1/network/ip/...` and `/api/v1/network/site/...`
- Compatibility shims for the current frontend or generated clients

## Frozen Resource Set

### Core resources

- `RuleSet`
- `Feed`
- `Export`
- `DNSZone`
- `DNSRecord`
- `Role`
- `RoleBinding`
- `ServiceAccount`

### Optional resources

- `Workflow`
- `TaskRun`

### Mapping from current models

- `IPPool` -> `RuleSet{spec.type=ip}`
- `SiteGroup` -> `RuleSet{spec.type=domain}`
- `IPSyncPolicy` -> `Feed{spec.type=ip}`
- `SiteSyncPolicy` -> `Feed{spec.type=domain}`
- `IPExport` -> `Export{spec.type=ip}`
- `SiteExport` -> `Export{spec.type=domain}`
- `Domain` -> `DNSZone`
- `Record` -> `DNSRecord`

## Frozen Object Contract

### Object shape

All new API resources use:

```json
{
  "apiVersion": "group/v1",
  "kind": "Kind",
  "metadata": {
    "name": "example",
    "uid": "generated-stable-id",
    "labels": {},
    "annotations": {},
    "generation": 1,
    "resourceVersion": "1",
    "creationTimestamp": "2026-04-01T00:00:00Z"
  },
  "spec": {},
  "status": {}
}
```

### List shape

All list responses use:

```json
{
  "apiVersion": "group/v1",
  "kind": "KindList",
  "metadata": {
    "continue": "",
    "resourceVersion": "1"
  },
  "items": []
}
```

### Frozen rules

- Public list pagination uses `metadata.continue`.
- `resourceVersion` is a `string`.
- `generation` increments only when `spec` changes.
- Config resources require `metadata.name`.
- Runtime resources may omit `metadata.name`; the server must generate `metadata.uid` and may generate `metadata.name`.
- Internal references prefer `metadata.uid` when both `name` and `uid` exist.

## Module Boundary Rules

- `rules` must not directly depend on `intelligence`.
- `intelligence` may extend `rules` only through explicit interfaces.
- `workflow` must not be required for DNS or rules CRUD.
- Discovery registration must move out of package `init()` and into module startup.
- Controllers must stop depending on package-level service singletons.

## Current Migration Blockers

The following files represent the main architectural blockers that must be retired during later phases:

- [backend/main.go](/home/dragon/Documents/IDEA/homelab/backend/main.go): hard-coded service construction, startup sequencing, and module lifecycle.
- [backend/route.go](/home/dragon/Documents/IDEA/homelab/backend/route.go): static route assembly around package-level controllers.
- [backend/pkg/controllers/ip_controller.go](/home/dragon/Documents/IDEA/homelab/backend/pkg/controllers/ip_controller.go): package-level service injection.
- [backend/pkg/controllers/site_controller.go](/home/dragon/Documents/IDEA/homelab/backend/pkg/controllers/site_controller.go): package-level service injection.
- [backend/pkg/controllers/intelligence_controller.go](/home/dragon/Documents/IDEA/homelab/backend/pkg/controllers/intelligence_controller.go): package-level service injection.
- [backend/pkg/runtime/registry/registry.go](/home/dragon/Documents/IDEA/homelab/backend/pkg/runtime/registry/registry.go): global discovery/resource registry.
- [backend/pkg/services/ip/service.go](/home/dragon/Documents/IDEA/homelab/backend/pkg/services/ip/service.go): package `init()` discovery registration.
- [backend/pkg/services/site/service.go](/home/dragon/Documents/IDEA/homelab/backend/pkg/services/site/service.go): package `init()` discovery registration.
- [backend/pkg/services/dns/service.go](/home/dragon/Documents/IDEA/homelab/backend/pkg/services/dns/service.go): package `init()` discovery registration.
- [backend/pkg/services/actions/service.go](/home/dragon/Documents/IDEA/homelab/backend/pkg/services/actions/service.go): package `init()` discovery registration.
- [backend/pkg/services/rbac/discovery.go](/home/dragon/Documents/IDEA/homelab/backend/pkg/services/rbac/discovery.go): package `init()` discovery registration.
- [backend/pkg/services/intelligence/service.go](/home/dragon/Documents/IDEA/homelab/backend/pkg/services/intelligence/service.go): package `init()` discovery registration.

## Test Baseline

The refactor safety baseline is:

- Existing unit tests are informative, not compatibility constraints.
- New phases must add tests around:
  - resource-store optimistic concurrency
  - `generation` vs `status` update semantics
  - `continue` pagination boundaries
  - instance-level RBAC denial
  - discovery permission filtering
  - module lifecycle start/stop
- Minimum regression commands after each phase:
  - `go test ./tests/unit/...`
  - `make backend-gen`
  - `make frontend-gen`
  - `make frontend-build`

## Phase 1 Entry Criteria

Phase 1 may start once all new API work targets `backend/pkg/apis/...` instead of adding new old-style `Meta/Status` models under `backend/pkg/models`.
