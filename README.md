# helper-go

[![Go Reference](https://pkg.go.dev/badge/github.com/alfin-efendy/helper-go.svg)](https://pkg.go.dev/github.com/alfin-efendy/helper-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/alfin-efendy/helper-go)](https://goreportcard.com/report/github.com/alfin-efendy/helper-go)
[![CI](https://github.com/alfin-efendy/helper-go/workflows/CI/badge.svg)](https://github.com/alfin-efendy/helper-go/actions/workflows/ci.yml)
[![Code Quality](https://github.com/alfin-efendy/helper-go/workflows/Code%20Quality/badge.svg)](https://github.com/alfin-efendy/helper-go/actions/workflows/code-quality.yml)
[![codecov](https://codecov.io/gh/alfin-efendy/helper-go/branch/main/graph/badge.svg)](https://codecov.io/gh/alfin-efendy/helper-go)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/release/alfin-efendy/helper-go.svg)](https://github.com/alfin-efendy/helper-go/releases/latest)

A comprehensive Go library providing essential utilities for backend developers. This library includes configuration management, HTTP server utilities, database helpers, logging, and various utility functions to accelerate development.

## 🚀 Features

- **Configuration Management**: Environment-based configuration with validation
- **HTTP Server**: Chi router with automatic route listing and middleware
- **Database Utilities**: SQL and Redis connection helpers with telemetry
- **Logging**: Structured logging with zerolog and GORM integration
- **Utility Functions**: File manipulation, stack frame inspection, and more
- **Telemetry**: OpenTelemetry integration for observability

## 📦 Installation

```bash
go get github.com/alfin-efendy/helper-go
```

## 🔧 Quick Start

### Sample Config

Create file `config.yml` on root project.

```yaml
app:
  name: ${APP_NAME}
  mode: ${APP_MODE} # debug/release/test, for production use release

log:
  level: ${LOG_LEVEL} #debug/info/warn/error/fatal
  location: ./logs/app.log

otel:
  address: ${OTEL_ADDRESS}
  timeout: 30
  trace: true
  metric: false

server:
  restApi:
    host: 0.0.0.0
    port: 5000

database:
  sql:
    driver: postgresql
    host: ${DATABASE_HOST$}
    port: 5432
    database: ${DATABASE_NAME}
    username: ${DATABASE_USER}
    password: ${DATABASE_PASSWROWD}
    poolingConnection:
      maxIdle: 10
      maxOpen: 100
      maxLifetime: 0
  redis:
    mode: ${REDIS_MODE} # single/sentinel
    address: ${REDIS_ADDRESS}
    db: 0
    poolSize: 10
    minIdleConns: 5
    maxConnAge: 0
    poolTimeout: 4
    idleTimeout: 300
    readTimeout: 500
    writeTimeout: 500
    maxRetries: 3
    minRetryBackoff: 8
    maxRetryBackoff: 512
    dialTimeout: 5
    tls: false
    tlsSkipVerify: false
    tlsServerName: ""
    tlsConfig: ""
    tlsInsecureSkipVerify: false
    tlsRootCA: ""
    tlsClientCert: ""
    tlsClientKey: ""
    tlsClientAuth: ""
    tlsHandshakeTimeout: 5
    tlsKeepAlive: 0
    tlsKeepAlivePeriod: 0
    tlsSessionTicketKey: ""
    tlsPreferServerCipherSuites: false
    tlsCipherSuites: ""
    tlsMinVersion: ""
    tlsMaxVersion: ""
    tlsCurvePreferences: ""
```
Environment Variable Substitution

```yaml
app:
  name: "${APP_NAME}"  # Automatic Replaced with env var value
```

```go
package main

import (
	"github.com/alfin-efendy/helper-go/server/http"
	"github.com/alfin-efendy/helper-go/app"
	"github.com/alfin-efendy/helper-go/database"
	"github.com/go-chi/chi/v5"
	"github.com/you/repo/data"
)

func main() {
	ctx := app.Start(
		func() {
			data.Migrate()
		}
	)

	database.InitSQL(ctx)
	
	router := chi.NewRouter()
	
	// Add your routes
	router.Get("/api/users", Handler(HealthCheck))
	
	// Start server with automatic route listing
	server := http.NewServer(":8080", router)
	server.Start() // This will print all registered routes
}
```

## 📖 Documentation

### Package Structure

```
├── config/          # Configuration management
│   └── schema/      # Configuration schemas
├── database/        # Database utilities (SQL, Redis)
├── logger/          # Logging utilities
├── server/          # HTTP server utilities
│   └── http/        # Chi router extensions
└── util/            # General utility functions
```

### Configuration

The configuration package supports:
- Environment variable loading
- Validation using struct tags
- Nested configuration structures
- Default values

### HTTP Server

Features include:
- Automatic route discovery and listing
- Context utilities
- Middleware integration
- Graceful shutdown support

### Database

Utilities for:
- SQL database connections with telemetry
- Redis connections with telemetry
- Connection pooling
- Health checks

### Logging

Structured logging with:
- JSON output
- Multiple log levels
- GORM integration
- Performance logging

## 🧪 Testing

Run tests with coverage:

```bash
# Using Make
make test-cover

# Using Go directly
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Run benchmarks:

```bash
make benchmark
```

## 🔍 Code Quality

This project maintains high code quality standards:

```bash
# Run all quality checks
make quality

# Individual checks
make lint          # Linting
make vet           # Go vet
make security      # Security scanning
make vuln-check    # Vulnerability checking
make complexity    # Complexity analysis
```

## 🚀 CI/CD

The project includes comprehensive GitHub Actions workflows:

- **CI Pipeline**: Tests across multiple Go versions and platforms
- **Code Quality**: Static analysis, security scanning, and complexity checks
- **Dependency Updates**: Automated dependency management
- **Release Management**: Automated releases with changelog generation

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feat/amazing-feature`)
3. Run quality checks (`make quality`)
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feat/amazing-feature`)
6. Open a Pull Request

### Development Setup

```bash
# Install development tools
make install-tools

# Run all checks before committing
make ci
```

## 📋 Requirements

- Go 1.25 or higher

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🎯 Roadmap

- [ ] GraphQL server utilities
- [ ] Message queue integrations
- [ ] Enhanced metrics and monitoring
- [ ] Docker utilities
- [ ] Kubernetes helpers

## 💬 Support

- 📧 Create an issue for bug reports or feature requests
- 📖 Check the [documentation](https://pkg.go.dev/github.com/alfin-efendy/helper-go)
- 💡 See [examples](examples/) for usage patterns

## 🙏 Acknowledgments

- [Chi Router](https://github.com/go-chi/chi) - HTTP router
- [Zerolog](https://github.com/rs/zerolog) - Structured logging
- [Viper](https://github.com/spf13/viper) - Configuration management
- [OpenTelemetry](https://opentelemetry.io/) - Observability