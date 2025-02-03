# Create project

Instructions on how was this project created. May come in handy in case of creating a similar project.

## Codebase setup

Codebase setup and initial implementation changes.

### Generate code with operator-sdk

Based on the [operator-sdk documentation][operator-sdk-docs] and the
[kubebuilder documentation][kubebuilder-documentation].

[operator-sdk-docs]: https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/#create-a-new-project
[kubebuilder-documentation]: https://book.kubebuilder.io/plugins/go-v4-plugin

Initialise the project with `operator-sdk`:

```shell
operator-sdk init --domain ibm.com --repo github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator --plugins=go/v4-alpha
```

Latest `v4` plugin is used to get the latest, most modern structure available with the `kubebuilder`. The other
parameters are following similar IBM projects' practices.

Initialise API and scaffold basic reconcile and test structure:

```shell
operator-sdk create api --group operator --version v1alpha1 --kind Scanner --resource --controller
```

As before, the parameters are similar to the other operators at IBM.

### Configure and extend generated code

Once the main codebase has been successfully generated, some files must be adjusted to allow integration with existing
tools and creation of acceptable artifacts. Changes are majorly due to the targets provided with the `Makefile`(s), as
all build and push processes are defined there.

The `Makefile`(s) is quite complex, therefore it is thoroughly described in a [separate document][makefile-docs]. You
should refer to it to learn about specific targets. If you want to learn about general processes instead, you should
refer to the [pipeline documentation][pipeline-docs].

The following changes were applied to the generated resources:

- Created a base manifest file at `config/manifests/bases`. See [`base-manifest.md` docs][base-manifest] for more
  information
- Adjusted `namespace` and `namePrefix` attributes at `config/default/kustomization.yaml`. This step is optional unless
  the names are too long
- Ran `kustomize edit fix --vars` to adjust outdated variables in all directories with `kustomization.yaml` files (so
  generally all directories within `config`)
- Replaced manager's container command (at `config/manager/manager.yaml`) with `ibm-license-service-scanner-operator`
to match the `Dockerfile` configuration.
- Ran `make generate manifests bundle` to make sure all extra files are generated

The last command is important as it must also be run during the development to update relevant files when needed
(part of the development pipeline).

Furthermore, the following files were also added to make the `Makefile`(s) functional:

- Base `Dockerfile` with the operator image
- Scripts to check & install dependencies

Note that any `Makefile`(s) additional to the main, core file, are present in the `targets` directory.

[makefile-docs]: ../development/makefile.md
[pipeline-docs]: ../development/pipeline.md
[base-manifest]: ./base-manifest.md

### Configure logging

Default logging has been adjusted in the `main.go` file.

### Add tests

The structure is documented in the [testing document][testing-docs].

TODO: Sample tests are yet to be added.

[testing-docs]: ../development/testing.md

### Add license

A standard IBM `LICENSE` file was added to the repository, with a few small adjustments, so that the text addresses
the License Service Scanner.

Furthermore, the boilerplate text file from the `hack` directory was modified to include IBM company info in the header.
All files affected by this text have been re-generated, and all others edited manually. At this stage, all files in the
repository should include the license information at the top of the file.

### Add documentation

A new `docs` folder was created to keep developer documentation. Keeping a reasonable amount of developer documentation
is beneficial both in short and long terms, for example:
- Avoiding single point of failures and complicated knowledge transfers
- Keeping useful, project-related commands and processes in one place
- Giving an overview of the project architecture without having to jump into the code

There is no specific structure to the documentation, but topics should be logically grouped up into sections
(sub-folders). Naturally, the documentation should be kept up-to-date at all times.

### Add linting

Modify `Makefile` by adding the `golangci-lint` command and structure for more linters. The linter runs based on a
`yml` file included in the root level of the project (for automatic detection).

Refer to the [golangci-lint repository][golangci-lint-repository] for most recent files and available config.

[golangci-lint-repository]: https://github.com/golangci/golangci-lint/blob/master/

## GitHub

GitHub repository setup to be compliant with IBM development guidelines.

The repository was created under a private workspace, as previously decided. Access was granted to selected team members
to enable contributions ot the codebase and overseeing of the development.

### Adjust .gitignore and .dockerignore

Small changes to make sure the git repository and docker builds are successful:

- Add editor config files to `.gitignore` (e.g. `.idea`)
- Add `/bin` folder to `.gitignore`

Note that the base `.gitignore` was templated for golang.

### Enable Travis

Travis must be enabled to allow basic, automated CI/CD processes.

The following steps should be followed:

- Change repo visibility to `public`
- Make sure Travis can see your repository in the [GitHub apps interface][github-apps]. In case it’s not visible, delete
and re-enable GitHub apps on it.
- Visit https://v3.travis.ibm.com/github/cloud-license-reporter/repo (replace `repo` with your repository's URL)
- If the repository is not active, click the `activate repository` button
- Wait for Travis to set everything up
- Change repo visibility to `private`
- Create `.travis.yml` file and trigger your first build

In case of any problems, contact Travis administrator using the `#whitewater-travis` Slack channel.

Once enabled, each commit should trigger development-specific targets, and prod-specific targets on `main` branch. See
`.travis.yml` file to learn about the exact commands executed by the pipeline.

[github-apps]: https://v3.travis.ibm.com/organizations/cloud-license-reporter/repositories

### Configure Whitesource

Whitesource configuration file (`.whitesource`) should have been automatically created for the repository.

The only step needed to be taken is to merge the pull request with the file into the main branch.