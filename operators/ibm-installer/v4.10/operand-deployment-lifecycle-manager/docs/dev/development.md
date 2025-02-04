<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Development Guide](#development-guide)
  - [Prerequisite](#prerequisite)
  - [Developer quick start](#developer-quick-start)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Development Guide

## Prerequisite

- git
- go version v1.12+
- Linting Tools

    | linting tool | version |
    | ------------ | ------- |
    | [hadolint](https://github.com/hadolint/hadolint#install) | [v1.17.2](https://github.com/hadolint/hadolint/releases/tag/v1.17.2) |
    | [shellcheck](https://github.com/koalaman/shellcheck#installing) | [v0.7.0](https://github.com/koalaman/shellcheck/releases/tag/v0.7.0) |
    | [yamllint](https://github.com/adrienverge/yamllint#installation) | [v1.17.0](https://github.com/adrienverge/yamllint/releases/tag/v1.17.0)
    | [helm client](https://helm.sh/docs/using_helm/#install-helm) | [v2.10.0](https://github.com/helm/helm/releases/tag/v2.10.0) |
    | [golangci-lint](https://github.com/golangci/golangci-lint#install) | [v1.18.0](https://github.com/golangci/golangci-lint/releases/tag/v1.18.0) |
    | [autopep8](https://github.com/hhatto/autopep8#installation) | [v1.4.4](https://github.com/hhatto/autopep8/releases/tag/v1.4.4) |
    | [mdl](https://github.com/markdownlint/markdownlint#installation) | [v0.5.0](https://github.com/markdownlint/markdownlint/releases/tag/v0.5.0) |
    | [awesome_bot](https://github.com/dkhamsing/awesome_bot#installation) | [1.19.1](https://github.com/dkhamsing/awesome_bot/releases/tag/1.19.1) |
    | [sass-lint](https://github.com/sasstools/sass-lint#install) | [v1.13.1](https://github.com/sasstools/sass-lint/releases/tag/v1.13.1) |
    | [tslint](https://github.com/palantir/tslint#installation--usage) | [v5.18.0](https://github.com/palantir/tslint/releases/tag/5.18.0)
    | [prototool](https://github.com/uber/prototool/blob/dev/docs/install.md) | `7df3b95` |
    | [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports) | `3792095` |

## Developer quick start

- Run the `code-dev` to check your code and run the unit test.

```bash
make code-dev
```

- Build and push the docker image for local development.

```bash
export DEV_REGISTRY=<YOUR_CUSTOMIZED_IMAGE_REGISTRY>
make build-push-dev-image
```

> **Note:** You need to login the docker registry before running the command above.
