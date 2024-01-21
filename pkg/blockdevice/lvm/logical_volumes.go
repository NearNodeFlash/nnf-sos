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

type LogicalVolume struct {
	Name        string
	Size        int64
	PercentVG   int
	VolumeGroup *VolumeGroup

	Log logr.Logger
}

func NewLogicalVolume(ctx context.Context, name string, vg *VolumeGroup, size int64, percentVG int, log logr.Logger) *LogicalVolume {
	return &LogicalVolume{
		Name:        name,
		VolumeGroup: vg,
		Size:        size,
		Log:         log,
		PercentVG:   percentVG,
	}
}

func (lv *LogicalVolume) parseArgs(args string) (string, error) {
	deviceNames := []string{}
	for _, pv := range lv.VolumeGroup.PhysicalVolumes {
		deviceNames = append(deviceNames, pv.Device)
	}

	// Initialize the VarHandler substitution variables
	varHandler := var_handler.NewVarHandler(map[string]string{
		"$DEVICE_NUM":  fmt.Sprintf("%d", len(deviceNames)),
		"$DEVICE_LIST": strings.Join(deviceNames, " "),
		"$VG_NAME":     lv.VolumeGroup.Name,
		"$LV_NAME":     lv.Name,
		"$LV_SIZE":     fmt.Sprintf("%vK", (lv.Size / 1000)),
		"$PERCENT_VG":  fmt.Sprintf("%v", lv.PercentVG) + "%VG",
	})

	if err := varHandler.ListToVars("$DEVICE_LIST", "$DEVICE"); err != nil {
		return "", fmt.Errorf("invalid internal device list: %w", err)
	}

	return varHandler.ReplaceAll(args), nil
}

func (lv *LogicalVolume) Create(ctx context.Context, rawArgs string) (bool, error) {
	args, err := lv.parseArgs(rawArgs)
	if err != nil {
		return false, err
	}

	existingLVs, err := lvsListVolumes(ctx, lv.Log)
	if err != nil {
		return false, err
	}

	for _, existingLV := range existingLVs {
		if existingLV.Name == lv.Name && existingLV.VGName == lv.VolumeGroup.Name {
			return false, nil
		}
	}

	if _, err := command.Run(fmt.Sprintf("lvcreate --yes %s", args), lv.Log); err != nil {
		return false, fmt.Errorf("could not create logical volume %s: %w", lv.Name, err)
	}

	return true, nil
}

func (lv *LogicalVolume) Remove(ctx context.Context, rawArgs string) (bool, error) {
	args, err := lv.parseArgs(rawArgs)
	if err != nil {
		return false, err
	}

	existingLVs, err := lvsListVolumes(ctx, lv.Log)
	if err != nil {
		return false, err
	}

	for _, existingLV := range existingLVs {
		if existingLV.Name == lv.Name && existingLV.VGName == lv.VolumeGroup.Name {
			if _, err := command.Run(fmt.Sprintf("lvremove --yes %s", args), lv.Log); err != nil {
				return false, fmt.Errorf("could not destroy logical volume %s: %w", lv.Name, err)
			}

			return true, nil
		}
	}

	return true, nil
}

func (lv *LogicalVolume) Change(ctx context.Context, rawArgs string) (bool, error) {
	args, err := lv.parseArgs(rawArgs)
	if err != nil {
		return false, err
	}

	if _, err := command.Run(fmt.Sprintf("lvchange %s", args), lv.Log); err != nil {
		return false, fmt.Errorf("could not change logical volume %s: %w", lv.Name, err)
	}

	return true, nil
}

func (lv *LogicalVolume) Activate(ctx context.Context, rawArgs string) (bool, error) {
	existingLVs, err := lvsListVolumes(ctx, lv.Log)
	if err != nil {
		return false, err
	}

	for _, existingLV := range existingLVs {
		if existingLV.Name == lv.Name && existingLV.VGName == lv.VolumeGroup.Name {
			if existingLV.Attrs[4] == 'a' {
				return false, nil
			}

			return lv.Change(ctx, rawArgs)
		}
	}

	return false, nil
}

func (lv *LogicalVolume) Deactivate(ctx context.Context, rawArgs string) (bool, error) {
	existingLVs, err := lvsListVolumes(ctx, lv.Log)
	if err != nil {
		return false, err
	}

	for _, existingLV := range existingLVs {
		if existingLV.Name == lv.Name && existingLV.VGName == lv.VolumeGroup.Name {
			if existingLV.Attrs[4] != 'a' {
				return false, nil
			}

			return lv.Change(ctx, rawArgs)
		}
	}

	return false, nil
}
