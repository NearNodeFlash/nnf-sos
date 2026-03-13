/*
 * Copyright 2025 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fence

import "os"

// Default fence recorder directory paths.
const (
	DefaultRequestDir  = "/localdisk/fence-recorder/requests"
	DefaultResponseDir = "/localdisk/fence-recorder/responses"
)

// Environment variable names for overriding the default paths.
const (
	EnvRequestDir  = "NNF_FENCE_REQUEST_DIR"
	EnvResponseDir = "NNF_FENCE_RESPONSE_DIR"
)

// RequestDir is where fence agents write fence request files.
// Override with NNF_FENCE_REQUEST_DIR environment variable.
var RequestDir = DefaultRequestDir

// ResponseDir is where nnf-sos writes fence response files.
// Override with NNF_FENCE_RESPONSE_DIR environment variable.
var ResponseDir = DefaultResponseDir

func init() {
	if v := os.Getenv(EnvRequestDir); v != "" {
		RequestDir = v
	}
	if v := os.Getenv(EnvResponseDir); v != "" {
		ResponseDir = v
	}
}
