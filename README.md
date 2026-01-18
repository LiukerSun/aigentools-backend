# AiGenTools Backend

AiGenTools Backend is a RESTful API server built with Go (Golang). It provides the core business logic for managing users, AI models, transactions, and tasks.

## Features

- **User Management:** Registration, login, and profile management using JWT authentication.
- **AI Model Integration:** Support for various AI models and parameter management.
- **Transaction System:** Tracking user transactions and payments.
- **Task Execution:** Handling AI-related tasks with asynchronous processing.
- **Admin Dashboard:** Administrative APIs for system management.
- **API Documentation:** Integrated Swagger UI for easy API exploration.

## Tech Stack

- **Framework:** [Gin](https://gin-gonic.com/)
- **Database:** [PostgreSQL](https://www.postgresql.org/) with [GORM](https://gorm.io/)
- **Cache:** [Redis](https://redis.io/)
- **Authentication:** JWT (JSON Web Tokens)
- **API Docs:** [Swagger](https://github.com/swaggo/swag)
- **Logger:** [Uber-go Zap](https://github.com/uber-go/zap)

## Prerequisites

- Go 1.25+
- PostgreSQL
- Redis
- Docker & Docker Compose (optional)

## Getting Started

### Using Docker (Recommended)

The easiest way to get started is using Docker Compose:

1. **Clone the repository:**
   ```bash
   git clone https://github.com/LiukerSun/aigentools-backend.git
   cd aigentools-backend
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Update .env with your specific configurations
   ```

3. **Start the services:**
   ```bash
   docker-compose up -d
   ```

The API will be available at `http://localhost:8080`.

### Manual Setup

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Run database migrations:**
   Migrations are handled automatically when the application starts.

3. **Run the application:**
   ```bash
   go run main.go
   ```

## API Documentation

Once the server is running, you can access the Swagger UI at:
`http://localhost:8080/swagger/index.html`

## Project Structure

```text
├── cmd/             # CLI commands and tools
├── config/          # Application configuration logic
├── docs/            # Swagger documentation files
├── internal/        # Private application and library code
│   ├── api/         # Route definitions and handlers
│   ├── database/    # Database connection and initialization
│   ├── middleware/  # Gin middlewares (Auth, Logger, etc.)
│   ├── models/      # GORM models and data structures
│   ├── services/    # Business logic layer
│   └── utils/       # Utility functions
├── pkg/             # Public library code
└── main.go          # Application entry point
```

## Development

### Running Tests
```bash
go test ./...
```

### Updating Swagger Docs
```bash
swag init
```

## License

This project is licensed under the MIT License.
