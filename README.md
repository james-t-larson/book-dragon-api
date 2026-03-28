# Book Dragon API

Welcome to the Book Dragon API! This is a RESTful backend service built with Go and the Chi router, featuring user authentication and SQLite for data storage.

## Prerequisites

- [Go](https://go.dev/) (1.20 or later recommended)

### git hooks

To ensure tests run automatically on commit, run the following command once to configure Git hooks:
```bash
git config core.hooksPath .githooks
```

## Starting the Project

1. Ensure you have downloaded the required dependencies:
   ```bash
   go mod download
   ```

2. Start the API server:
   ```bash
   go run cmd/api/main.go
   ```

The server will start on `http://localhost:8080` by default. It will also automatically create a `bookdragon.db` SQLite database file in your current directory if it doesn't already exist.

## Testing Your First Endpoint

Once the server is running, you can test the **Register** endpoint. Open a new terminal window and use `curl` to create a new user account:

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "dragonrider",
    "email": "rider@example.com",
    "password": "supersecretpassword"
  }'
```

If successful, you should receive a `201 Created` response containing your new user details (minus the password!).

### Exploring the Interactive API Documentation

This project includes interactive OpenAPI (Swagger) documentation. Once your server is running, you can explore all available endpoints, their required schemas, and even test them directly from your browser by visiting:

[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
