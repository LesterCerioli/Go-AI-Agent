# ⚡ Go AI Agent

> AI-powered backend development assistant built with **Go 1.24+** and **Fiber v2**  
> Runs on Windows 11 · Integrates with VS Code

---

## 🏗️ Project Structure

```
go-ai-agent/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── agent/
│   │   ├── service.go       # AI agent logic + session management
│   │   └── service_test.go  # Unit tests
│   ├── config/
│   │   └── config.go        # Environment-based config
│   ├── handlers/
│   │   └── handlers.go      # Fiber HTTP handlers
│   ├── middleware/
│   │   └── middleware.go    # Error handler, request ID
│   └── models/
│       └── models.go        # Request/response types
├── api/
│   └── requests.http        # VS Code REST Client tests
├── web/
│   └── index.html           # Frontend UI
├── .vscode/
│   ├── settings.json        # Go + editor settings
│   ├── launch.json          # Debug configurations
│   ├── tasks.json           # Build/run/test tasks
│   └── extensions.json      # Recommended extensions
├── .env.example             # Environment template
├── .air.toml                # Hot-reload config
├── Makefile                 # Build automation
├── setup.bat                # Windows 11 setup script
└── go.mod
```

---

## 🚀 Quick Start (Windows 11)

### 1. Prerequisites

- [Go 1.24+](https://go.dev/dl/) — download the Windows installer
- [VS Code](https://code.visualstudio.com/) — with the Go extension
- An [Anthropic API key](https://console.anthropic.com/)

### 2. Setup

Double-click `setup.bat` or run in PowerShell:

```powershell
# Clone or extract the project
cd go-ai-agent

# Run setup (installs deps, creates .env, builds)
.\setup.bat
```

### 3. Configure

Edit `.env`:
```env
ANTHROPIC_API_KEY=sk-ant-...your-key-here...
PORT=8080
DEFAULT_MODEL=claude-sonnet-4-20250514
```

### 4. Run

```powershell
# Option A: Direct run
go run .\cmd\server\main.go

# Option B: Run binary
.\build\go-ai-agent.exe

# Option C: Hot-reload during development (requires air)
air -c .air.toml

# Option D: VS Code task
# Ctrl+Shift+P → "Tasks: Run Task" → "run"
```

Open browser: **http://localhost:8080**

---

## 🔌 API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/health` | Server health + API status |
| `GET` | `/api/v1/models` | List available Claude models |
| `GET` | `/api/v1/tools` | List available agent tools |
| `POST` | `/api/v1/agent/chat` | Send a message (blocking) |
| `POST` | `/api/v1/agent/stream` | Send a message (SSE streaming) |
| `GET` | `/api/v1/agent/sessions` | List all sessions |
| `GET` | `/api/v1/agent/sessions/:id` | Get session with history |
| `DELETE` | `/api/v1/agent/sessions/:id` | Delete a session |
| `POST` | `/api/v1/tools/execute` | Execute a built-in tool |

### Chat Request

```json
POST /api/v1/agent/chat
{
  "message": "Write a Fiber middleware for JWT auth",
  "session_id": "optional-existing-session-id",
  "model": "claude-sonnet-4-20250514",
  "system_prompt": "You are a Go expert. Give idiomatic code.",
  "max_tokens": 2048,
  "temperature": 0.7
}
```

### Chat Response

```json
{
  "session_id": "abc-123",
  "message_id": "msg-456",
  "content": "Here's a JWT middleware for Fiber...",
  "model": "claude-sonnet-4-20250514",
  "tokens": 342,
  "duration_ms": 1240
}
```

### Streaming (SSE)

```json
POST /api/v1/agent/stream
// Returns server-sent events:
data: {"type":"delta","content":"Here"}
data: {"type":"delta","content":"'s a"}
data: {"done":true}
```

---

## 🖥️ VS Code Integration

### Debug (F5)
Select **"Launch AI Agent Server"** from the debug panel.

### Tasks (Ctrl+Shift+B)
- **build** — compile the binary
- **run** — start the server  
- **test** — run all tests
- **lint** — run golangci-lint
- **build-windows** — compile for Windows

### REST Client
Open `api/requests.http` and click **"Send Request"** above any request.

### Recommended Extensions
Accept the prompt to install all recommended extensions from `.vscode/extensions.json`.

---

## 🧪 Running Tests

```powershell
# All tests
go test ./...

# With race detection
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 🔧 Configuration Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | *(required)* | Your Anthropic API key |
| `PORT` | `8080` | Server port |
| `HOST` | `localhost` | Bind address |
| `DEFAULT_MODEL` | `claude-sonnet-4-20250514` | Default Claude model |
| `MAX_TOKENS` | `8192` | Max response tokens |
| `READ_TIMEOUT` | `60s` | HTTP read timeout |
| `WRITE_TIMEOUT` | `60s` | HTTP write timeout |
| `MAX_SESSION_AGE` | `24h` | Session TTL |
| `MAX_CONVERSATIONS` | `100` | Max in-memory sessions |

---

## 🏗️ Extending the Agent

### Add a new tool

In `internal/handlers/handlers.go`:

```go
// Add to builtinTools()
{
  Name:        "my_tool",
  Description: "Does something useful",
  Category:    "development",
  Parameters: map[string]interface{}{
    "input": map[string]interface{}{"type": "string"},
  },
},

// Add to executeTool()
case "my_tool":
  input := params["input"].(string)
  return map[string]interface{}{"result": process(input)}, nil
```

### Add a new route

In `cmd/server/main.go`:

```go
myHandler := handlers.NewMyHandler(cfg)
api.Get("/my-endpoint", myHandler.Handle)
```

---

## 📦 Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24+ |
| HTTP Framework | [Fiber v2](https://gofiber.io/) |
| AI API | [Anthropic Claude](https://console.anthropic.com/) |
| Config | `godotenv` |
| IDs | `google/uuid` |
| Dev | Air (hot-reload), golangci-lint |

---

## 🪟 Windows 11 Notes

- All paths use `\` in `setup.bat` but Go handles `/` on all platforms
- The binary output is `build\go-ai-agent.exe`
- Add `%GOPATH%\bin` to PATH for dev tools (goimports, air, etc.)
- Windows Defender may flag new `.exe` files — add an exception for the `build\` folder