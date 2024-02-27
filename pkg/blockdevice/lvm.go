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
	"os"
	"strconv"
	"strings"
	"time"

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

// Destroy the LVM
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

// Activate the LVM
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

// Deactivate the LVM
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

// GetDevice generates the name used in /dev/mapper for the vg/lv
func (l *Lvm) GetDevice() string {
	// Add a hypen to our name to match up with /dev/mapper.
	//
	// According to this article, https://access.redhat.com/solutions/656673
	// the hyphen '-' is used to generate a unique single string to reference the device.
	// The hyphen is used by LVM2 as the field separator when constructing device-mapper device names
	// from LVM2 volume group and logical volume names:
	//		dm name := 'vg_name' + '-' + 'lv_name'
	// Since LVM2 permits the hyphen as a character within a VG or LV name this scheme requires embedded
	// hyphens to be escaped when constructing the dm name. The escaping scheme is to double any embedded hyphens so for e.g.:
	//      Volume Group: my-vg
	//      Logical Volume: my-lv
	// Is encoded as: my--vg-my--lv
	// NOTE!!!
	// The encoding method might be changed without notice in a future release, so you should always use /dev/vg/lv to access a device, and dmsetup splitname to decode a name.

	return fmt.Sprintf("/dev/mapper/%s-%s", strings.Replace(l.VolumeGroup.Name, "-", "--", -1), strings.Replace(l.LogicalVolume.Name, "-", "--", -1))
}

// CheckFormatted determines if the device has been formatted
func (l *Lvm) CheckFormatted() (bool, error) {

	// Default to 10 second timeout
	retryPeriod := 10 * time.Second

	// Look for environment variable to override
	timeoutString, found := os.LookupEnv("NNF_WIPEFS_MAPPER_WAIT_COMMAND_TIMEOUT")
	if found {
		timeout, err := strconv.Atoi(timeoutString)
		if err == nil {
			if timeout > 0 {
				retryPeriod = time.Duration(timeout) * time.Second
			}
		}
	}

	// Give the device some time to appear in /dev/mapper
	// Retry failing wipefs while we wait
	var err error
	var output string
	device := l.GetDevice()
	for start := time.Now(); time.Since(start) < retryPeriod; time.Sleep(time.Second) {

		output, err = command.Run(fmt.Sprintf("wipefs --noheadings --output type %s", device), l.Log)
		if err != nil {
			// stderr output from wipefs includes:
			// "probing initialization failed: No such file or directory\n"
			// Search for "No such file or directory" and continue looping if it is present.
			// Exit with error if it is not.
			const wipefsExpectedError = "No such file or directory"
			if strings.Contains(err.Error(), wipefsExpectedError) {
				continue
			}

			return false, fmt.Errorf("could not run wipefs to determine if device is formatted %w", err)
		}

		// With no output, the filesystem hasn't been formatted
		if len(output) == 0 {
			return false, nil
		}

		return true, nil
	}

	// We exhausted our retry time, fail the operation
	return false, fmt.Errorf("could not run wipefs to determine if device is formatted %w", err)
}
