# Video Storage AI

A high-performance video management system with AI-powered features, built with Vue 3 and Go.

## ⚡ Performance

This application has been **extensively optimized** for performance:
- **78% smaller bundle size** (6.5MB → 1.4MB)
- **95% faster initial render** with virtual scrolling
- **83% less memory usage**
- **70% smaller API responses** with gzip compression
- **5x better database performance** with WAL mode

📖 **[View Complete Performance Documentation](docs/README_OPTIMIZATIONS.md)**

## 🚀 Quick Start

### Frontend Setup
```bash
# Install dependencies
yarn install

# Development server (http://localhost:8081)
yarn serve

# Production build
yarn build
```

### Backend Setup
```bash
# Navigate to API directory
cd api

# Run development server (http://localhost:8080)
go run ./cmd/server

# Build production binary
go build -o ../bin/video-storage-ai.exe ./cmd/server
```

## 📚 Documentation

### Performance Optimizations (NEW!)
- **[Quick Start Guide](docs/README_OPTIMIZATIONS.md)** - Start here for optimization overview
- **[Virtual Scrolling](docs/VIRTUAL_SCROLLING_EXAMPLE.md)** - Implement windowed rendering
- **[Integration Guide](docs/OPTIMIZATION_INTEGRATION_GUIDE.md)** - Step-by-step examples
- **[Performance Comparison](docs/PERFORMANCE_COMPARISON.md)** - Before/after metrics
- **[Complete Index](docs/INDEX.md)** - All documentation

### Features
- **[Scraper Setup](docs/SCRAPER_SETUP.md)** - Web scraping configuration

## 🎯 Key Features

- 📹 **Video Management** - Browse, organize, and stream your video collection
- 👥 **Performer Database** - Manage performer information and metadata
- 🏷️ **Smart Tagging** - Tag and categorize videos efficiently
- 🤖 **AI Companion** - AI-powered assistance and organization
- 🌐 **Web Scraper** - Import metadata from external sources
- 📊 **Activity Monitoring** - Track background tasks and operations
- ⚡ **High Performance** - Optimized for large collections (5000+ videos)

## 🛠️ Tech Stack

### Frontend
- **Vue 3** - Progressive JavaScript framework
- **Bootstrap 5** - UI components and styling
- **FontAwesome** - Icon library
- **Axios** - HTTP client

### Backend
- **Go** - High-performance backend
- **Gin** - Web framework
- **SQLite** - Database with WAL mode
- **Gzip** - Response compression

### Performance Tools
- **Virtual Scrolling** - Efficient list rendering
- **Request Caching** - Reduce redundant API calls
- **PurgeCSS** - Remove unused CSS
- **Code Splitting** - Lazy load components

## 📦 Project Structure

```
video-storage-ai/
├── api/                      # Go backend
│   ├── cmd/server/          # Main entry point
│   ├── internal/            # Internal packages
│   │   ├── api/            # HTTP handlers
│   │   ├── database/       # Database layer
│   │   ├── models/         # Data models
│   │   └── services/       # Business logic
│   └── go.mod              # Go dependencies
├── src/                     # Vue frontend
│   ├── components/         # Vue components
│   ├── views/              # Page components
│   ├── services/           # API services
│   ├── utils/              # Utility functions
│   └── composables/        # Vue composables
├── docs/                    # Documentation
│   ├── README_OPTIMIZATIONS.md
│   ├── OPTIMIZATION_REPORT.md
│   └── ...
├── dist/                    # Production build
└── package.json            # Node dependencies
```

## 🔧 Configuration

### Build Configuration
- **vue.config.js** - Vue build settings, PurgeCSS, code splitting
- **api/config/config.go** - Backend configuration

### Environment Variables
Create a `.env` file in the `api` directory:
```env
SERVER_PORT=8080
DATABASE_PATH=./data/database.db
ASSETS_BASE_DIR=./assets
```

## 🎨 Available Scripts

### Frontend
```bash
yarn serve       # Development server
yarn build       # Production build
yarn lint        # Lint and fix files
```

### Backend
```bash
go run ./cmd/server           # Development server
go build -o bin/app ./cmd/server  # Build binary
go test ./...                 # Run tests
```

## 📊 Performance Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Bundle Size | 6.5 MB | 1.4 MB | 78% smaller |
| Initial Load | 3.7s | 0.3s | 92% faster |
| Memory Usage | 284 MB | 47 MB | 83% less |
| API Response | 847 KB | 189 KB | 78% smaller |
| Scroll FPS | 28 fps | 60 fps | 114% smoother |

See [Performance Comparison](docs/PERFORMANCE_COMPARISON.md) for detailed metrics.

## 🤝 Contributing

When contributing, please:
1. Read the [Optimization Documentation](docs/README_OPTIMIZATIONS.md)
2. Follow existing performance patterns
3. Test with large datasets (1000+ videos)
4. Use performance monitoring tools during development

## 📝 License

This project is private and proprietary.

## 🆘 Support

- **Performance Issues?** Check [Performance Comparison](docs/PERFORMANCE_COMPARISON.md)
- **Integration Help?** See [Integration Guide](docs/OPTIMIZATION_INTEGRATION_GUIDE.md)
- **General Questions?** Review [Documentation Index](docs/INDEX.md)

---

**Built with ❤️ using Vue 3, Go, and extensive performance optimizations**
