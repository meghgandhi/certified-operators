# Testing infrastructure and processes

Testing can be split into manual and automated, both requiring thoughtful design and documentation. While manual testing
is quite informal and only tips or good practices will be included, automated tests cover a wide range of code and
should be maintained and documented with great care.

## Automated tests

Automated tests that can be invoked locally or via automated pipeline processes (such as travis build jobs). Some tests
may involve a local cluster creation.

### Structure

Each of the test types is included as follows:

- Unit tests are added next to the source code files, with the `_test` suffix.
- Integration tests are added similarly to the unit tests
- End-to-end tests are added in a separate `tests/e2e` folder
- Build validation tests are added in a separate `tests/bvt` folder

Upon running the tests, a coverage report is also generated.

TODO: Relevant `make` commands are yet to be designed and documented.

In addition, external repositories may hold more tests with different architectures, goals, tools, and similar. The
purpose of this repository is to simply provide tests tightly-coupled with the codebase.

## Manual testing

TODO: Manual testing processes are yet to be designed and documented.
