# OpenSandbox: General-Purpose AI Sandbox Platform

OpenSandbox is a universal sandbox platform designed for AI applications. It provides multi-language SDKs, unified sandbox APIs, and flexible runtimes (Docker/Kubernetes) to safely execute AI-generated code, automate browsers, and run training workloads.

## Project Architecture & Core Components

OpenSandbox follows a layered architecture:
- **SDKs Layer (`sdks/`)**: Client libraries in Python, Java/Kotlin, JavaScript/TypeScript, and Go.
- **Specs Layer (`specs/`)**: OpenAPI definitions for Lifecycle (sandbox management) and Execution (`execd` agent).
- **Runtime Layer (`server/`)**: FastAPI-based control plane managing the sandbox lifecycle on Docker or Kubernetes.
- **Components (`components/`)**:
    - `execd`: Go-based execution daemon injected into sandboxes for command and file operations.
    - `ingress/egress`: Traffic control and routing components.
- **Sandboxes (`sandboxes/`)**: Pre-built runtime environments like Code Interpreter and Browser (CDP).

For a detailed deep dive, refer to `docs/architecture.md`.

## Tech Stack
- **Server**: Python 3.10+, FastAPI, Docker SDK, Kubernetes SDK.
- **Execution Daemon (execd)**: Go 1.24+, Beego, Jupyter Kernel Protocol.
- **CLI**: Python, Click, Rich.
- **SDKs**: Python (uv), Java/Kotlin (Gradle), JS/TS (pnpm/TS).
- **Documentation**: VitePress (located in `docs/`).

## Key Workflows

### Server Development (Python)
The server is the central control plane.
- **Setup**: `cd server && uv sync`
- **Configuration**: Uses TOML (default: `~/.sandbox.toml`).
- **Run**: `uv run python -m opensandbox_server.main`
- **Lint/Format**: `uv run ruff check .` and `uv run ruff format .`
- **Test**: `uv run pytest`

### Execution Daemon Development (Go)
`execd` runs inside the sandbox container.
- **Build**: `cd components/execd && go build -o bin/execd .`
- **Test**: `go test ./pkg/...`

### Python SDK Development
- **Setup**: `cd sdks/sandbox/python && uv sync`
- **Test**: `uv run pytest`

### CLI Development
- **Setup**: `cd cli && uv sync`
- **Run**: `uv run opensandbox --help` or `uv run osb --help`

## Development Conventions & Standards

- **Conventional Commits**: Use `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`.
- **Branching**: `main` for stable, `feature/*` for new features, `fix/*` for bugs.
- **Code Style**:
    - **Python**: PEP 8, ruff for linting/formatting, mandatory type hints.
    - **Go**: `gofmt` and Effective Go standards.
    - **Kotlin**: Kotlin Coding Conventions and `ktlint`.
- **License**: Apache 2.0. Every file must include the license header (use `scripts/add-license.sh`).
- **Testing**: E2E tests are located in `tests/`. New features must include unit and integration tests.
- **OSEP**: Major changes require an **OpenSandbox Enhancement Proposal** (see `oseps/README.md`).

## Building Sandboxes
Sandboxes are Docker images. Check `sandboxes/` for Dockerfiles.
Example: `sandboxes/browser-cdp/Dockerfile` builds a Chromium environment with `execd` injected.

## Key Files for Reference
- `README.md`: Project overview and quick start.
- `CONTRIBUTING.md`: Detailed contribution and setup guide.
- `docs/architecture.md`: In-depth system design.
- `specs/sandbox-lifecycle.yml`: Lifecycle API definition.
- `specs/execd-api.yaml`: Execution API (execd) definition.
- `DEPLOY.md`: Deployment instructions.
