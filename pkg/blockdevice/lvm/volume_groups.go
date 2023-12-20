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
	"strings"

	"github.com/NearNodeFlash/nnf-sos/pkg/command"
	"github.com/NearNodeFlash/nnf-sos/pkg/var_handler"
	"github.com/go-logr/logr"
)

type VolumeGroup struct {
	Name            string
	PhysicalVolumes []*PhysicalVolume
	Shared          bool

	Log logr.Logger
}

func NewVolumeGroup(ctx context.Context, name string, pvs []*PhysicalVolume, log logr.Logger) *VolumeGroup {
	return &VolumeGroup{
		Name:            name,
		PhysicalVolumes: pvs,
		Log:             log,
	}
}

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

func (vg *VolumeGroup) parseArgs(args string) (string, error) {
	deviceNames := []string{}
	for _, pv := range vg.PhysicalVolumes {
		deviceNames = append(deviceNames, pv.Device)
	}

	// Initialize the VarHandler substitution variables
	varHandler := var_handler.NewVarHandler(map[string]string{
		"$DEVICE_NUM":  fmt.Sprintf("%d", len(deviceNames)),
		"$DEVICE_LIST": strings.Join(deviceNames, " "),
		"$VG_NAME":     vg.Name,
	})

	if err := varHandler.ListToVars("$DEVICE_LIST", "$DEVICE"); err != nil {
		return "", fmt.Errorf("invalid internal device list: %w", err)
	}

	return varHandler.ReplaceAll(args), nil
}

func (vg *VolumeGroup) Create(ctx context.Context, rawArgs string) (bool, error) {
	args, err := vg.parseArgs(rawArgs)
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
	args, err := vg.parseArgs(rawArgs)
	if err != nil {
		return false, err
	}

	if _, err := command.Run(fmt.Sprintf("vgchange %s", args), vg.Log); err != nil {
		return false, err
	}

	return true, nil
}

func (vg *VolumeGroup) LockStart(ctx context.Context, rawArgs string) (bool, error) {
	return vg.Change(ctx, rawArgs)
}

func (vg *VolumeGroup) LockStop(ctx context.Context, rawArgs string) (bool, error) {
	exists, err := vg.Exists(ctx)
	if err != nil {
		return false, err
	}

	if exists == false {
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
	args, err := vg.parseArgs(rawArgs)
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
