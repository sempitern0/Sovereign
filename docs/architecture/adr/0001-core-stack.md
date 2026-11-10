# ADR-0001 — Go CLI, versioned manifests and independent runtime

Status: accepted for H0 foundation (2026-11-10).

## Context
The plan requires Windows/Linux device orchestration and guided Android onboarding, without always-on NAS services or a runtime prerequisite for end users.

## Decision
Go CLI with no daemon; YAML catalogs and profiles validated by JSON Schema; Python for development-time contract checking only. Keep system package managers as separate future adapters. Android is guided, not silent. Multisuite is an optional connector.

## Consequences
H0 is a development scaffold. H1 must provide deterministic manifest parsing/resolution; H2 adds user-consented, verified package operations. End-user release ZIPs will eventually include prebuilt binaries.
