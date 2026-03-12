# Fence Recorder Configuration

This directory contains shared configuration for fence recorder paths used by both the nnf-sos controller and the fence-recorder agent.

## Files

- **config.go** - Go package with fence directory paths for nnf-sos (reads env vars at init)
- **config.py** - Python module with fence directory paths for fence-agents (reads env vars at import)
- **config_test.go** - Unit tests for config.go env var handling

## Usage

### In nnf-sos (Go)

```go
import "github.com/NearNodeFlash/nnf-sos/pkg/fence"

// Use the directory paths (populated from env vars or defaults at init)
requestDir := fence.RequestDir
responseDir := fence.ResponseDir
```

### In fence-agents (Python)

Copy `config.py` to your fence-agents repository and import it:

```python
# If installed in fence-agents/agents/lib/
from lib.fence.config import REQUEST_DIR, RESPONSE_DIR

# Or add the nnf-sos path to sys.path
import sys
sys.path.insert(0, '/path/to/nnf-sos/pkg/fence')
import config
REQUEST_DIR = config.REQUEST_DIR
RESPONSE_DIR = config.RESPONSE_DIR
```

## Configuring Paths

The directory paths default to `/localdisk/fence-recorder/requests` and `/localdisk/fence-recorder/responses`. To override them, set environment variables **before** the process starts:

```bash
export NNF_FENCE_REQUEST_DIR=/my/custom/requests
export NNF_FENCE_RESPONSE_DIR=/my/custom/responses
```

Both `config.go` (via `init()`) and `config.py` (via `os.environ.get()`) read these variables automatically, falling back to the defaults when unset.

In Kubernetes, set these in the pod spec for **both** the nnf-node-manager and fence-agent pods:

```yaml
env:
  - name: NNF_FENCE_REQUEST_DIR
    value: "/my/custom/requests"
  - name: NNF_FENCE_RESPONSE_DIR
    value: "/my/custom/responses"
```

### Changing the Defaults

If you need to change the compiled-in default paths:

1. Update `DefaultRequestDir` / `DefaultResponseDir` in `config.go`
2. Update `_DEFAULT_REQUEST_DIR` / `_DEFAULT_RESPONSE_DIR` in `config.py`
3. Ensure both repositories are updated together

## Directory Structure

The fencing protocol uses two directories:

- **Request Directory** (`/localdisk/fence-recorder/requests`): Where fence agents write JSON request files
- **Response Directory** (`/localdisk/fence-recorder/responses`): Where nnf-sos writes JSON response files

## Alternative Approaches

If environment variables are insufficient for your use case:

### System Configuration File

Create `/etc/nnf/fence-config.json`:

```json
{
  "request_dir": "/localdisk/fence-recorder/requests",
  "response_dir": "/localdisk/fence-recorder/responses"
}
```

Both Go and Python code could read this file at runtime. This would require adding a `LoadConfig()` function.

## Current Implementation

The nnf-sos controller uses `fence.RequestDir` and `fence.ResponseDir` from `config.go` in:

- `internal/controller/nnf_node_controller.go` - NnfNode controller watches both request and response directories using fsnotify
- `internal/controller/nnf_node_block_storage_controller.go` - Processes fence requests and writes response files

The fence agent should import `config.py` to use the same paths.

## Environment Variables Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `NNF_FENCE_REQUEST_DIR` | `/localdisk/fence-recorder/requests` | Directory where fence agents write JSON request files |
| `NNF_FENCE_RESPONSE_DIR` | `/localdisk/fence-recorder/responses` | Directory where nnf-sos writes JSON response files |
