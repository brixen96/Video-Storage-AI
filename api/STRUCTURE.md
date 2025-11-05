# Video Storage AI - API Structure

## Directory Layout

```
api/
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
│
├── internal/                          # Private application code
│   ├── api/
│   │   └── router.go                  # Route definitions and handler stubs
│   ├── config/
│   │   └── config.go                  # Configuration management
│   ├── database/
│   │   └── database.go                # Database initialization and schema
│   ├── middleware/
│   │   ├── cors.go                    # CORS middleware
│   │   └── logger.go                  # Logging and recovery middleware
│   ├── models/                        # Data models (to be implemented)
│   └── services/                      # Business logic (to be implemented)
│
├── pkg/
│   └── utils/                         # Utility functions (to be implemented)
│
├── assets/                            # Static files
│   ├── performers/                    # Performer preview videos
│   │   ├── [Performer Name]/
│   │   │   └── *.mkv, *.webm
│   └── thumbnails/                    # Video thumbnails (auto-generated)
│
├── data/
│   ├── .gitkeep                       # Keep directory in git
│   └── video_storage.db               # SQLite database (auto-created)
│
├── bin/                               # Compiled binaries
│   └── video-storage-ai.exe
│
├── tmp/                               # Temporary files (air hot-reload)
│
├── .env                               # Environment variables (not in git)
├── .env.example                       # Environment template
├── .gitignore                         # Git ignore rules
├── .air.toml                          # Air hot-reload configuration
├── Makefile                           # Build automation
├── go.mod                             # Go module definition
├── go.sum                             # Go dependency checksums
├── README.md                          # API documentation
└── STRUCTURE.md                       # This file
```

## Core Components

### Configuration (`internal/config/`)
- Environment variable management
- Default values and validation
- Type-safe configuration struct

### Database (`internal/database/`)
- SQLite connection management
- Schema creation and migrations
- Connection pooling
- Health checks

### Middleware (`internal/middleware/`)
- **CORS**: Cross-Origin Resource Sharing for Vue frontend
- **Logger**: HTTP request logging with custom format
- **Recovery**: Panic recovery for graceful error handling

### API Router (`internal/api/`)
All endpoints are prefixed with `/api/v1/`

#### Resource Endpoints
- `/videos` - Video CRUD operations
- `/performers` - Performer management
- `/studios` - Studio management
- `/groups` - Group management (sub-labels)
- `/tags` - Tag management

#### Feature Endpoints
- `/activity` - Activity monitor logs and status
- `/files` - File operations (scan, rename, move, delete)
- `/ai` - AI assistant features

#### Static Assets
- `/assets/*` - Serve static files (videos, thumbnails)

#### Health Check
- `/health` - Database and server health status

## Database Schema

### Tables
1. **videos** - Video metadata and file information
2. **performers** - Performer information and preview paths
3. **studios** - Studio information
4. **groups** - Sub-labels under studios
5. **tags** - Video tags with colors and icons
6. **video_performers** - Many-to-many: videos ↔ performers
7. **video_tags** - Many-to-many: videos ↔ tags
8. **video_studios** - Many-to-many: videos ↔ studios
9. **video_groups** - Many-to-many: videos ↔ groups
10. **activity_logs** - Background task monitoring

### Indexes
- Optimized for common queries (file paths, names, dates)
- Task type and status for activity monitoring

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `SERVER_PORT` | API server port | `8080` |
| `SERVER_HOST` | API server host | `localhost` |
| `SERVER_MODE` | Gin mode (debug/release) | `debug` |
| `DATABASE_PATH` | SQLite database file path | `./data/video_storage.db` |
| `ASSETS_BASE_DIR` | Base directory for assets | `./assets` |
| `THUMBNAIL_DIR` | Thumbnail storage | `./assets/thumbnails` |
| `PERFORMER_DIR` | Performer previews | `./assets/performers` |
| `ADULTDATALINK_API_KEY` | AdultDataLink API key | *required* |

## Next Implementation Steps

### Phase 1: Models
- [ ] Define Go structs for all entities
- [ ] Add JSON tags for API responses
- [ ] Add validation tags

### Phase 2: Database Operations
- [ ] Implement CRUD operations for each entity
- [ ] Add search and filtering
- [ ] Add pagination support

### Phase 3: Handler Implementation
- [ ] Replace placeholder handlers with real implementations
- [ ] Add request validation
- [ ] Add error handling
- [ ] Add response formatting

### Phase 4: File Operations
- [ ] Video file scanning
- [ ] FFmpeg integration for metadata extraction
- [ ] Thumbnail generation
- [ ] File management (rename, move, delete)

### Phase 5: External API Integration
- [ ] AdultDataLink API client
- [ ] Metadata fetching and mapping
- [ ] Error handling and retries

### Phase 6: AI Features
- [ ] AI assistant chat interface
- [ ] Auto-tagging suggestions
- [ ] Naming convention suggestions
- [ ] Library analysis

### Phase 7: Activity Monitor
- [ ] Background task tracking
- [ ] Progress reporting
- [ ] Real-time status updates
- [ ] WebSocket support for live updates

### Phase 8: Testing
- [ ] Unit tests for all packages
- [ ] Integration tests for API endpoints
- [ ] End-to-end tests

## Build and Run

### Development
```bash
# Run with hot-reload
make dev

# Run normally
make run

# Run tests
make test
```

### Production
```bash
# Build optimized binary
make build

# Run binary
./bin/video-storage-ai.exe
```

### Useful Commands
```bash
make help         # Show all commands
make clean        # Clean build artifacts
make db-reset     # Reset database
make check        # Run fmt, vet, and test
```

## Dependencies

### Core
- **gin-gonic/gin** - HTTP web framework
- **mattn/go-sqlite3** - SQLite driver
- **joho/godotenv** - Environment management
- **gin-contrib/cors** - CORS middleware

### Planned
- **FFmpeg bindings** - Video processing
- **WebSocket** - Real-time updates
- **HTTP client** - External API calls

## Development Guidelines

1. **Keep internal packages private** - Use `internal/` for app-specific code
2. **Use interfaces** - Make services testable and swappable
3. **Validate input** - Always validate API requests
4. **Handle errors gracefully** - Return appropriate HTTP status codes
5. **Log important events** - Use structured logging
6. **Write tests** - Aim for high coverage
7. **Document APIs** - Add comments for exported functions
8. **Use constants** - Avoid magic strings and numbers

## Security Considerations

1. **API Key Protection** - Never commit `.env` to git
2. **Input Validation** - Sanitize all user input
3. **Path Traversal** - Validate file paths
4. **SQL Injection** - Use parameterized queries (already done)
5. **CORS** - Configure allowed origins properly for production
6. **Rate Limiting** - Add rate limiting middleware (future)

## Performance Optimizations

1. **Connection Pooling** - Already configured for SQLite
2. **Indexing** - Database indexes on common queries
3. **Caching** - Consider adding Redis for frequently accessed data
4. **Pagination** - Implement pagination for large result sets
5. **Compression** - Enable gzip compression for responses
6. **Static Assets** - Consider CDN for production

---

**Status**: ✅ Foundation Complete
**Next**: Implement models and handlers
