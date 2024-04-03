/*
 * Copyright 2024 Hewlett Packard Enterprise Development LP
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

package lvm

import (
	"context"
	"fmt"

	"github.com/NearNodeFlash/nnf-sos/pkg/command"
	"github.com/go-logr/logr"
)

func dlmLockSpaceExists(ctx context.Context, vgName string, log logr.Logger) (bool, error) {
	output, err := command.Run(fmt.Sprintf("dlm_tool ls | grep %s", vgName), log)
	if err != nil {
		return false, fmt.Errorf("could not list dlm lockspaces: %w", err)
	}

	if len(output) == 0 {
		return false, nil
	} else {
		return true, nil
	}
}
