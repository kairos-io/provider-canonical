# CLAUDE.md — provider-canonical

## What This Repo Is

`provider-canonical` is a Kairos cluster plugin that implements the Canonical Kubernetes (snap-based) provider. It consumes the `kairos-sdk/clusterplugin` interface and produces `yip` stage configs that are executed at `boot.before` to install, bootstrap, join, and upgrade Canonical K8s nodes. The provider delegates all actual snap installation and `k8s` CLI invocations to shell scripts in `scripts/`; the Go code is purely responsible for generating the correct yip stage definitions.

## Package Layout

```
main.go                          — wires ClusterPlugin, registers event handlers
pkg/provider/provider.go         — ClusterProvider func, CreateClusterContext, stage dispatch
pkg/provider/reset.go            — HandleClusterReset event handler
pkg/domain/cluster_context.go   — ClusterContext struct (the sole shared state)
pkg/domain/constants.go          — path constants only (no logic)
pkg/stages/pre.go                — GetPreSetupStages: proxy, pre-setup, local images
pkg/stages/init.go               — GetInitStage: bootstrap config + bootstrap + upgrade
pkg/stages/join.go               — GetControlPlaneJoinStage, GetWorkerJoinStage
pkg/stages/reconfigure.go        — args file regeneration + cert regeneration + service restarts
pkg/stages/upgrade.go            — getUpgradeStage (one-liner wrapper)
pkg/stages/proxy.go              — proxy env files for kubelet and containerd
pkg/utils/certs.go               — x509 cert generation, signing, loading
pkg/utils/files.go               — FileExists, DirExists (vfs.FS wrappers)
pkg/utils/proxy.go               — no-proxy assembly, CIDR helpers
pkg/utils/stages.go              — GetFileStage helper
pkg/fs/fs.go + init.go           — package-level OSFS var, set to vfs.OSFS in init()
pkg/log/log.go                   — logrus setup with lumberjack rotation
pkg/version/version.go           — Version string set via ldflags
scripts/common.sh                — snap install helpers, retry loop, wait-for-ready
scripts/bootstrap.sh             — installs snaps, runs k8s bootstrap, holds refresh
scripts/join.sh                  — installs snaps, runs k8s join-cluster
scripts/upgrade.sh               — configmap lock, installs snaps, waits ready
scripts/pre-setup.sh             — enables/restarts snapd
scripts/reset.sh                 — removes k8s snap, purges state dirs
scripts/import-images.sh         — loads local OCI images
```

## Provider Implementation Patterns

- `ClusterProvider` is a pure function: `func ClusterProvider(cluster clusterplugin.Cluster) yip.YipConfig`. It takes the SDK cluster value and returns a complete yip config. No side effects, no global state mutation beyond the `ClusterContext` it builds internally.
- `CreateClusterContext` converts `clusterplugin.Cluster` into `*domain.ClusterContext`. All defaults are set here (`CustomAdvertiseAddress = "''"` when absent, `LocalImagesPath = domain.DefaultLocalImagesDir` when empty).
- Stage dispatch lives in `getFinalStages`: check `clusterCtx.NodeRole` against `clusterplugin.RoleInit`, `RoleControlPlane`, `RoleWorker` using `if/else if` chains, not switch statements.
- Every exported stage-getter returns `[]yip.Stage`. Every unexported stage-getter returns either `yip.Stage` (single stage) or `*[]yip.Stage` when the result is optional (nil means "skip this stage entirely").
- Stages are appended in order: pre-setup first, then the role-specific stage, then upgrade, then optional reconfigure, then optional cert regeneration. Never reorder this sequence.
- The yip stage `If` field is used for idempotency guards: `fmt.Sprintf("[ ! -f %s ]", "/opt/canonical/canonical.bootstrap")`. The sentinel files are created by the shell scripts, not by Go.
- Config parsing uses `yaml.Unmarshal` with `_ =` — errors are silently discarded. This is deliberate: a missing or empty config produces a zero-value `apiv1.BootstrapConfig` and the code proceeds with defaults.

## Canonical-Specific Patterns

**Snap services**: All systemd service names follow the snap naming convention `snap.k8s.<component>.service`. When restarting services, always include `systemctl daemon-reload` as the first command. The full set for a control plane restart is: containerd, kube-apiserver, kube-controller-manager, kube-scheduler, kube-proxy, kubelet. Workers only restart containerd, kube-proxy, kubelet.

**Args files**: Canonical K8s stores kube component args in flat `key=value` files at `/var/snap/k8s/common/args/<component>`. The constant is `domain.KubeComponentsArgsPath`. Merging args uses `maps.Copy(currentArgs, updatedArgs)` — incoming args overwrite existing ones. Each arg becomes `key=value` joined by `\n`.

**Bootstrap config type**: Use `apiv1.BootstrapConfig` from `github.com/canonical/k8s-snap-api/api/v1` for init nodes, `apiv1.ControlPlaneJoinConfig` for control plane joins, `apiv1.WorkerJoinConfig` for worker joins. Always unmarshal from `clusterCtx.UserOptions`. Always marshal back to YAML to write the config file stage.

**dqlite HA**: HA join uses the same `GetControlPlaneJoinStage` path as non-HA control planes. The token, advertise address, and node role are passed as positional arguments to `join.sh`. The shell script calls `k8s join-cluster $token --file /opt/canonical/join-config.yaml [--address $addr]`. No Go-level dqlite awareness is needed.

**Local images**: Check `utils.DirExists(fs.OSFS, clusterCtx.LocalImagesPath)` before appending the import-images stage. Never assume the directory exists.

**Reconfigure guard**: Check `utils.DirExists(fs.OSFS, domain.KubeComponentsArgsPath)` before appending reconfigure stages. Args files only exist after a node has been bootstrapped/joined at least once.

**Cert regeneration guard**: `getApiserverCertRegenerateStage` returns `nil` if SANs are empty, if the cert file does not exist yet, or if all incoming SANs are already present. The caller pattern is:
```go
if certStage := getApiserverCertRegenerateStage(sans); certStage != nil {
    stages = append(stages, *certStage...)
}
```

## Error Handling Rules

- In stage-building code that cannot propagate errors (functions that must return `yip.Stage`), use `logrus.Fatalf` on unrecoverable errors. This is the pattern throughout `reconfigure.go`. Do not wrap these in returned errors.
- In event handlers (`HandleClusterReset`), set `response.Error` as a formatted string and return early. Do not panic. Pattern:
  ```go
  if err := json.Unmarshal(...); err != nil {
      logrus.Error("failed to parse reset event: ", err.Error())
      response.Error = fmt.Sprintf("failed to parse reset event: %s", err.Error())
      return response
  }
  ```
- In utility functions that return `(T, error)`, wrap errors with `errors.Wrap` (from `github.com/pkg/errors`) when adding context to an underlying error from a library. Use `fmt.Errorf("...: %w", err)` for errors generated in this codebase.
- YAML unmarshal errors on user-supplied config are discarded with `_ =`. Do not add error handling here.
- File read errors in `getRootCaAndKey` are returned as plain errors, not wrapped.

## Code Style Rules

- Functions are short. If a function body exceeds ~20 lines it should be split. The pattern is many small, single-purpose functions each returning one stage or one value.
- No named return values anywhere in the codebase.
- No pointer receivers. All methods use value receivers (the only method is `CanonicalLogger.Format`).
- Structs use both `json` and `yaml` struct tags side by side.
- Boolean flags are declared as `bool` variables before use, not inlined: `enableDns := true` then `canonicalConfig.ClusterConfig.DNS.Enabled = &enableDns`.
- `var stages []yip.Stage` — always declare the accumulator slice as nil (no `make`), then `append` to it. Never pre-allocate with a known capacity.
- Package-level `const` blocks use grouping for related path constants. No `iota`. No typed constants.
- Import aliasing: `apiv1 "github.com/canonical/k8s-snap-api/api/v1"` and `yip "github.com/mudler/yip/pkg/schema"` are always aliased. Do not change these aliases.
- Inline constants for small string values that appear once in proxy.go (`envPrefix = "Environment="`). Do not define constants for values used in only one expression.
- `filepath.Join` for all path construction. Never string concatenation for paths.
- `fmt.Sprintf` for shell command construction. Commands are always `[]string` in the yip stage `Commands` field.

## Testing Conventions

- Test framework: `github.com/onsi/gomega` only. Import with the dot: `. "github.com/onsi/gomega"`. No testify. No assertions from the standard library.
- Instantiation pattern: `g := NewWithT(t)` at the top of the test function, then use `g.Expect(...)` for all assertions. Sub-tests do not create their own `g`.
- Sub-tests use `t.Run("description", func(t *testing.T) { ... })` with lowercase, descriptive strings that read as sentences ("appends element when not present", "not append when element already present").
- Virtual filesystem for tests that touch the real FS: use `vfst.NewTestFS(map[string]interface{}{...})` from `github.com/twpayne/go-vfs/v4/vfst`. Swap the global `fs.OSFS` and restore with `defer`:
  ```go
  originalFS := fs.OSFS
  fs.OSFS = testFS
  defer func() { fs.OSFS = originalFS }()
  ```
- Tests live in the same package as the code they test (`package stages`, `package provider`, `package utils`). There are no `_test` package suffixes.
- Test only exported functions in `provider_test.go`. Test unexported functions directly in the same package (e.g., `readServiceArgsFile`, `getArgs`, `containsAnyNonMatch`).
- Inline fixture data as `var` block strings at the top of the test file (PEM certs, key files).
- Do not mock interfaces. Swap the `fs.OSFS` global directly.

## Patterns to Avoid

- Do not add error return values to stage-builder functions. They return `yip.Stage` or `[]yip.Stage`, period.
- Do not use `switch` for node role dispatch. The codebase uses `if/else if`.
- Do not introduce new packages. All new functionality belongs in an existing package (`stages`, `utils`, `domain`, `provider`).
- Do not add abstraction layers (interfaces, registries, factories). The provider is a direct pipeline: cluster input -> context -> stages -> yip config.
- Do not use `context.Context`. No timeouts or cancellation in Go code; retry logic lives in bash scripts.
- Do not add structured logging fields to individual log calls. Use `logrus.Info/Error/Fatal` with plain string messages.
- Do not write to the filesystem from Go. All filesystem writes happen via yip `Files` entries or shell scripts.
- Do not shell out from Go (except `exec.Command` in `reset.go` which handles a live event, not stage generation).
- Do not use `make([]string, 0)` — use `var x []string`.
- Do not use `interface{}` — use `any` (Go 1.18+).
- Do not add YAML/JSON struct tags to types in the `stages` or `utils` packages; tags belong only on `ClusterContext`.

## Function Design & Testability

- **Every function does one thing and fits in ~20–30 lines.** If it grows beyond that, extract named helpers.
- **Write functions so they can be unit tested in isolation** — no hidden side effects, no global state access, no I/O buried inside business logic.
- **Most business logic must be unit testable** without spinning up a server, database, or Kubernetes cluster. Separate I/O at the boundary.
- **Use guard clauses / early returns** to reduce nesting. Flat code is easier to read and test than deeply nested.
- **Accept interfaces, return concrete types.** This makes callers mockable without reflection or code generation.
- **Keep interfaces small** — 1–3 methods. Large interfaces are hard to mock and signal poor separation of concerns.

## General Go Practices

- **Dependency injection over globals.** Pass dependencies via constructors or function parameters — not package-level singletons (except logging).
- **`context.Context` is always the first parameter** on any function that performs I/O. Never store it in a struct field.
- **Table-driven tests** for any function with multiple input/output cases: `[]struct{ name, input, expected }` with `t.Run`.
- **Test naming:** `TestFuncName_Scenario` — e.g. `TestCreateCluster_MissingName`.
- **Prefer `switch` over long `if/else if` chains.**
- **Short variable names in small scopes** (`i`, `v`, `err`) are idiomatic; use descriptive names in wider scopes.
- **No goroutines unless concurrency is genuinely required.** Sequential code is easier to test and reason about.
- **Avoid `init()` for anything except registering handlers or loggers.** Never use it for config loading or side-effectful setup.
- **Respect context cancellation** in any loop that calls external services.
- **Import grouping:** stdlib / external / internal — separated by blank lines, sorted by `goimports`.
- **Don't over-abstract.** Don't create an interface or wrapper until there are ≥2 concrete implementations or a clear testing need.
- **No naked `panic` in library code.** Panics are only acceptable in `main` or test setup for truly unrecoverable state.
