# Contributing to Shoulders

## Prerequisites

- **Docker** — Required for vind (vCluster in Docker)
- **Go** 1.25+ — CLI development (`shoulders-cli/`)
- **Node.js** 20+ — MCP server and portal plugin
- **kubectl** — Kubernetes interaction

## Repository Structure

| Directory | Language | Description |
|-----------|----------|-------------|
| `1-cluster/` | Shell/YAML | Cluster provisioning (vind) |
| `2-addons/` | YAML | Platform components (FluxCD, Crossplane, Helm) |
| `3-user-space/` | YAML | Example team resources |
| `shoulders-cli/` | Go | CLI tool |
| `shoulders-mcp-server/` | TypeScript | MCP server |
| `shoulders-portal-plugin/` | TypeScript/React | Headlamp plugin |

## Building

### CLI

```bash
cd shoulders-cli
go mod tidy
go generate ./...
go build -o shoulders
```

### MCP Server

```bash
cd shoulders-mcp-server
npm install
npm run build
```

### Portal Plugin

```bash
cd shoulders-portal-plugin
npm install
npm run build
```

## Testing

### CLI

```bash
cd shoulders-cli
go test -v ./...
```

Lint:

```bash
golangci-lint run --timeout 5m
```

### MCP Server

```bash
cd shoulders-mcp-server
npm test
```

### Portal Plugin

```bash
cd shoulders-portal-plugin
npm test
npx headlamp-plugin tsc   # type check
npm run lint
```

### Integration

Test your changes end-to-end with a fresh cluster:

```bash
shoulders down
shoulders up
shoulders status --wait
```

## Pull Request Guidelines

1. Fork the repository and create a feature branch from `main`.
2. Follow existing code style — the CI enforces `golangci-lint` for Go and ESLint for TypeScript.
3. Add or update tests when changing behavior.
4. Use [conventional commit](https://www.conventionalcommits.org/) messages (e.g., `feat:`, `fix:`, `chore:`).
5. Keep PRs focused — one logical change per PR.

## Crossplane Development

When adding or modifying platform abstractions:

1. Define the XRD in `2-addons/manifests/crossplane/definitions/`.
2. Implement the composition in `2-addons/manifests/crossplane/compositions/`.
3. Add RBAC permissions in `2-addons/manifests/crossplane/rbac/crossplane-composed-resources.yaml`.
4. Add a canonical example in `3-user-space/team-a/`.

See [`.github/copilot-instructions.md`](.github/copilot-instructions.md) for Crossplane composition patterns and conventions.

## AI-Assisted Contributions

This repository includes Copilot workspace instructions (`.github/copilot-instructions.md`) and an agent skill (`.github/skills/shoulders/`) that provide context about the project's architecture and conventions. AI agents working in this codebase will automatically pick up these instructions.

## Releases

Releases are automated via GoReleaser on `v*` tags. The release workflow:

1. Builds CLI binaries for Linux/macOS/Windows (amd64/arm64).
2. Updates the Homebrew formula in `jherreros/homebrew-tap`.
3. Builds and publishes the Headlamp portal plugin.
4. Updates Artifact Hub metadata.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
