# 0087 - Build versioned release artifacts on native CI runners

Date: 2026-11-04

## Status

Accepted

## Context

Local Linux builds exceeded available memory and were killed with exit status
137. Wails applications require platform-specific CGO dependencies, and release
artifacts need predictable packaging without relying on a developer's machine.

Version values exist in several packaging metadata files. Naming an artifact
from one of those files can disagree with the version compiled into the app.
GitHub-hosted runners also need to install public Git dependencies without SSH
credentials.

## Decision

- Run the Release Artifacts workflow on pushes to `main`, including merges,
  and allow manual runs through `workflow_dispatch`.
- Build natively on `macos-15` for Apple Silicon, `macos-15-intel` for Intel
  macOS, and `ubuntu-24.04` for x86-64 Linux. Do not introduce Docker-based
  cross-compilation or require developer-hosted runners.
- Read the Go version from `go.mod`, install the Wails CLI version selected by
  the Go module, and use Node 24 for frontend commands. The checkout, Go setup,
  Node setup, and artifact upload actions also declare Node 24 runtimes.
- Install Linux's compiler, pkg-config, GTK4, WebKitGTK 6.0, and X11 development
  libraries before generating bindings or compiling CGO code.
- Configure GitHub SSH URLs to resolve through HTTPS. Use `npm ci` with Git
  fetching restricted to dependencies declared directly in the frontend
  manifest (`npm_config_allow_git=root`).
- Generate bindings and icons, install dependencies, run the frontend tests,
  build the frontend, then compile the production executable. Check that the
  frontend entry point and native executable exist before packaging.
- Keep these stages sequential within each job. Limit Task execution to one
  task at a time and Go execution/compilation parallelism to two workers through
  `GOMAXPROCS=2` and `GOFLAGS=-p=2`. Platform jobs may run concurrently, and a
  failed platform job does not cancel its siblings.
- After compilation, run the executable with `--version`. This command prints
  the compiled application version and returns before cache or GUI startup in
  `main`. Reject empty or filename-unsafe output rather than falling back to
  packaging metadata. No `jq` or YAML parsing is needed for artifact naming.
- Name artifacts `ttyphoon-<version>-<platform>-<architecture>`. Package macOS
  `.app` bundles as ZIP archives and the Linux executable as a `.tar.gz` archive;
  include a separate SHA-256 checksum file for each archive.
- Ad-hoc sign and verify macOS bundles. Developer ID signing and notarization
  are not part of this workflow and require a separate credentials decision.
- Upload archives and checksums as GitHub Actions artifacts retained for 30
  days, failing if there are no files to upload. Do not create GitHub Releases,
  publish tags, deploy applications, or grant repository write permissions.
- Serialize workflow runs for the same ref without cancelling an active run.
  Limit each platform job to 45 minutes.

## Consequences

- Merges to `main` produce downloadable, versioned artifacts without requiring
  enough memory to build locally. Resource limits reduce contention but cannot
  guarantee that an individual compiler or bundler will fit available memory.
- Artifact names reflect the executable's reported version, whose source is
  `app/app.go`. This does not synchronize plist, Windows, Linux package, or
  Wails configuration versions; those metadata values remain separately owned.
- macOS artifacts may require explicit Gatekeeper approval. Linux executables
  are not fully standalone: destination systems need compatible GTK4,
  WebKitGTK 6.0, X11, and other dynamically linked runtime libraries.
- This is continuous artifact delivery, not an automated public release or
  deployment system. Downloads expire; durable public releases, Windows/Linux
  ARM builds, and universal macOS binaries are outside the current scope.
- CI runs the frontend suite and native compilation, not the full Go test
  suite. Workflow lint and local macOS packaging checks do not establish that
  every hosted platform job has completed successfully.

## References

- [Release workflow](../.github/workflows/release-artifacts.yml)
- [Build and artifact documentation](../build/README.md)
- [Headless version command](../main.go)
- [Compiled application version](../app/app.go)
- [ADR 0085: Wails v3 migration](0085-wails-v3-migration.md)
- [ADR 0014: verification baseline](0014-test-baseline-and-verification.md)
- [Wails native-runner CI guidance](https://wails.io/guides/build/cross-platform/#cicd-integration)