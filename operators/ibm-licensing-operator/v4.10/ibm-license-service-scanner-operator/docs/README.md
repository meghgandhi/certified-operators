# Developer documentation

Documentation of the tool split into multiple subsections, to provide developers with accurate and up-to-date
information on product architecture, significant code sections, infrastructure design and maintenance instructions,
and similar.

This document should be extended by the product's developers only, as it is NOT meant to be provided to clients or any
other people (even internally) who are not directly involved in the development process. It should be kept as technical
as possible, to provide the highest degree of clarity and conciseness.

## Project setup

Documentation on configuring and setting up the project, to get it to a runnable state.

### [Create project][create-project]

Step-by-step instructions on how was this project created. Useful to follow if a similar product must be built. The
general process was split into two parts:

- [Codebase setup][codebase-setup]
- [Repository setup][repository-setup]

Each section is thoroughly described and includes commands which were run. Where applicable, reasoning behind tool
choices or project structure decisions is also provided.

[create-project]: ./setup/create-project.md
[codebase-setup]: ./setup/create-project.md#codebase-setup
[repository-setup]: ./setup/create-project.md#github

## Development

Documentation on daily development practices and processes related to writing clean code.

### [Pipeline][pipeline]

Steps every developer should go through when contributing to the repository, together with descriptions of available
tools and recommendations for running them.

Includes details on the `Makefile`(s) - see [makefile docs][makefile-docs].

[pipeline]: ./development/pipeline.md
[makefile-docs]: ./development/makefile.md

### [Testing][testing]

Documentation on testing methodology and processes.

Testing includes the following types of tests:

- Unit
- Integration
- End-to-End
- Build validation

Relevant details and `make` commands are provided to ensure an understanding on how to keep a high quality codebase.
Furthermore, coverage reports are generated to try to measure said quality.

[testing]: ./development/testing.md
