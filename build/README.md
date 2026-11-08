# Build Directory

The build directory contains the Wails v3 project configuration, platform build tasks, and assets.

Frontend tooling requires Node 20.19+, Node 22.12+, or Node 24+ (see the
`engines` field in `frontend/package.json`). Install the locked dependencies
with `npm --prefix frontend ci --allow-git=root`.

The structure is:

* bin - Build output directory
* config.yml - Wails v3 product and development configuration
* Taskfile.yml - Shared frontend build and asset-generation tasks
* darwin, linux, windows - Platform build tasks and assets

Edit product metadata in `config.yml`, then run
`wails task common:update:build-assets` to update the platform build assets.
Wails preserves custom plist values when updating them.
The same task then applies `info.license` to the Linux nfpm config, since Wails
otherwise generates it with an MIT license. This label follows the project's
license grant; `LICENSE` retains the full license text.

## Compiled Application Metadata

Local platform tasks (including Makefile build/package and development targets)
and CI use `go run ./build/buildflags` to read `config.yml` and inject the product
name, description, version, copyright, and license with Go's `-ldflags -X`. The helper
also stamps the current Git branch (short commit on detached checkouts) and UTC
build time. It fails on missing or unrepresentable metadata rather than building
with placeholders. No Go source files are rewritten and no `jq` or `yq` is needed.

`make build-flags` prints these flags for manual builds. Direct `go build` or
`go test` without the flags leaves those fields, including the license, as `"undef"`.

## CI Artifacts

`.github/workflows/release-artifacts.yml` builds on every push to `main`
(including merges) and supports manual runs from GitHub Actions. It follows
the [Wails native-runner CI guidance](https://wails.io/guides/build/cross-platform/#cicd-integration).

Download artifacts from the completed **Release Artifacts** workflow run:

* `ttyphoon-<version>-macos-arm64` - Apple Silicon `.app` bundle in a ZIP archive
* `ttyphoon-<version>-macos-amd64` - Intel `.app` bundle in a ZIP archive
* `ttyphoon-<version>-linux-amd64` - x86-64 executable in a `.tar.gz` archive

Each artifact contains an archive and a separate SHA-256 checksum file.
Artifacts are retained for 30 days; the workflow does not publish GitHub
Releases. macOS bundles are ad-hoc signed, not Developer ID signed or notarized,
so Gatekeeper may require explicit approval. Linux builds target Ubuntu 24.04
and require GTK4, WebKitGTK 6.0, and X11 runtime libraries on the destination
machine.


## Mac

The `darwin` directory holds files specific to macOS builds. `Info.plist` and
`Info.dev.plist` are updated from `config.yml` by Wails.

The directory contains the following files:

- `Info.plist` - the main generated plist used when packaging the app bundle.
- `Info.dev.plist` - the generated development plist.

## Linux

The `linux` directory holds Linux build tasks and packaging assets.
The desktop entry and nfpm package metadata are updated from `config.yml` by
Wails.

Global hotkeys require CGO and X11 development libraries (`libx11-dev` on
Debian/Ubuntu or `libX11-devel` on Fedora). Registration requires an accessible
X display; native Wayland global shortcuts are not supported by this backend.

## Windows

The `windows` directory contains the manifest and resource files used for
Windows builds. `info.json` and `wails.exe.manifest` are updated from
`config.yml` by Wails.

- `icon.ico` - The icon used for the application. This is used when building using `wails build`. If you wish to
  use a different icon, simply replace this file with your own. If it is missing, a new `icon.ico` file
  will be created using the `appicon.png` file in the build directory.
- `installer/*` - The files used to create the Windows installer.
- `info.json` - Application details used for Windows builds. The data here will be used by the Windows installer,
  as well as the application itself (right click the exe -> properties -> details)
- `wails.exe.manifest` - The main application manifest file.