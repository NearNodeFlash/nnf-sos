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
	"fmt"

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
	Id   nvme.NamespaceIdentifier               `json:"Id"`
	Guid nvme.NamespaceGloballyUniqueIdentifier `json:"Guid"`

	// Path refers to the system path for this particular NVMe Namespace. On unix
	// variants, the path is of the form `/dev/nvme[CTRL]n[INDEX]` where CTRL is the
	// parent NVMe Controller and INDEX is assigned by the operating system. INDEX
	// does _not_ refer to the namespace ID (NSID)
	path string

	debug bool // If this storage namespace is in debug mode
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

func (s *Storage) MountFileSystem(fs FileSystemApi, opts FileSystemOptions) error {
	return s.ctrl.MountFileSystem(s, fs, opts)
}

func (s *Storage) UnmountFileSystem(fs FileSystemApi, opts FileSystemOptions) error {
	return s.ctrl.UnmountFileSystem(s, fs)
}

// Returns the list of devices for this pool.
func (s *Storage) Devices() []string {
	devices := make([]string, len(s.discoveredNamespaces))
	for idx := range s.discoveredNamespaces {
		devices[idx] = s.discoveredNamespaces[idx].path
	}

	return devices
}

func (a StorageNamespace) String() string {
	return fmt.Sprintf("NSID: %d GUID: %s", a.Id, a.Guid.String())
}

func (a StorageNamespace) Equals(b *StorageNamespace) bool {
	return a.Id == b.Id && a.Guid == b.Guid
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
