package server

import (
	"fmt"

	"github.com/google/uuid"
)

var ErrServerControllerDisabled = fmt.Errorf("Server Controller Disabled")

// Server Controller API for when there is no server present, or the server is disabled
type DisabledServerController struct{}

func NewDisabledServerController() ServerControllerApi {
	return &DisabledServerController{}
}

func (*DisabledServerController) Connected() bool {
	return false
}

func (*DisabledServerController) GetServerInfo() ServerInfo {
	return ServerInfo{}
}

func (*DisabledServerController) NewStorage(uuid.UUID, []StorageNamespace) *Storage {
	return nil
}

func (*DisabledServerController) Delete(*Storage) error {
	return ErrServerControllerDisabled
}

func (*DisabledServerController) GetStatus(*Storage) (StorageStatus, error) {
	return StorageStatus_Disabled, nil
}

func (*DisabledServerController) CreateFileSystem(*Storage, FileSystemApi, FileSystemOptions) error {
	return ErrServerControllerDisabled
}

func (*DisabledServerController) MountFileSystem(*Storage, FileSystemOptions) error {
	return ErrServerControllerDisabled
}

func (*DisabledServerController) UnmountFileSystem(*Storage) error {
	return ErrServerControllerDisabled
}

func (*DisabledServerController) DeleteFileSystem(*Storage) error {
	return ErrServerControllerDisabled
}
