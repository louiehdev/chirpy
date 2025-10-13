# chirpy

**chirpy** is a lightweight REST API built in Go, developed as part of backend API practice through [boot.dev](https://www.boot.dev/). It provides endpoints for user management, authentication, and posting short-form “chirps.” The project focuses on clean API design, token-based auth, and structured request handling.

---

## Features

- User registration and authentication (with JWTs)
- Create, fetch, and delete chirps
- Token refresh and revocation
- Admin metrics and database reset endpoints
- Polka webhooks for upgrading users
- Structured HTTP routing with Go’s `net/http` and `ServeMux`

---

## Prerequisites

Before you can run **chirpy**, ensure you have:

- **Go** (version 1.22 or newer recommended)
- **PostgreSQL** (running and accessible)

---

## Installation

Clone the repository:

```bash
git clone https://github.com/louiehdev/chirpy.git
cd chirpy
```

Build or install:

```bash
go install
```

---

## Configuration

chirpy expects environment variables or configuration to define database connection and other runtime settings.

Example .env or environment setup:

```bash
DB_URL=postgres://username:password@localhost:5432/chirpydb?sslmode=disable
JWT_SECRET=your-secret-key
POLKA_API_KEY=your-polka-key
```

Adjust:

- username and password
- host and port
- database name
- sslmode if needed

Ensure the database exists and credentials are valid.

---

## API Overview

Below is a summary of available routes and their purposes.
|Method|	Endpoint | Description|
|:---|:------------|:-----------:|
|GET|	/api/healthz|	Health check|
|GET|	/admin/metrics|	Returns internal metrics|
|GET|	/api/chirps|	Fetch all chirps|
|GET|	/api/chirps/{chirpID}|	Fetch a specific chirp|
|POST|	/api/chirps|	Create a new chirp|
|DELETE|	/api/chirps/{chirpID}|	Delete a chirp|
|POST|	/api/users|	Create a new user|
|PUT|	/api/users|	Update user info|
|POST|	/api/login|	Authenticate user|
|POST|	/api/refresh|	Refresh a token|
|POST|	/api/revoke|	Revoke a token|
|POST|	/admin/reset|	Reset database state (admin only)|
|POST|	/api/polka/webhooks|	Handle Polka webhook events|

---

## Running the Server

You can run the server locally with:

```bash
go run .
```

The API will be available at:

```bash
http://localhost:8080
```
