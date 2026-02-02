/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
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
	"github.com/NearNodeFlash/nnf-sos/pkg/var_handler"
	"github.com/go-logr/logr"
)

type PhysicalVolume struct {
	Device string

	Vars map[string]string

	Log logr.Logger
}

func NewPhysicalVolume(ctx context.Context, device string, log logr.Logger) *PhysicalVolume {
	return &PhysicalVolume{
		Device: device,
		Log:    log,
	}
}

func (pv *PhysicalVolume) parseArgs(args string, device string) (string, error) {

	// Initialize the VarHandler substitution variables
	varHandler := var_handler.NewVarHandler(map[string]string{
		"$DEVICE": device,
	})

	for key, value := range pv.Vars {
		varHandler.AddVar(key, value)
	}

	return varHandler.ReplaceAll(args), nil
}

func (pv *PhysicalVolume) Create(ctx context.Context, rawArgs string) (bool, error) {
	if len(rawArgs) == 0 {
		return false, nil
	}

	args, err := pv.parseArgs(rawArgs, pv.Device)
	if err != nil {
		return false, err
	}

	existingPVs, err := pvsListVolumes(ctx, pv.Log)
	if err != nil {
		return false, err
	}

	for _, existingPV := range existingPVs {
		if existingPV.Name == pv.Device {
			return false, nil
		}
	}

	// No existing LVM PV found. Create one
	if _, err := command.Run(fmt.Sprintf("pvcreate %s", args), pv.Log); err != nil {
		if err != nil {
			return false, fmt.Errorf("could not create LVM physical volume: %w", err)
		}
	}

	return true, nil
}

func (pv *PhysicalVolume) Remove(ctx context.Context, rawArgs string) (bool, error) {
	if len(rawArgs) == 0 {
		return false, nil
	}

	args, err := pv.parseArgs(rawArgs, pv.Device)
	if err != nil {
		return false, err
	}

	existingPVs, err := pvsListVolumes(ctx, pv.Log)
	if err != nil {
		return false, err
	}

	for _, existingPV := range existingPVs {
		if existingPV.Name == pv.Device {
			// LVM PV found. Delete it
			if _, err := command.Run(fmt.Sprintf("pvremove --yes %s", args), pv.Log); err != nil {
				if err != nil {
					return false, fmt.Errorf("could not destroy LVM physical volume: %w", err)
				}
			}

			return true, nil
		}
	}

	return false, nil
}
