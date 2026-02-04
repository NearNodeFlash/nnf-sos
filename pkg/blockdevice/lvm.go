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

package blockdevice

import (
	"context"
	"errors"
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
	Extend    string
	Reduce    string
}

type LvmLvCommandArgs struct {
	Create     string
	Remove     string
	Activate   string
	Deactivate string
	Repair     string
}

type LvmUserCommandArgs struct {
	PreActivate    []string
	PostActivate   []string
	PreDeactivate  []string
	PostDeactivate []string
}

type LvmCommandArgs struct {
	PvArgs   LvmPvCommandArgs
	VgArgs   LvmVgCommandArgs
	LvArgs   LvmLvCommandArgs
	UserArgs LvmUserCommandArgs

	Vars map[string]string
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

	if len(l.CommandArgs.PvArgs.Create) > 0 {
		for _, pv := range l.PhysicalVolumes {
			created, err := pv.Create(ctx, l.CommandArgs.PvArgs.Create)
			if err != nil {
				return false, err
			}
			if created {
				objectCreated = true
			}
		}
	}

	if len(l.CommandArgs.VgArgs.Create) > 0 {
		created, err := l.VolumeGroup.Create(ctx, l.CommandArgs.VgArgs.Create)
		if err != nil {
			return false, err
		}
		if created {
			objectCreated = true
		}
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

	if len(l.CommandArgs.LvArgs.Create) > 0 {
		created, err := l.LogicalVolume.Create(ctx, l.CommandArgs.LvArgs.Create)
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

	// If the node was rebooted, the lockspace may not be started
	if vgExists && len(l.CommandArgs.VgArgs.LockStart) > 0 {
		destroyed, err := l.VolumeGroup.LockStart(ctx, l.CommandArgs.VgArgs.LockStart)
		if err != nil {
			return false, err
		}
		if destroyed {
			objectDestroyed = true
		}
	}

	if len(l.CommandArgs.LvArgs.Remove) > 0 {
		destroyed, err := l.LogicalVolume.Remove(ctx, l.CommandArgs.LvArgs.Remove)
		if err != nil {
			return false, err
		}
		if destroyed {
			objectDestroyed = true
		}
	}

	if len(l.CommandArgs.VgArgs.Remove) > 0 {
		// Check to ensure the VG has no LVs before removing
		if count, err := l.VolumeGroup.NumLVs(ctx); err != nil {
			return false, err
		} else if count != 0 {
			return objectDestroyed, nil
		}

		destroyed, err := l.VolumeGroup.Remove(ctx, l.CommandArgs.VgArgs.Remove)
		if err != nil {
			return false, err
		}
		if destroyed {
			objectDestroyed = true
		}
	}

	if len(l.CommandArgs.PvArgs.Remove) > 0 {
		for _, pv := range l.PhysicalVolumes {
			destroyed, err := pv.Remove(ctx, l.CommandArgs.PvArgs.Remove)
			if err != nil {
				return false, err
			}
			if destroyed {
				objectDestroyed = true
			}
		}
	}

	return objectDestroyed, nil
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

// WaitForMapper - After Lvm is activated, it can take some time before it appears in the '/dev/mapper' directory
// wait for it to appear, polling every second
func (l *Lvm) waitForMapper() error {

	// Default to 10 second timeout
	retryPeriod := 10 * time.Second

	// Look for environment variable to override
	timeoutString, found := os.LookupEnv("NNF_MAPPER_WAIT_TIMEOUT")
	if found {
		timeout, err := strconv.Atoi(timeoutString)
		if err == nil {
			if timeout > 0 {
				retryPeriod = time.Duration(timeout) * time.Second
			}
		}
	}

	// Give the device some time to appear in /dev/mapper
	var err error
	device := l.GetDevice()
	for start := time.Now(); time.Since(start) < retryPeriod; time.Sleep(time.Second) {

		_, err = os.Stat(device)
		if err == nil {
			return nil
		}

		if errors.Is(err, os.ErrNotExist) {
			// file does not exist
			continue
		} else {
			return fmt.Errorf("could not stat device %w", err)
		}
	}

	return fmt.Errorf("timeout waiting for device %w", err)
}

// Activate the LVM
func (l *Lvm) Activate(ctx context.Context) (bool, error) {
	// Make sure the locking has been started on the VG. The node might have been rebooted
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

		// Activation can take a while. Wait for device to become available
		if err := l.waitForMapper(); err != nil {
			return false, err
		}
	}

	return false, nil
}

// Deactivate the LVM
func (l *Lvm) Deactivate(ctx context.Context, full bool) (bool, error) {

	if len(l.CommandArgs.LvArgs.Deactivate) > 0 {
		_, err := l.LogicalVolume.Deactivate(ctx, l.CommandArgs.LvArgs.Deactivate)
		if err != nil {
			return false, err
		}
	}

	// If we're doing a full deactivation, then stop the lockspace as well
	if full && len(l.CommandArgs.VgArgs.LockStop) > 0 {
		_, err := l.VolumeGroup.LockStop(ctx, l.CommandArgs.VgArgs.LockStop)
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

// CheckFormatted checks the LVM to determine if a filesystem has been created there.
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

func (l *Lvm) CheckExists(ctx context.Context) (bool, error) {
	vgExists, err := l.VolumeGroup.Exists(ctx)
	if err != nil {
		return false, err
	}

	if !vgExists {
		return false, nil
	}

	lvExists, err := l.LogicalVolume.Exists(ctx)
	if err != nil {
		return false, err
	}

	if !lvExists {
		return false, nil
	}

	return true, nil
}

func (l *Lvm) CheckHealth(ctx context.Context) (bool, error) {
	// If there is no LV, then consider the block device healthy
	if len(l.CommandArgs.LvArgs.Create) == 0 {
		return true, nil
	}

	return l.LogicalVolume.IsHealthy(ctx)
}

func (l *Lvm) CheckReady(ctx context.Context) (bool, error) {
	// If there is no LV, then consider the block device ready
	if len(l.CommandArgs.LvArgs.Create) == 0 {
		return true, nil
	}

	return l.LogicalVolume.IsSynced(ctx)
}

func (l *Lvm) Repair(ctx context.Context) (err error) {
	if len(l.CommandArgs.PvArgs.Create) > 0 {
		for _, pv := range l.PhysicalVolumes {
			created, err := pv.Create(ctx, l.CommandArgs.PvArgs.Create)
			if err != nil {
				return err
			}

			if created {
				l.Log.Info("created new PV for rebuild", "Name", pv.Device, "VG", l.VolumeGroup.Name)
			}
		}
	}

	if _, err := l.VolumeGroup.Extend(ctx, l.CommandArgs.VgArgs.Extend); err != nil {
		return err
	}

	if _, err := l.Activate(ctx); err != nil {
		return err
	}

	defer func() {
		if _, deactivateErr := l.Deactivate(ctx, false); deactivateErr != nil {
			err = deactivateErr
		}
	}()

	if _, err := l.LogicalVolume.Repair(ctx, l.CommandArgs.LvArgs.Repair); err != nil {
		return err
	}

	if _, err := l.VolumeGroup.Reduce(ctx, l.CommandArgs.VgArgs.Reduce); err != nil {
		return err
	}

	return nil
}

func (l *Lvm) RunCommands(ctx context.Context, complete bool, commands []string, phase string, args map[string]string) (bool, error) {
	if len(commands) == 0 {
		l.Log.Info("Skipping " + phase + " - no commands specified")
		return false, nil
	}

	if complete {
		return false, nil
	}

	// Build the commands from the args provided
	if l.CommandArgs.Vars == nil {
		l.CommandArgs.Vars = make(map[string]string)
	}

	for k, v := range args {
		l.CommandArgs.Vars[k] = v
	}

	for _, rawCommand := range commands {
		formattedCommand, err := l.LogicalVolume.ParseArgs(rawCommand)
		if err != nil {
			return false, fmt.Errorf("could not format %s command: %s: %w", phase, rawCommand, err)
		}
		l.Log.Info(phase, "command", formattedCommand)

		output, err := command.Run(formattedCommand, l.Log)
		if err != nil {
			return false, fmt.Errorf("could not run %s command: %s: %w", phase, formattedCommand, err)
		}

		l.Log.Info(phase, "output", output)
	}

	return true, nil
}

func (l *Lvm) PreActivate(ctx context.Context, complete bool) (bool, error) {
	return l.RunCommands(ctx, complete, l.CommandArgs.UserArgs.PreActivate, "PreActivate", nil)
}

func (l *Lvm) PostActivate(ctx context.Context, complete bool) (bool, error) {
	return l.RunCommands(ctx, complete, l.CommandArgs.UserArgs.PostActivate, "PostActivate", nil)
}

func (l *Lvm) PreDeactivate(ctx context.Context, complete bool) (bool, error) {
	return l.RunCommands(ctx, complete, l.CommandArgs.UserArgs.PreDeactivate, "PreDeactivate", nil)
}

func (l *Lvm) PostDeactivate(ctx context.Context, complete bool) (bool, error) {
	return l.RunCommands(ctx, complete, l.CommandArgs.UserArgs.PostDeactivate, "PostDeactivate", nil)
}
