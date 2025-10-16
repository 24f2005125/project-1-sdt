# Project 1-SDT

A Go-based AI project creator that will take a prompt and create a GitHub repository with the relevant files.

## 🚀 Features

- **RESTful API** with JSON request/response handling
- **Queue-based processing** with configurable workers
- **OpenAI integration** for task processing and content generation
- **GitHub API integration** for repository operations
- **Graceful shutdown** with proper signal handling
- **Environment-based configuration** with `.env` support

## 📋 Prerequisites

- Go 1.24.0 or higher
- Valid OpenAI API key
- Valid GitHub API key
- Environment variables configured (see Configuration section)

## 🛠️ Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/24f2005125/project-1-sdt.git
   cd project-1-sdt
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Set up environment variables:**
   Create a `.env` file in the root directory you can use the provided `.env.example` as a template.

4. **Build the application:**
   ```bash
   go build -o tmp/main .
   ```

## 🚀 Usage

### Starting the Server

```bash
go run .
```

The server will start on the port specified in your environment variables (default: 8765).

### API Endpoints

#### Health Check
```http
GET /
```

Returns server status and queue information:
```json
{
  "status": "running",
  "time": "17-10-2025 14:30:45",
  "queue": {
    "capacity": 100,
    "len": 0,
    "workers": 3
  }
}
```

#### Task Ingestion
```http
POST /ingest
Content-Type: application/json
```

Submit a task for processing:
```json
{
  "email": "user@example.com",
  "secret": "your_api_secret",
  "task": "task_identifier",
  "round": 1,
  "nonce": "unique_nonce",
  "brief": "Task description",
  "checks": ["check1", "check2"],
  "evaluation_url": "https://example.com/callback",
  "attachments": [
    {
      "name": "file.txt",
      "url": "data:text/plain;base64,SGVsbG8gd29ybGQh"
    }
  ]
}
```

## 🏗️ Architecture

### Core Components

- **HTTP Server** (`http_server.go`): Gin-based web server handling API requests
- **Queue System** (`queue.go`): Background job processing with configurable workers
- **OpenAI Integration** (`openai.go`): AI-powered task processing and content generation
- **GitHub Integration** (`git.go`): Repository operations and GitHub API interactions
- **HTTP Client** (`http_client.go`): Utility functions for external API calls
- **Utils** (`utils.go`): Helper functions and ASCII art generation

### Request Flow

1. Client submits task via `/ingest` endpoint
2. Request validation and authentication
3. Job queued for background processing
4. Worker processes job using OpenAI and GitHub APIs
5. Results sent to evaluation URL

## 🔧 Configuration

### Environment Variables


| Variable | Description | Required |
|----------|-------------|----------|
| `PORT` | Server port number | Yes |
| `OPENAI_KEY` | OpenAI API key | Yes |
| `GITHUB_KEY` | GitHub API token | Yes |
| `API_SECRET` | API authentication secret | Yes |
| `GITHUB_USER` | GitHub username for commits | Yes |
| `GITHUB_NAME` | GitHub name for commits | Yes |
| `GITHUB_EMAIL` | GitHub email for commits | Yes |

### Queue Configuration

- **Queue Size**: 100 jobs (configurable in `StartServer()`)
- **Workers**: 3 concurrent workers (configurable in `StartServer()`)
- **Timeout**: 200ms enqueue timeout

## 📦 Dependencies

Key dependencies include:

- **Gin**: Web framework for HTTP server
- **OpenAI Go SDK**: OpenAI API integration
- **Godotenv**: Environment variable management
- **UUID**: Unique identifier generation
- **Go-Figure**: ASCII art generation

See `go.mod` for the complete list of dependencies.

## 🧪 Development

### Project Structure

```
.
├── main.go              # Application entry point
├── http_server.go       # Web server and API routes
├── queue.go            # Background job processing
├── openai.go           # OpenAI integration
├── git.go              # GitHub API integration
├── http_client.go      # HTTP utility functions
├── utils.go            # Helper utilities
├── go.mod              # Go module definition
├── scripts/            # Helper scripts and test data
│   ├── ingest.sh       # Request submission script
│   ├── data.csv        # Test data
│   └── request-*.json  # Sample requests
└── tmp/                # Build artifacts
```

### Building for Production

```bash
go build -ldflags="-s -w" -o bin/project-1-sdt .
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👤 Author

**Hayzam Sherif** - [GitHub](https://github.com/hayzamjs)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📞 Support

For support or questions, please contact the author or open an issue in the repository.