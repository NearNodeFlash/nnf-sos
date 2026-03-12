#!/usr/bin/env python3
# Copyright 2025-2026 Hewlett Packard Enterprise Development LP
# Other additional copyright holders may be indicated within.
#
# The entirety of this work is licensed under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
#
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Fence Recorder Configuration

Override with NNF_FENCE_REQUEST_DIR and NNF_FENCE_RESPONSE_DIR
environment variables.
"""

import os

# Default paths
_DEFAULT_REQUEST_DIR = "/localdisk/fence-recorder/requests"
_DEFAULT_RESPONSE_DIR = "/localdisk/fence-recorder/responses"

# Directory where fence agents write fence request files
REQUEST_DIR = os.environ.get("NNF_FENCE_REQUEST_DIR", _DEFAULT_REQUEST_DIR)

# Directory where nnf-sos writes fence response files
RESPONSE_DIR = os.environ.get("NNF_FENCE_RESPONSE_DIR", _DEFAULT_RESPONSE_DIR)
