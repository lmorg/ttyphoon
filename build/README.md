# Build Directory

The build directory contains the Wails v3 project configuration, platform build tasks, and assets.

The structure is:

* bin - Build output directory
* config.yml - Wails v3 product and development configuration
* Taskfile.yml - Shared frontend build and asset-generation tasks
* darwin, linux, windows - Platform build tasks and assets

## Mac

The `darwin` directory holds files specific to macOS builds.

The directory contains the following files:

- `Info.plist` - the main plist file used when packaging the app bundle.
- `Info.dev.plist` - the development plist file.

## Windows

The `windows` directory contains the manifest and resource files used for Windows builds.

- `icon.ico` - The icon used for the application. This is used when building using `wails build`. If you wish to
  use a different icon, simply replace this file with your own. If it is missing, a new `icon.ico` file
  will be created using the `appicon.png` file in the build directory.
- `installer/*` - The files used to create the Windows installer.
- `info.json` - Application details used for Windows builds. The data here will be used by the Windows installer,
  as well as the application itself (right click the exe -> properties -> details)
- `wails.exe.manifest` - The main application manifest file.