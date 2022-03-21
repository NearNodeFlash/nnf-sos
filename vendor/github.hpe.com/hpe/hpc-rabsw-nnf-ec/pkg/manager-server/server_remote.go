package server

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	RemoteStorageServiceId   = "NNFServer"
	RemoteStorageServicePort = 60050
)

type StoragePoolOem struct {
	Namespaces []StorageNamespace `json:"Namespaces"`
}

// Remote Server Controller defines a Server Controller API for a connected server
// that is not local to the running process. This contains many "do nothing" methods
// as management of storage, file systems, and related content is done "off-rabbit".
type RemoteServerController struct{}

func NewRemoteServerController(opts ServerControllerOptions) ServerControllerApi {
	return &RemoteServerController{}
}

func (c *RemoteServerController) Connected() bool { return true }

func (c *RemoteServerController) GetServerInfo() ServerInfo {
	return ServerInfo{}
}

func (c *RemoteServerController) NewStorage(pid uuid.UUID, expectedNamespaces []StorageNamespace) *Storage {
	return &Storage{
		Id:   pid,
		ctrl: c,
	}
}

func (c *RemoteServerController) Delete(s *Storage) error {
	return nil
}

func (c *RemoteServerController) GetStatus(s *Storage) (StorageStatus, error) {
	return StorageStatus_Ready, nil
}

func (c *RemoteServerController) CreateFileSystem(s *Storage, f FileSystemApi, opts FileSystemOptions) error {
	return fmt.Errorf("Cannot create file system on remote server")
}

func (r *RemoteServerController) DeleteFileSystem(s *Storage) error {
	return nil
}
