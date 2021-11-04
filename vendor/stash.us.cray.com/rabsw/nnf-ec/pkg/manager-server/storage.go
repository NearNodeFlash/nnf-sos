package server

import (
	"github.com/google/uuid"

	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
)

// Storage represents a unique collection of Server Storage Volumes
// that are identified by a shared Storage Pool ID.
type Storage struct {
	// ID is the Pool ID identified by recovering the NVMe Namespace Metadata
	// for this particular namespace. The Pool ID is common for all namespaces
	// which are part of that storage pool.
	Id uuid.UUID

	ctrl ServerControllerApi

	// The assigned File System for this Storage object, or nil if no
	// fs is present.
	fileSystem FileSystemApi

	ns []StorageNamespace

	nsExpected int // Expected number of namespaces within this storage pool
}

// Storage Namespace represents an NVMe Namespace present on the host.
type StorageNamespace struct {
	// Path refers to the system path for this particular NVMe Namespace. On unix
	// variants, the path is of the form `/dev/nvme[CTRL]n[INDEX]` where CTRL is the
	// parent NVMe Controller and INDEX is assigned by the operating system. INDEX
	// does _not_ refer to the namespace ID (NSID)
	path string

	nsid int

	id uuid.UUID

	poolId    uuid.UUID
	poolIdx   int // Index of this namespace wihin the storage pool
	poolTotal int // Total number of namespaces within the storage pool, as reported by this namespace

	debug bool // If this storage namespace is in debug mode
}

func (s *Storage) GetStatus() StorageStatus {
	return s.ctrl.GetStatus(s)
}

func (s *Storage) Delete() error {
	return s.ctrl.Delete(s)
}

func (s *Storage) CreateFileSystem(fs FileSystemApi, opts FileSystemOptions) error {
	return s.ctrl.CreateFileSystem(s, fs, opts)
}

func (s *Storage) DeleteFileSystem(fs FileSystemApi) error {
	return s.ctrl.DeleteFileSystem(s)
}

// Returns the list of devices for this pool.
func (s *Storage) Devices() []string {
	devices := make([]string, len(s.ns))
	for idx := range s.ns {
		devices[idx] = s.ns[idx].path
	}

	return devices
}

func (s *Storage) UpsertStorageNamespace(sns *StorageNamespace) {
	for _, ns := range s.ns {
		// Debug mode uses matching paths to track the namespaces
		// This isn't practical in production because the same namespace
		// and come and go on different paths.
		if sns.debug {

			if ns.path == sns.path {
				return
			}
		} else {
			if ns.id == sns.id {
				return
			}
		}
	}

	s.ns = append(s.ns, *sns)

	s.nsExpected = sns.poolTotal
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
