# API Tests

This directory contains integration and endpoint tests for the `book-dragon` API.

## Running Tests

To run all tests in this directory, use the following command from the root of the project or from within this directory:

```bash
go test ./tests/...
```

To run a specific test (e.g., `TestRegister`), you can use the `-run` flag:

```bash
go test ./tests/... -run TestRegister
```

## Adding New Tests

When adding new tests, ensure that:
1. Files are aptly named with the `_test.go` suffix.
2. The package is declared as `tests`.
3. You use isolated test states, such as the in-memory SQLite database setup available in `setupTestStore`.
