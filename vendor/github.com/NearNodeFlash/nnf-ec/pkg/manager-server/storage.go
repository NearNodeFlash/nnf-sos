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
	"github.com/google/uuid"

	"github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/nvme"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

// Storage represents a unique collection of Server Storage Volumes
// that are identified by a shared Storage Pool ID.
type Storage struct {
	// ID is the Pool ID identified by recovering the NVMe Namespace Metadata
	// for this particular namespace. The Pool ID is common for all namespaces
	// which are part of that storage pool.
	Id uuid.UUID

	// Expected Namespaces are the list of storage namespaces that are expected
	// to be part of this Storage resource. This is configured once upon creation
	// of the Storage resource.
	expectedNamespaces []StorageNamespace

	// Discovered Namespaces contains the list of storage namespaces that were
	// discovered by the server controller. If everything is operating correctly
	// this list should compare to the expected namespaces.
	discoveredNamespaces []StorageNamespace

	ctrl ServerControllerApi
}

// Storage Namespace represents an NVMe Namespace present on the host.
type StorageNamespace struct {
	// The drive serial number
	SerialNumber string `json:"SerialNumber"`

	// The NVMe Namespace Identifer
	Id nvme.NamespaceIdentifier `json:"Id"`

	// Path refers to the system path for this particular NVMe Namespace. On unix
	// variants, the path is of the form `/dev/nvme[A]n[B]` where A is a number
	// assigned by the OS for any particular NVMe controller, and B is a number
	// assigned by the OS for any NVMe Namespace
	// "A" does _not_ refer to the Controller ID (CNTL ID)
	// "B" does _not_ refer to the Namespace ID (NSID)
	path string
}

func (s *Storage) GetStatus() (StorageStatus, error) {
	return s.ctrl.GetStatus(s)
}

func (s *Storage) Delete() error {
	return s.ctrl.Delete(s)
}

func (s *Storage) CreateFileSystem(fs FileSystemApi, opts FileSystemOptions) error {
	return s.ctrl.CreateFileSystem(s, fs, opts)
}

func (s *Storage) DeleteFileSystem(fs FileSystemApi) error {
	return s.ctrl.DeleteFileSystem(s, fs)
}

func (s *Storage) MountFileSystem(fs FileSystemApi, mountPoint string) error {
	return s.ctrl.MountFileSystem(s, fs, mountPoint)
}

func (s *Storage) UnmountFileSystem(fs FileSystemApi, mountPoint string) error {
	return s.ctrl.UnmountFileSystem(s, fs, mountPoint)
}

// Returns the list of devices for this pool.
func (s *Storage) Devices() []string {
	devices := make([]string, len(s.discoveredNamespaces))
	for idx := range s.discoveredNamespaces {
		devices[idx] = s.discoveredNamespaces[idx].path
	}

	return devices
}

type StorageStatus string

const (
	StorageStatus_Starting StorageStatus = "starting"
	StorageStatus_Ready                  = "ready"
	StorageStatus_Error                  = "error"
	StorageStatus_Disabled               = "disabled"
)

func (s StorageStatus) State() sf.ResourceState {
	switch s {
	case StorageStatus_Starting:
		return sf.STARTING_RST
	case StorageStatus_Ready:
		return sf.ENABLED_RST
	case StorageStatus_Error:
		return sf.UNAVAILABLE_OFFLINE_RST
	default:
		return sf.DISABLED_RST
	}
}
