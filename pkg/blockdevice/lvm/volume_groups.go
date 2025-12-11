/*
 * Copyright 2023-2024 Hewlett Packard Enterprise Development LP
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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/NearNodeFlash/nnf-sos/pkg/command"
	"github.com/NearNodeFlash/nnf-sos/pkg/var_handler"
	"github.com/go-logr/logr"
)

type VolumeGroup struct {
	Name            string
	PhysicalVolumes []*PhysicalVolume
	Shared          bool

	Vars map[string]string

	Log logr.Logger
}

// NewVolumeGroup returns a VolumeGroup for operations.
func NewVolumeGroup(ctx context.Context, name string, pvs []*PhysicalVolume, log logr.Logger) *VolumeGroup {
	return &VolumeGroup{
		Name:            name,
		PhysicalVolumes: pvs,
		Log:             log,
	}
}

// Exists determines if the VG exists in the OS
func (vg *VolumeGroup) Exists(ctx context.Context) (bool, error) {
	existingVGs, err := vgsListVolumes(ctx, vg.Log)
	if err != nil {
		return false, err
	}

	for _, existingVG := range existingVGs {
		if existingVG.Name == vg.Name {
			return true, nil
		}
	}

	return false, nil
}

// WaitForAppearance checks the existence of the VG and
// waits a brief time for the VG to be created if it is not present.
// This intentionally fails quickly to trigger an NVMe namespace rescan,
// relying on the controller's exponential backoff for retries.
func (vg *VolumeGroup) WaitForAppearance(ctx context.Context) (bool, error) {

	// Default to 1 second timeout - fail quickly to trigger ns-rescan
	retryPeriod := 1 * time.Second

	// Look for environment variable to override
	timeoutString, found := os.LookupEnv("NNF_MAPPER_WAIT_TIMEOUT")
	if found {
		timeout, err := strconv.Atoi(timeoutString)
		if err == nil && timeout > 0 {
			retryPeriod = time.Duration(timeout) * time.Second
		}
	}

	// Give the VG time to appear
	var exists bool
	var err error
	for start := time.Now(); time.Since(start) < retryPeriod; time.Sleep(100 * time.Millisecond) {
		exists, err = vg.Exists(ctx)
		if err != nil {
			return false, err
		}

		if exists {
			return true, nil
		}
	}

	return false, fmt.Errorf("timeout waiting for VG")
}

func (vg *VolumeGroup) parseArgs(args string, vars map[string]string) (string, error) {
	deviceNames := []string{}
	for _, pv := range vg.PhysicalVolumes {
		deviceNames = append(deviceNames, pv.Device)
	}

	// Initialize the VarHandler substitution variables
	varHandler := var_handler.NewVarHandler(map[string]string{
		"$DEVICE_NUM": fmt.Sprintf("%d", len(deviceNames)),
		// These are added in case any VG commands need the count of data devices
		// for a RAID array
		"$DEVICE_NUM-1": fmt.Sprintf("%d", len(deviceNames)-1),
		"$DEVICE_NUM-2": fmt.Sprintf("%d", len(deviceNames)-2),
		"$DEVICE_LIST":  strings.Join(deviceNames, " "),
		"$VG_NAME":      vg.Name,
	})

	if err := varHandler.ListToVars("$DEVICE_LIST", "$DEVICE"); err != nil {
		return "", fmt.Errorf("invalid internal device list: %w", err)
	}

	for key, value := range vg.Vars {
		varHandler.AddVar(key, value)
	}

	for key, value := range vars {
		varHandler.AddVar(key, value)
	}

	return varHandler.ReplaceAll(args), nil
}

func (vg *VolumeGroup) Create(ctx context.Context, rawArgs string) (bool, error) {
	args, err := vg.parseArgs(rawArgs, nil)
	if err != nil {
		return false, err
	}

	existingVGs, err := vgsListVolumes(ctx, vg.Log)
	if err != nil {
		return false, err
	}

	for _, existingVG := range existingVGs {
		if existingVG.Name == vg.Name {
			return false, nil
		}
	}

	if _, err := command.Run(fmt.Sprintf("vgcreate %s", args), vg.Log); err != nil {
		return false, fmt.Errorf("could not create volume group: %w", err)
	}

	return true, nil
}

func (vg *VolumeGroup) Change(ctx context.Context, rawArgs string) (bool, error) {
	args, err := vg.parseArgs(rawArgs, nil)
	if err != nil {
		return false, err
	}

	if _, err := command.Run(fmt.Sprintf("vgchange %s", args), vg.Log); err != nil {
		return false, err
	}

	return true, nil
}

func (vg *VolumeGroup) LockStart(ctx context.Context, rawArgs string) (bool, error) {
	exists, err := vg.WaitForAppearance(ctx)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	return vg.Change(ctx, rawArgs)
}

func (vg *VolumeGroup) LockStop(ctx context.Context, rawArgs string) (bool, error) {
	exists, err := vg.Exists(ctx)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	lvs, err := lvsListVolumes(ctx, vg.Log)
	for _, lv := range lvs {
		if lv.VGName == vg.Name && lv.Attrs[4] == 'a' {
			return false, nil
		}
	}

	return vg.Change(ctx, rawArgs)
}

func (vg *VolumeGroup) Remove(ctx context.Context, rawArgs string) (bool, error) {
	args, err := vg.parseArgs(rawArgs, nil)
	if err != nil {
		return false, err
	}

	existingVGs, err := vgsListVolumes(ctx, vg.Log)
	if err != nil {
		return false, err
	}

	for _, existingVG := range existingVGs {
		if existingVG.Name == vg.Name {
			if _, err := command.Run(fmt.Sprintf("vgremove --yes %s", args), vg.Log); err != nil {
				return false, fmt.Errorf("could not destroy volume group: %w", err)
			}

			return true, nil
		}
	}

	return false, nil
}

func (vg *VolumeGroup) Extend(ctx context.Context, rawArgs string) (bool, error) {
	existingPVs, err := pvsListVolumes(ctx, vg.Log)
	if err != nil {
		return false, err
	}

	// Check check whether all the PVs used in the VG are currently added. Extend the VG if necessary
	extended := false
	for _, desiredPV := range vg.PhysicalVolumes {
		for _, existingPV := range existingPVs {
			if existingPV.Name != desiredPV.Device {
				continue
			}

			if existingPV.VGName != "" {
				if existingPV.VGName != vg.Name {
					return false, fmt.Errorf("physical volume: %s attached to incorrect volume group: %s. Expected: %s", existingPV.Name, existingPV.VGName, vg.Name)
				}

				continue
			}

			args, err := vg.parseArgs(rawArgs, map[string]string{"$DEVICE": desiredPV.Device})
			if err != nil {
				return false, err
			}

			if _, err := command.Run(fmt.Sprintf("vgextend %s", args), vg.Log); err != nil {
				return false, fmt.Errorf("could not extend volume group: %w", err)
			}

			extended = true
		}
	}

	return extended, nil
}

func (vg *VolumeGroup) Reduce(ctx context.Context, rawArgs string) (bool, error) {
	args, err := vg.parseArgs(rawArgs, nil)
	if err != nil {
		return false, err
	}

	// remove any missing PVs from the VG
	if _, err := command.Run(fmt.Sprintf("vgreduce %s", args), vg.Log); err != nil {
		return false, fmt.Errorf("could not reduce volume group: %w", err)
	}

	return true, nil
}

func (vg *VolumeGroup) NumLVs(ctx context.Context) (int, error) {
	count := 0

	lvs, err := lvsListVolumes(ctx, vg.Log)
	if err != nil {
		return count, err
	}
	for _, lv := range lvs {
		if lv.VGName == vg.Name {
			count += 1
		}
	}

	return count, nil
}
