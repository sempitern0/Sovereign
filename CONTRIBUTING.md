# Contributing

Open focused PRs with acceptance criteria and test results. Start with [docs/STATUS.md](docs/STATUS.md) and [AGENTS.md](AGENTS.md), which says which parts of the technical plan each task needs. Do not run real installations from tests; use simulated providers.

For new applications follow «Adding or verifying an application» in AGENTS.md: official upstream, license, exact package ID and source per platform with date, a matching entry in `catalog/install-allowlist.yaml`, and real-device acceptance recorded in STATUS. Keep unverified packages pending. Update STATUS at the end of every PR.
