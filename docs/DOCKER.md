# Docker Build & Deployment Guide

This guide describes how to build, run, and manage the Book Dragon API using Docker and Docker Compose.

---

## 1. Prerequisites

Before getting started, ensure that you have the following installed on your local machine:
- [Docker Engine](https://docs.docker.com/engine/install/) / [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- [Docker Compose](https://docs.docker.com/compose/install/) (comes bundled with Docker Desktop)

---

## 2. Building the Docker Image

The repository includes a multi-stage `Dockerfile` which compiles the Go application with SQLite (`CGO_ENABLED=1`) and packages it into a minimal Debian image.

To build the image locally, navigate to the root directory of the project and run:

```bash
docker build -t book-dragon-api .
```

To build and tag the image for the DigitalOcean Container Registry (matching the configuration in `docker-compose.yml`), use:

```bash
docker build -t registry.digitalocean.com/book-dragon/images:api .
```

---

## 3. Running the API with Docker Compose

Running via Docker Compose is the recommended way to test and run the application. It automatically builds the application (if no image is found), sets up the port bindings, and configures a persistent data volume.

To spin up the service in the background:

```bash
docker compose up -d
```

### Checking Logs & Status

To view the running containers and status:
```bash
docker compose ps
```

To view the API logs:
```bash
docker compose logs -f
```

To stop the service:
```bash
docker compose down
```

### Docker Compose Volume & Persistence
In the `docker-compose.yml`, the local `./data` directory is mounted to `/app` inside the container:
```yaml
volumes:
  - ./data:/app
```
Since the Go application's database file `bookdragon.db` is initialized in the working directory `/app`, mounting the local `./data` folder allows the database to be persisted across container restarts in the host's `./data` directory.

---

## 4. Running the API directly via `docker run`

If you prefer to run the container manually using the `docker` CLI without Compose, use the command below:

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app \
  --name book-dragon-api \
  --restart unless-stopped \
  book-dragon-api
```

### Command Breakdown:
- `-d`: Runs the container in the background (detached mode).
- `-p 8080:8080`: Maps port `8080` on the host to port `8080` inside the container.
- `-v $(pwd)/data:/app`: Mounts the `./data` folder on the host to `/app` in the container for SQLite database persistence.
- `--name book-dragon-api`: Assigns a descriptive name to the running container.
- `--restart unless-stopped`: Ensures the container starts up automatically if the Docker daemon or machine restarts.

---

## 5. Testing the Dockerized API

Once the container is running (either via Compose or direct `docker run`), the API is accessible on port `8080`.

### Access Swagger UI
You can open the interactive documentation in your browser to verify the API is up:
[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

### Using cURL to test
You can also use `curl` to test the registration endpoint:

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "dragonrider",
    "email": "rider@example.com",
    "password": "supersecretpassword"
  }'
```

