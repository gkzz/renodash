# Spike observations: 2026-11-15

This section records the original Python-based Spike.
Its results do not establish that the replacement Go runner or Ubuntu image passes the same checks.
See the Go migration section below for current validation.

The existing adapters can support local investigation without changing the renodash CLI.
The Node.js process contracts and headless Dev Container isolation probes passed on Linux amd64 under WSL2.
The Skill remains an example. Model behavior and agent-specific policy enforcement have not been verified.
See [the guide](README.md) for commands, contracts, permission proposals, costs, and remaining boundaries.

## Investigation

The initial checkout was clean on `main` at `4b62d362661beb2ee551890288f25d0060d0f080`.
No branch was switched, rewritten, or merged. Existing Spike branches were preserved.
Two unrelated worktree records pointed to absent temporary directories; neither record was changed.

| Existing artifact | Finding |
| --- | --- |
| `internal/command/run.go` and GitHub client | `parse`, `get`, and `version`; no advisory CLI integration |
| `examples/skills/renodash-dashboard/SKILL.md` | Existing frontmatter format, provenance guidance, and untrusted-input boundaries; dedicated vulnerability investigation excluded |
| `.agents/skills/` | Documentation and TDD guidance; no persistent agent permission policy |
| `internal/summary/render.go`, `internal/tools/fake-actions-summary`, `renodash.yml` | Separate offline Summary renderer and workflow; no change required for this Spike |
| `spike/vulnerability-evidence-adapter` at `2654fc92c0bb4c900b1d23dd5ae6af3caf4b6870` | Separate module; saved Node.js sources, SemVer library, JSON stdin/stdout, aggregation, provenance, fixtures, and tests |
| `spike/unseeded-advisory-discovery` at `c27bcd21baf8196d2eabeddfa28427763048f415` | Separate module; official Go scan API, injectable Runner, bounded evidence capture, failure handling, and synthetic tests |
| `spike/update-target-vulnerability` at `2d8c47858ffd5f082c63460263bd5b5ba15fe1c0` | No adapter directory at its tip; reuse the two other branches |
| `Makefile`, mise configuration, test/lint workflows | `make check`, Go tests and vet, English/Japanese textlint; nested adapter modules require explicit commands |

[Issue #80](https://github.com/gkzz/renodash/issues/80) returned an open candidate-list issue with zero comments.
The supplied comment ID `6888490117` returned HTTP 404.
The user's supplied instructions define the publication boundary for this experiment.
No issue or remote setting was modified.

The Node.js adapter needed no code changes: its existing process interface met the local execution requirements.
The Go Runner mock tests also passed without code changes.
The Go live executable writes a new evidence directory and contacts its database; this experiment did not invoke that executable.
No shared adapter interface, live OSV client, or new vulnerability matcher was added.
Existing Markdown and Summary modes remain in the exported source but are outside the Skill workflow.
The existing main documentation's broader reporting proposals were not rewritten as part of this isolated experiment.

## Original Python Spike files (historical)

| Path relative to repository root | Purpose |
| --- | --- |
| `examples/skills/renodash-local-advisory/SKILL.md` | Experimental input routing, state reporting, provenance, and action boundaries |
| `spikes/local-skill/prepare.py` | Export pinned original sources and compile a runtime bundle |
| `spikes/local-skill/contracts.py` | Synthetic process contracts using the original adapter |
| `spikes/local-skill/probe.py` | In-container file, process, network, and normal-execution probes |
| `spikes/local-skill/isolate.py` | Headless Dev Container startup, effective mount audit, host canary, and cleanup |
| `spikes/local-skill/validate.py` | Upstream Skill validation plus resource and command checks |
| `spikes/local-skill/.devcontainer/devcontainer.json` | Read-only, nonroot, offline execution configuration |
| `spikes/local-skill/.devcontainer/Dockerfile` | Pinned base image, nonroot account, and CLI setup markers |
| `spikes/local-skill/evals.json` | Eight model evaluation cases and trace expectations |
| `spikes/local-skill/README.md`, `RESULTS.md` | Reproduction, design decisions, and observed limits |

## Original Python Spike checks (historical)

The final bundle was `/tmp/renodash-local-skill-final`.
Earlier preparation attempts used separate directories and were not overwritten.

| Command or check | Observed result |
| --- | --- |
| `python3 spikes/local-skill/prepare.py /tmp/renodash-local-skill-final` | Passed; compiled the original Node.js adapter |
| `python3 spikes/local-skill/validate.py /tmp/renodash-local-skill-final` | Passed upstream YAML/name validation, resource checks, and three process test methods |
| Five state pairs | Passed Affected/NoMatch, Affected/Affected, NoMatch/Affected, Unknown/Unknown, and Affected/Unknown |
| Four failure conditions | Retrieval failure, unsupported product, empty payload, and no sources all remained Unknown |
| Hostile source text | Preserved as data by adapter transport; model injection resistance remains unverified |
| Original Node.js module: `GOCACHE=/tmp/renodash-spike-go-cache go test -buildvcs=false ./...` | Passed, including saved-fixture tests; run from the v2 exported bundle |
| Original Go module: same test command | Passed, including Runner mocks and saved SDK fixture tests; run from the v2 exported bundle |
| `python3 spikes/local-skill/isolate.py /tmp/renodash-local-skill-final` | Passed actual Dev Container startup, Docker inspection, all isolation probes, and all process contracts |
| `make check` | Passed with permission to run outside the session sandbox; root Go tests, formatting, vet, and documentation lint |
| `git diff --check` and Python AST parsing | Passed |
| Explicit textlint for Spike README and results | Passed |

Tools observed: Go 1.27.3, Docker 28.1.1, Dev Container CLI 0.88.0, and Codex 0.163.0.
The validator source was pinned to `83021e2be457a9b6679148719c1f6074f0ad975e`.
Python base image digest: `sha256:e48f59cdba69a3cbd9ed2cf0f348d90d45a8d350d98134cf1070bc106ee9ed90`.
No paid API or model evaluation was executed.
The new Python scripts are outside `make check`; they were executed separately.
The Spike Markdown files are outside the default lint glob and need explicit textlint checks.

Initial environment failures were distinct from behavior failures:

- GitHub and validator downloads required approved network access; the unavailable issue comment remained a 404.
- Go cache writes required a writable temporary cache; exported sources also required `-buildvcs=false`.
- A harness check against `/bin/true` rejected empty output. This was not a TDD Red result or an adapter failure.
- The first Dev Container startup attempted a forbidden `/etc/environment` write. Image setup markers resolved it without relaxing runtime isolation.
- A retry found the earlier failed container; the launcher now cleans containers returned in startup errors.
- Root cancellation tests could not bind loopback inside the session sandbox; the approved `make check` rerun passed.
- Textlint findings were corrected. The validator's temporary tool directory was moved under `/tmp`.

## Decision and limits

All eight requested success areas have evidence or an explicit design, with these qualifications:

- The Node.js mock runs in the actual Dev Container. Go mock tests ran on the host; Go execution inside the container remains unverified.
- Read-only files, absent host shares, zero capabilities, no elevation, and blocked transport were observed.
- GitHub writes were prevented at the transport layer. Agent deny rules and real token authorization were not exercised.
- Metadata validation ran. Codex/Claude Skill loading and compatibility are still unverified.
- Evals have concrete inputs, semantic criteria, and trace requirements; no model scores exist yet.
  The revised plan starts with three runs, then expands to eight cases with three repetitions after the initial gate passes.
- Public CI suitability is documented, but no workflow was added and no result was uploaded.

The current executor needs no GitHub or model credentials.
Do not move an online agent into this offline image and assume the same network guarantees remain.
A future model controller needs separate credentials and a narrow adapter tool boundary.
No container escape, editor integration, external extension, or alternate-launcher security claim is established.

Proceed with a separate local adapter module and an optional isolated executor.
Keep real investigation output local and Public CI limited to synthetic contracts and static checks.
Use the script as the baseline, then decide whether Skill routing and reporting improve it through model Evals.
Dev Container should remain an optional execution environment for renodash.

## Review follow-up

The review changed the evaluation plan, not the observed isolation results.
Start with `route`, `merge`, and `injection`, once each.
The eight-case, three-repetition plan remains an expansion stage.
See the [initial evaluation gate](README.md#initial-evaluation-gate) for prerequisites and separate boundary and model verdicts.

The controller boundary remains unimplemented and unverified.
No agent run, agent-specific allow/deny check, or model evaluation was performed during this revision.
No Public CI workflow was added.
Compare the Skill with direct script output before deciding whether to adopt it.

## Go migration and Ubuntu runtime

The five Python scripts were replaced with Go code using only the standard library.
The renodash CLI and the original advisory adapters were not changed.
Python and uv are needed only when running the optional upstream metadata validator.
The Go resource check does not claim metadata validation.

| Current file | Responsibility |
| --- | --- |
| `main.go` | Bundle preparation, command dispatch, and local resource checks |
| `contracts.go` | Synthetic inputs and original adapter process contracts |
| `isolate.go` | Dev Container startup, effective configuration audit, cleanup, and runtime probes |
| `main_test.go` | Offline runner tests, unsafe configuration cases, and optional original adapter integration |

The runtime image is Ubuntu 24.04 at `sha256:2198e4c65076e9cb0b3d695219df667c0e3b0072da64ca3c2648aca17dcecaf5`.
Both binaries are compiled on the host with `CGO_ENABLED=0` for Linux amd64.
The container needs neither Python nor the Go compiler.
Checks return errors directly; they cannot be disabled by an interpreter optimization flag.

| Executed check | Result |
| --- | --- |
| Go runner tests with `RENODASH_ADVISORY_BINARY=/tmp/renodash-local-skill-final/runtime/bin/advisory` | Passed against the original adapter and synthetic process fixture |
| `go run ./spikes/local-skill prepare /tmp/renodash-local-skill-go-v2` | Passed; exported pinned sources and built both binaries |
| `go run ./spikes/local-skill validate /tmp/renodash-local-skill-go-v2` | Passed resource checks and original adapter contracts |
| `go run ./spikes/local-skill isolate /tmp/renodash-local-skill-go-v2` | Passed on Ubuntu, including configuration inspection, host canary, runtime probes, and contracts |
| `make check` | Passed formatting, vet, root Go tests including the Spike runner, and documentation lint |
| Explicit textlint for Spike README and results | Passed |
| `git diff --check` | Passed |

The first Go process-fixture test failed because the test executable rejected the adapter's `-format` argument.
Handling the synthetic child process before test flag parsing fixed the test harness.
This was not an adapter behavior failure or a test-first Red result.
The migration preserves the existing contracts; no deliberate defect was introduced for a Red result.

The standard Go tests use a synthetic child process and do not require Docker or exported adapter sources.
The original adapter integration test skips unless `RENODASH_ADVISORY_BINARY` is supplied.
Bundle validation and container probes ran that original adapter explicitly.
The optional metadata validator was not rerun during this migration.
Model Evals and agent policy enforcement remain unverified.
