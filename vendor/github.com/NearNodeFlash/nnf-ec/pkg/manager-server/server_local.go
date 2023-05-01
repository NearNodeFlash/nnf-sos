/*
 * Copyright 2020, 2021, 2022 Hewlett Packard Enterprise Development LP
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

package server

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/NearNodeFlash/nnf-ec/pkg/logging"

	"github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/nvme"
)

type DefaultServerControllerProvider struct{}

func (DefaultServerControllerProvider) NewServerController(opts ServerControllerOptions) ServerControllerApi {
	if opts.Local {
		return &LocalServerController{}
	}
	return NewRemoteServerController(opts)
}

type LocalServerController struct {
	storages []*Storage
}

func (c *LocalServerController) Connected() bool { return true }

func (c *LocalServerController) GetServerInfo() ServerInfo {
	serverInfo := ServerInfo{}

	output, err := c.run("lctl list_nids")
	if err != nil {
		return serverInfo
	}

	for _, nid := range strings.Split(string(output), "\n") {
		if strings.Contains(nid, "@") {
			serverInfo.LNetNids = append(serverInfo.LNetNids, nid)
		}
	}

	return serverInfo
}

func (c *LocalServerController) NewStorage(pid uuid.UUID, expectedNamespaces []StorageNamespace) *Storage {

	storage := &Storage{
		Id:                   pid,
		expectedNamespaces:   expectedNamespaces,
		discoveredNamespaces: make([]StorageNamespace, 0),
		ctrl:                 c,
	}

	c.storages = append(c.storages, storage)
	return storage
}

func (c *LocalServerController) Delete(s *Storage) error {
	for storageIdx, storage := range c.storages {
		if storage.Id == s.Id {
			c.storages = append(c.storages[:storageIdx], c.storages[storageIdx+1:]...)
			break
		}
	}

	return nil
}

func (c *LocalServerController) GetStatus(s *Storage) (StorageStatus, error) {

	if len(s.discoveredNamespaces) < len(s.expectedNamespaces) {
		if err := c.Discover(s); err != nil {
			return StorageStatus_Error, err
		}
	}

	if len(s.discoveredNamespaces) < len(s.expectedNamespaces) {
		return StorageStatus_Starting, nil
	}

	return StorageStatus_Ready, nil
}

func (c *LocalServerController) CreateFileSystem(s *Storage, fs FileSystemApi, opts FileSystemOptions) error {
	if err := fs.Create(s.Devices(), opts); err != nil {
		return err
	}

	return nil
}

func (c *LocalServerController) DeleteFileSystem(s *Storage, fs FileSystemApi) error {
	return fs.Delete()
}

func (c *LocalServerController) MountFileSystem(s *Storage, fs FileSystemApi, mountPoint string) error {
	if mountPoint == "" {
		return nil
	}

	if err := fs.Mount(mountPoint); err != nil {
		return err
	}

	return nil
}

func (c *LocalServerController) UnmountFileSystem(s *Storage, fs FileSystemApi, mountPoint string) error {
	if err := fs.Unmount(mountPoint); err != nil {
		return err
	}

	return nil
}

func (c *LocalServerController) Discover(s *Storage) error {
	if devices, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
		return c.discoverUsingSuppliedDevices(s, devices)
	}

	return c.discoverViaNamespaces()
}

func (c *LocalServerController) discoverUsingSuppliedDevices(s *Storage, devices string) error {

	// All devices will be placed into the same pool.
	devList := strings.Split(devices, ",")
	for idx, devSupplied := range devList {
		log.Info("Supplied device ", " idx ", idx, " dev ", devSupplied)
		s.discoveredNamespaces = append(s.discoveredNamespaces, StorageNamespace{
			Id:   nvme.NamespaceIdentifier(idx),
			path: devSupplied,
		})
	}

	s.expectedNamespaces = s.discoveredNamespaces

	return nil
}

// Discover NVMe Namespaces
func (c *LocalServerController) discoverViaNamespaces() error {

	data, err := c.run("nvme list --output-format=json")
	if err != nil {
		return err
	}

	devices := nvmeListDevices{}
	if err := json.Unmarshal(data, &devices); err != nil {
		return err
	}

	for _, storage := range c.storages {

		// Skip storages that already discovered the expected namespaces
		if len(storage.expectedNamespaces) == len(storage.discoveredNamespaces) {
			continue
		}

		// For this storage resource, try to build the list of discovered namespaces
		// from that which is expected.
		for _, ns := range storage.expectedNamespaces {
			for _, dev := range devices.Devices {
				if dev.equals(ns) {

					discovered := false
					for _, ns := range storage.discoveredNamespaces {
						if dev.equals(ns) {
							discovered = true
							break
						}
					}

					if !discovered {
						ns := StorageNamespace{
							SerialNumber: dev.SerialNumber,
							Id:           nvme.NamespaceIdentifier(dev.Namespace),
							path:         dev.DevicePath,
						}

						storage.discoveredNamespaces = append(storage.discoveredNamespaces, ns)
					}

					break
				}
			}
		}
	}

	return nil
}

func (c *LocalServerController) findStorage(pid uuid.UUID) *Storage {
	for idx, s := range c.storages {
		if s.Id == pid {
			return c.storages[idx]
		}
	}

	return nil
}

func (c *LocalServerController) namespaces() ([]string, error) {
	nss, err := c.run("ls -A1 /dev/nvme* | { grep -E 'nvme[0-9]+n[0-9]+' || :; }")

	// The above will return an err if zero namespaces exist. In
	// this case, ENOENT is returned and we should instead return
	// a zero length array.
	if exit, ok := err.(*exec.ExitError); ok {
		if syscall.Errno(exit.ExitCode()) == syscall.ENOENT {
			return make([]string, 0), nil
		}
	}

	return strings.Fields(string(nss)), err
}

func (c *LocalServerController) run(cmd string) ([]byte, error) {
	return logging.Cli.Trace(cmd, func(cmd string) ([]byte, error) {
		return exec.Command("bash", "-c", cmd).Output()
	})
}

// The NVMe List command, `nvme list --output-format=json`, will output a list of devices
//
//	{
//	  "Devices" : [
//	    {
//	      "NameSpace" : 1,
//	      "DevicePath" : "/dev/nvme0n1",
//	      "Firmware" : "1TCRS102",
//	      "Index" : 0,
//	      "ModelNumber" : "KIOXIA KCM7DRJE1T92",
//	      "ProductName" : "Non-Volatile memory controller: KIOXIA Corporation Device 0x0014",
//	      "SerialNumber" : "YCT0A08H03A1",
//	      "UsedBytes" : 528384,
//	      "MaximumLBA" : 15256,
//	      "PhysicalSize" : 62488576,
//	      "SectorSize" : 4096
//	    },
//	    ...
//	  ]
//	}
//
// We use this data to match expected namespaces for a particular drive against what is
// found by the host to grab the Device Path which can be used to create a file system.
type nvmeListDevice struct {
	Namespace    uint32 `json:"Namespace"`
	DevicePath   string `json:"DevicePath"`
	SerialNumber string `json:"SerialNumber"`
}

type nvmeListDevices struct {
	Devices []nvmeListDevice `json:"Devices"`
}

func (dev *nvmeListDevice) equals(sn StorageNamespace) bool {
	return dev.SerialNumber == sn.SerialNumber && dev.Namespace == uint32(sn.Id)
}
