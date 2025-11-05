# Video Storage AI - API

Backend API for Video Storage AI, built with Go, Gin, and SQLite.

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+** - [Download](https://golang.org/dl/)
- **Make** (optional but recommended) - Comes with Git for Windows
- **FFmpeg 8.0** - For video processing

### Installation

1. **Clone the repository and navigate to the API directory:**
   ```bash
   cd "c:\Repos\Video Storage AI\api"
   ```

2. **Copy the environment file:**
   ```bash
   copy .env.example .env
   ```

3. **Edit `.env` and add your AdultDataLink API key:**
   ```
   ADULTDATALINK_API_KEY=your_api_key_here
   ```

4. **Install dependencies:**
   ```bash
   make deps
   ```
   Or without Make:
   ```bash
   go mod download
   go mod tidy
   ```

### Running the Server

#### Using Make (Recommended)

```bash
# Run in development mode (standard)
make run

# Run with hot-reload (requires air)
make dev

# Build and run the binary
make build
./bin/video-storage-ai.exe
```

#### Without Make

```bash
# Run directly
go run cmd/server/main.go

# Build first, then run
go build -o bin/video-storage-ai.exe cmd/server/main.go
./bin/video-storage-ai.exe
```

The API will start on `http://localhost:8080`

### Health Check

Visit `http://localhost:8080/health` to verify the server is running.

## 📁 Project Structure

```
api/
├── cmd/
│   └── server/          # Application entry point
│       └── main.go
├── internal/            # Private application code
│   ├── api/             # HTTP handlers and routes
│   ├── config/          # Configuration management
│   ├── database/        # Database setup and migrations
│   ├── middleware/      # HTTP middleware (CORS, logging, etc.)
│   ├── models/          # Data models
│   └── services/        # Business logic
├── pkg/                 # Public libraries
│   └── utils/           # Utility functions
├── assets/              # Static files (videos, thumbnails)
│   └── performers/      # Performer preview videos
├── data/                # SQLite database (auto-created)
├── .env                 # Environment variables (not in git)
├── .env.example         # Environment template
├── Makefile             # Build automation
└── README.md            # This file
```

## 🛠️ Development

### Available Make Commands

```bash
make help           # Show all available commands
make deps           # Download dependencies
make build          # Build optimized binary
make build-debug    # Build with debug symbols
make run            # Run without building
make dev            # Run with hot-reload (requires air)
make test           # Run tests
make test-coverage  # Run tests with coverage report
make clean          # Remove build artifacts
make db-reset       # Reset database (deletes all data)
make fmt            # Format code
make vet            # Run go vet
make lint           # Run linter (requires golangci-lint)
make check          # Run fmt, vet, and test
```

### Hot Reload with Air

For the best development experience, install Air for automatic reloading:

```bash
make install
# Or manually:
go install github.com/cosmtrek/air@latest
```

Then run:
```bash
make dev
```

### Configuration

All configuration is managed through environment variables in `.env`:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | API server port | `8080` |
| `SERVER_HOST` | API server host | `localhost` |
| `SERVER_MODE` | Gin mode (debug/release) | `debug` |
| `DATABASE_PATH` | SQLite database path | `./data/video_storage.db` |
| `ASSETS_BASE_DIR` | Base directory for assets | `./assets` |
| `THUMBNAIL_DIR` | Thumbnail storage path | `./assets/thumbnails` |
| `PERFORMER_DIR` | Performer previews path | `./assets/performers` |
| `ADULTDATALINK_API_KEY` | AdultDataLink API key | *required* |

## 📡 API Endpoints

### Core Resources

- **Videos**: `/api/v1/videos`
- **Performers**: `/api/v1/performers`
- **Studios**: `/api/v1/studios`
- **Groups**: `/api/v1/groups`
- **Tags**: `/api/v1/tags`
- **Activity Logs**: `/api/v1/activity`
- **File Operations**: `/api/v1/files`
- **AI Assistant**: `/api/v1/ai`

### Static Assets

- **Assets**: `/assets/*` - Serves performer previews, thumbnails, etc.

### Health Check

```
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "database": "connected",
  "version": "0.1.0"
}
```

## 🗄️ Database

The application uses **SQLite** for simplicity and portability. The database is automatically created on first run.

### Schema

- `videos` - Video metadata and file information
- `performers` - Performer information and previews
- `studios` - Studio information
- `groups` - Sub-labels under studios
- `tags` - Video tags
- `video_performers` - Many-to-many relationship
- `video_tags` - Many-to-many relationship
- `video_studios` - Video-studio relationships
- `video_groups` - Video-group relationships
- `activity_logs` - Background task monitoring

### Database Management

```bash
# Reset database (WARNING: deletes all data)
make db-reset

# Database location
./data/video_storage.db
```

## 🔧 Building for Production

```bash
# Build optimized binary
make build

# Output location
bin/video-storage-ai.exe

# Set production mode in .env
SERVER_MODE=release
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# View coverage report
# Opens coverage.html in browser
```

## 📦 Dependencies

Key dependencies:
- **[Gin](https://github.com/gin-gonic/gin)** - HTTP web framework
- **[SQLite3](https://github.com/mattn/go-sqlite3)** - Database driver
- **[godotenv](https://github.com/joho/godotenv)** - Environment variable management
- **[CORS](https://github.com/gin-contrib/cors)** - CORS middleware

See [go.mod](go.mod) for full dependency list.

## 🐛 Troubleshooting

### Port Already in Use

If port 8080 is in use, change `SERVER_PORT` in `.env`:
```
SERVER_PORT=8081
```

### Database Locked

Stop all running instances of the server:
```bash
# Windows
taskkill /F /IM video-storage-ai.exe

# Or reset the database
make db-reset
```

### Missing Dependencies

```bash
make deps
# Or
go mod download
go mod tidy
```

## 📝 License

See root [LICENSE](../LICENSE) file.

## 🤝 Contributing

This is a personal project, but suggestions and feedback are welcome!

---

**Next Steps:**
1. Complete handler implementations in `internal/api/`
2. Add model structs in `internal/models/`
3. Implement services in `internal/services/`
4. Integrate AdultDataLink API
5. Add video scanning and FFmpeg integration
