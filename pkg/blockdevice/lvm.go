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

package blockdevice

import (
	"context"
	"fmt"
	"strings"

	"github.com/NearNodeFlash/nnf-sos/pkg/blockdevice/lvm"
	"github.com/NearNodeFlash/nnf-sos/pkg/command"
	"github.com/go-logr/logr"
)

type LvmPvCommandArgs struct {
	Create string
	Remove string
}

type LvmVgCommandArgs struct {
	Create    string
	Remove    string
	LockStart string
	LockStop  string
}

type LvmLvCommandArgs struct {
	Create     string
	Remove     string
	Activate   string
	Deactivate string
}

type LvmCommandArgs struct {
	PvArgs LvmPvCommandArgs
	VgArgs LvmVgCommandArgs
	LvArgs LvmLvCommandArgs
}

type Lvm struct {
	Log         logr.Logger
	CommandArgs LvmCommandArgs

	PhysicalVolumes []*lvm.PhysicalVolume
	VolumeGroup     *lvm.VolumeGroup
	LogicalVolume   *lvm.LogicalVolume
}

// Check that Lvm implements the BlockDevice interface
var _ BlockDevice = &Lvm{}

// Create an LVM Device
func (l *Lvm) Create(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	objectCreated := false

	for _, pv := range l.PhysicalVolumes {
		created, err := pv.Create(ctx, l.CommandArgs.PvArgs.Create)
		if err != nil {
			return false, err
		}
		if created {
			objectCreated = true
		}
	}

	created, err := l.VolumeGroup.Create(ctx, l.CommandArgs.VgArgs.Create)
	if err != nil {
		return false, err
	}
	if created {
		objectCreated = true
	}

	if len(l.CommandArgs.VgArgs.LockStart) > 0 {
		created, err := l.VolumeGroup.LockStart(ctx, l.CommandArgs.VgArgs.LockStart)
		if err != nil {
			return false, err
		}
		if created {
			objectCreated = true
		}
	}

	created, err = l.LogicalVolume.Create(ctx, l.CommandArgs.LvArgs.Create)
	if err != nil {
		return false, err
	}
	if created {
		objectCreated = true
	}

	if len(l.CommandArgs.VgArgs.LockStop) > 0 {
		created, err := l.VolumeGroup.LockStop(ctx, l.CommandArgs.VgArgs.LockStop)
		if err != nil {
			return false, err
		}
		if created {
			objectCreated = true
		}
	}

	return objectCreated, nil
}

func (l *Lvm) Destroy(ctx context.Context) (bool, error) {
	objectDestroyed := false

	vgExists, err := l.VolumeGroup.Exists(ctx)
	if err != nil {
		return false, err
	}

	if vgExists && len(l.CommandArgs.VgArgs.LockStart) > 0 {
		destroyed, err := l.VolumeGroup.LockStart(ctx, l.CommandArgs.VgArgs.LockStart)
		if err != nil {
			return false, err
		}
		if destroyed {
			objectDestroyed = true
		}
	}

	destroyed, err := l.LogicalVolume.Remove(ctx, l.CommandArgs.LvArgs.Remove)
	if err != nil {
		return false, err

	}
	if destroyed {
		objectDestroyed = true
	}

	destroyed, err = l.VolumeGroup.Remove(ctx, l.CommandArgs.VgArgs.Remove)
	if err != nil {
		return false, err
	}
	if destroyed {
		objectDestroyed = true
	}

	for _, pv := range l.PhysicalVolumes {
		destroyed, err := pv.Remove(ctx, l.CommandArgs.PvArgs.Remove)
		if err != nil {
			return false, err
		}
		if destroyed {
			objectDestroyed = true
		}
	}

	return objectDestroyed, nil
}

func (l *Lvm) Activate(ctx context.Context) (bool, error) {
	// Make sure the that locking has been started on the VG. The node might have been rebooted
	// since the VG was created
	if len(l.CommandArgs.VgArgs.LockStart) > 0 {
		_, err := l.VolumeGroup.LockStart(ctx, l.CommandArgs.VgArgs.LockStart)
		if err != nil {
			return false, err
		}
	}

	if len(l.CommandArgs.LvArgs.Activate) > 0 {
		_, err := l.LogicalVolume.Activate(ctx, l.CommandArgs.LvArgs.Activate)
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

func (l *Lvm) Deactivate(ctx context.Context) (bool, error) {

	if len(l.CommandArgs.LvArgs.Deactivate) > 0 {
		_, err := l.LogicalVolume.Deactivate(ctx, l.CommandArgs.LvArgs.Deactivate)
		if err != nil {
			return false, err
		}
	}

	if len(l.CommandArgs.VgArgs.LockStop) > 0 {
		_, err := l.VolumeGroup.LockStop(ctx, l.CommandArgs.VgArgs.LockStop)
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

func (l *Lvm) GetDevice() string {
	return fmt.Sprintf("/dev/mapper/%s-%s", strings.Replace(l.VolumeGroup.Name, "-", "--", -1), strings.Replace(l.LogicalVolume.Name, "-", "--", -1))
}

func (l *Lvm) CheckFormatted() (bool, error) {
	output, err := command.Run(fmt.Sprintf("wipefs --noheadings --output type %s", l.GetDevice()), l.Log)
	if err != nil {
		return false, fmt.Errorf("could not run wipefs to determine if device is formatted: %w", err)
	}

	if len(output) == 0 {
		return false, nil
	}

	return true, nil
}
