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
	"strings"

	"github.com/NearNodeFlash/nnf-sos/pkg/command"
	"github.com/NearNodeFlash/nnf-sos/pkg/var_handler"
	"github.com/go-logr/logr"
)

type ZpoolCommandArgs struct {
	Create  string
	Replace string

	Vars map[string]string
}

type Zpool struct {
	Log         logr.Logger
	CommandArgs ZpoolCommandArgs

	Devices []string
	Name    string
	DataSet string
}

// Check that Lvm implements the BlockDevice interface
var _ BlockDevice = &Zpool{}

func (z *Zpool) parseArgs(args string) string {
	m := map[string]string{
		"$DEVICE_NUM":   fmt.Sprintf("%d", len(z.Devices)),
		"$DEVICE_NUM-1": fmt.Sprintf("%d", len(z.Devices)-1),
		"$DEVICE_NUM-2": fmt.Sprintf("%d", len(z.Devices)-2),
		"$DEVICE_LIST":  strings.Join(z.Devices, " "),
		"$POOL_NAME":    z.Name,
	}

	for k, v := range z.CommandArgs.Vars {
		m[k] = v
	}

	// Initialize the VarHandler substitution variables
	varHandler := var_handler.NewVarHandler(m)
	return varHandler.ReplaceAll(args)
}

func (z *Zpool) Create(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	output, err := command.Run("zpool list -H", z.Log)
	if err != nil {
		return false, fmt.Errorf("could not list zpools")
	}

	// Check whether the zpool already exists
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == z.Name {
			if fields[9] == "ONLINE" {
				return false, nil
			}
			return false, fmt.Errorf("zpool has unexpected health %s", fields[9])
		}
	}

	if _, err := command.Run(fmt.Sprintf("zpool create %s", z.parseArgs(z.CommandArgs.Create)), z.Log); err != nil {
		return false, fmt.Errorf("could not create file system: %w", err)
	}

	return true, nil
}

func (z *Zpool) Destroy(ctx context.Context) (bool, error) {
	_, err := command.Run(fmt.Sprintf("zpool destroy %s", z.Name), z.Log)
	if err != nil && !strings.Contains(err.Error(), "no such pool") {
		return false, fmt.Errorf("could not destroy zpool %s", z.Name)
	}

	return true, nil
}

func (z *Zpool) Activate(ctx context.Context) (bool, error) {
	return false, nil
}

func (z *Zpool) Deactivate(ctx context.Context, full bool) (bool, error) {
	return false, nil
}

func (z *Zpool) GetDevice() string {
	// The zpool device is just the name of the zpool
	return fmt.Sprintf("%s/%s", z.Name, z.DataSet)
}

func (z *Zpool) CheckFormatted() (bool, error) {
	output, err := command.Run(fmt.Sprintf("zfs get -H lustre:fsname %s", z.GetDevice()), z.Log)
	if err != nil {
		// If the error is because the data set doesn't exist yet, then that means it's not formatted
		if strings.Contains(err.Error(), "dataset does not exist") {
			return false, nil
		}

		return false, fmt.Errorf("could not run 'zfs get' to check for zpool device %w", err)
	}

	if len(output) == 0 {
		return false, fmt.Errorf("'zfs get' returned no output")
	}

	return true, nil
}

func (z *Zpool) CheckExists(ctx context.Context) (bool, error) {
	output, err := command.Run("zpool list -H", z.Log)
	if err != nil {
		return false, fmt.Errorf("could not list zpools")

	}

	// Check whether the zpool already exists
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == z.Name {
			return true, nil
		}
	}

	return false, nil
}
func (z *Zpool) CheckReady(ctx context.Context) (bool, error) {
	return true, nil
}

func (z *Zpool) CheckHealth(ctx context.Context) (bool, error) {
	output, err := command.Run("zpool list -H", z.Log)
	if err != nil {
		return false, fmt.Errorf("could not list zpools")

	}

	// Check whether the zpool is healthy
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == z.Name {
			if fields[9] == "ONLINE" {
				return true, nil
			}
			return false, nil
		}
	}

	return false, nil
}

func (z *Zpool) Repair(ctx context.Context) error {
	// "zpool status" doesn't have a json output we can rely on, so we have to scrape the
	// human readable text.
	output, err := command.Run(fmt.Sprintf("zpool status -P %s", z.Name), z.Log)
	if err != nil {
		return fmt.Errorf("could not list zpools")
	}

	// Create a map of the underlying devices that should be part of this zpool. Use the output of
	// "zpool status" to determine whether the device is present
	deviceMap := map[string]bool{}
	for _, device := range z.Devices {
		deviceMap[device] = false
		// If the device is present, mark the map entry as "true"
		for _, line := range strings.Split(output, "\n") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				if strings.HasPrefix(fields[0], device) {
					if fields[1] == "ONLINE" {
						deviceMap[device] = true
					}
				}
			}
		}
	}

	// Limit the zpool status output to only devices with an error status
	output, err = command.Run(fmt.Sprintf("zpool status -eP %s", z.Name), z.Log)
	if err != nil {
		return fmt.Errorf("could not list zpools")
	}

	// Make a list of all the devices currently in the zpool that have an unealthy status
	unhealthyDevices := []string{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 {
			if strings.HasPrefix(fields[0], "/dev") {
				if fields[1] != "ONLINE" {
					unhealthyDevices = append(unhealthyDevices, fields[0])
				}
			}
		}
	}

	// Find any devices that aren't currently part of the zpool and and use them to replace the
	// unhealthy devices
	i := 0
	for device, healthy := range deviceMap {
		if !healthy {
			_, err = command.Run(fmt.Sprintf("zpool replace %s %s %s", z.Name, unhealthyDevices[i], device), z.Log)
			if err != nil {
				return fmt.Errorf("could not run zpool replace")
			}

			i++
		}
	}

	return nil
}

func ZpoolImportAll(log logr.Logger) (bool, error) {
	// If test environment or KIND, don't do anything
	_, found := os.LookupEnv("NNF_TEST_ENVIRONMENT")
	if found || os.Getenv("ENVIRONMENT") == "kind" {
		return false, nil
	}

	_, err := command.Run("zpool import -a", log)
	if err != nil {
		if strings.Contains(err.Error(), "no pools available") {
			return false, nil
		} else {
			return false, fmt.Errorf("could not import zpools: %w", err)
		}
	}

	return true, nil
}
