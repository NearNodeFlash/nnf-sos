# GFS2 Fence Configuration

This directory contains shared configuration for GFS2 fencing paths used by both the nnf-sos controller and the fence-agents.

## Files

- **config.go** - Go package with fence directory constants for nnf-sos
- **config.py** - Python module with fence directory constants for fence-agents

## Usage

### In nnf-sos (Go)

```go
import "github.com/NearNodeFlash/nnf-sos/pkg/fence"

// Use the shared constants
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

## Updating Paths

When you need to change the fence request/response directory paths:

1. Update the constants in **both** `config.go` and `config.py`
2. Ensure both repositories are updated together
3. Test both the nnf-sos controller and fence agents with the new paths

## Directory Structure

The fencing protocol uses two directories:

- **Request Directory** (`/localdisk/gfs2-fencing/requests`): Where fence agents write JSON request files
- **Response Directory** (`/localdisk/gfs2-fencing/responses`): Where nnf-sos writes JSON response files

## Alternative Approaches

If you prefer a different approach to sharing configuration:

### Option 1: Environment Variables

Set environment variables on both systems:

```bash
export GFS2_FENCE_REQUEST_DIR=/localdisk/gfs2-fencing/requests
export GFS2_FENCE_RESPONSE_DIR=/localdisk/gfs2-fencing/responses
```

### Option 2: System Configuration File

Create `/etc/nnf/fence-config.json`:

```json
{
  "request_dir": "/localdisk/gfs2-fencing/requests",
  "response_dir": "/localdisk/gfs2-fencing/responses"
}
```

Both Go and Python code can read this file at runtime.

### Option 3: Git Submodule

Make this configuration directory a separate git repository and include it as a submodule in both nnf-sos and fence-agents.

## Current Implementation

The nnf-sos controller currently uses the constants from `config.go` in:

- `internal/controller/gfs2_fence_watcher.go` - File system watcher
- `internal/controller/nnf_node_storage_controller.go` - Response file writer

The fence agents should import `config.py` to use the same paths.
