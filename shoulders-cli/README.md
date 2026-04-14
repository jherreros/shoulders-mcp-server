# shoulders-cli

Developer CLI for the Shoulders Internal Developer Platform. The bootstrap flow uses Go-native APIs (vCluster library + Helm SDK) rather than shelling out to scripts.

## Install

With Homebrew:

```bash
brew install jherreros/tap/shoulders
```

Or via the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/jherreros/shoulders/main/scripts/install.sh | bash
```

## Build

```bash
cd shoulders-cli
go mod tidy
go build -o shoulders
```

## Usage

### Cluster Management
```bash
./shoulders up                        # Create and bootstrap the platform (default name: shoulders)
./shoulders up --verbose              # Same, with detailed per-phase progress
./shoulders cluster list              # List running clusters
./shoulders cluster use dev           # Switch context to 'dev' cluster
./shoulders down --name dev           # Delete the cluster
./shoulders update                    # Check for and install a new CLI version
```

### Workspace Management
```bash
./shoulders workspace create team-a
./shoulders workspace list
./shoulders workspace use team-a
./shoulders workspace current
./shoulders workspace delete team-a
```

### Application Lifecycle
```bash
./shoulders app init hello --image nginx:1.26 --replicas 1
./shoulders app list
./shoulders app describe hello
./shoulders logs hello
./shoulders app delete hello
```

### Infrastructure
```bash
./shoulders infra add-db app-db --type postgres --tier dev
./shoulders infra add-stream events --topics "logs,events" --partitions 3 --replicas 3 \
	--config cleanup.policy=compact
./shoulders infra list
./shoulders infra delete app-db
```

### Platform
```bash
./shoulders status                    # Show cluster & platform health
./shoulders status --wait             # Poll until all components are healthy
./shoulders dashboard                 # Opens grafana.localhost (falls back to localhost:3000)
./shoulders portal                    # Opens Headlamp portal (falls back to localhost:4466)
./shoulders reporter                  # Opens Policy Reporter UI (falls back to localhost:8082)
./shoulders skill install             # Install the Shoulders agent skill for AI assistants
./shoulders skill install --workspace # Install into the current project instead of globally
```

`*.localhost` access requires host port `80` to be available when the cluster is created. If you changed these settings, recreate the cluster:

```bash
./shoulders down
./shoulders up
```

## Configuration
The current workspace context is stored at `~/.shoulders/config.yaml`.

## Output formats
Use `-o table|json|yaml` for supported list and status commands.

## Notes
- `shoulders app init` supports `--dry-run` to emit YAML instead of applying it.
- `shoulders logs` attempts a Loki query first and falls back to direct pod log streaming (no `kubectl`).
- `shoulders up` provisions the cluster via the vCluster Go library (vind/Docker driver) and installs Cilium + Flux without running shell scripts. It pulls the Cilium chart and Flux install manifest from their upstream URLs.
- `shoulders up --verbose` shows detailed descriptions for each bootstrap phase.
- `shoulders up` displays a live timer, per-phase durations, and a final summary (e.g. "Shoulders platform provisioned in 04:32").
- `shoulders reporter` opens the Policy Reporter UI at `reporter.localhost`, falling back to a local port-forward on port 8082.
- `shoulders infra add-stream` supports `--partitions`, `--replicas`, and repeatable `--config key=value` entries.
- `shoulders up` and `down` support `--name` to create/delete specifically named clusters.
- `shoulders status --wait` polls every 3 seconds and refreshes the TUI display until all components are healthy.
- `shoulders update` checks the latest GitHub release and self-updates the binary.
- Commands that interact with the cluster (status, down, workspace, app, etc.) verify that the current kubeconfig context is a Shoulders-managed vind cluster. Use `shoulders cluster use <name>` to switch contexts.
