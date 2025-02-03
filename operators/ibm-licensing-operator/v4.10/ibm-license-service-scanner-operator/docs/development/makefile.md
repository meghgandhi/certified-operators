# Makefile definitions

Main `Makefile` is responsible for some common variable and targets definitions, as well as including more "specialised"
makefiles:

- `build.mk` -> build & push
- `deps.mk` -> dependencies
- `lint.mk` -> linting

Each `Makefile` is thoroughly described below.

## Makefile

Includes targets from the other makefiles and defines common variables and targets.

Provides:

- system info (arch, os)
- online registries' links
- docker tags
- bundle-related values & target
- catalog-related values
- controller-gen targets
- `run` target
- `update-version` target

Most values are provided as generated. Some require updating (e.g. `CHANNELS`) with each new version, which is done by
the `update-version` target or manually.

Any target with `(-dev)` can be built either for the `main` branch or the non-main branches (when the `-dev` is added
to the end of the target's name). Most differences will be related to tagging the artifacts.

## build.mk

Targets related to building the codebase, the bundle, and the catalog, and pushing the builds to the artifactory.
Provides necessary targets to communicate with the registries, as well as some values required for configuration
(other, common ones may be included in the core `Makefile`).

The build processes are combined into the `all` target for convenience. This target is used in the automated pipeline.

### Docker

Interact with online registries and build the codebase.

#### docker-login

Log into the registry with `docker login`. Note that providing the base URL only is enough to push to all development
registries.

#### docker-push

Push data into the registry.

#### docker-manifest (-dev)

Create and push the manifests.

#### buildx (-dev)

Build and push the codebase.

### Binaries

Executables for different architectures.

Supported architectures:

- `amd64`

More architectures may be added in the future to support different deployment environments.

### Bundle

Building and pushing the bundle image.

#### bundle (-dev)

Build the bundle image.

#### bundle-push (-dev)

Push the bundle image.

### Catalog

Building and pushing the catalog image.

#### catalog (-dev)

Build the catalog image.

#### catalog-push (-dev)

Push the catalog image.

## deps.mk

Targets related to checking and installing the dependencies of other targets. Each dependency has a specific version and
location in the local bin folder assigned. Some dependencies may rely on this location being non-empty, while others
first check the dependency system-wide before downloading an executable into the bin folder.

In general, each section provides the following three targets:
- `require-dependency`
- `check-dependency`
- `install-dependency`

TODO: This structure should be simplified with a common function and arguments passing.

Each time the `check-dependency` function passes, an extra check is made to compare versions of the installed and
required software. In case of a mismatch, a warning is `echo`-ed.

The processes are combined into the `all` target for convenience. This target is useful for local development (call it
once on project setup to never have to download dependencies again).

### buildx

First checks if the `docker buildx` command works already. If not, `require-cli-plugins-dir` target is used to download
the binary locally and create a symlink into the `~/.docker` directory.

This is required in case `buildx` was already installed with something else, for example provided with docker desktop.

### operator-sdk

Standard check and install processes against the local directory.

### kustomize

Standard check and install processes against the local directory.

### controller-gen

Standard check and install processes against the local directory.

### yq

Standard check and install processes against the local directory.

### opm

First checks if the `docker buildx` command works already. If not, follow the standard check and install processes
against the local directory.

This is required in case `opm` was already installed with something else, for example golang.

## lint.mk

At the moment, there is only a single lint target (also added to `all`), but more will be added in the future.
