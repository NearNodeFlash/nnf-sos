package nnf

import (
	"encoding/json"
	"fmt"

	"stash.us.cray.com/rabsw/nnf-ec/internal/kvstore"
	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
)

type FileShare struct {
	id        string
	mountRoot string

	storageGroup *StorageGroup
	fileSystem   *FileSystem
}

func (sh *FileShare) OdataId() string {
	return fmt.Sprintf("%s/ExportedFileShares/%s", sh.fileSystem.OdataId(), sh.id)
}

func (sh *FileShare) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", sh.OdataId(), ref)}
}

func (sh *FileShare) getStatus() *sf.ResourceStatus {

	status, _ := sh.storageGroup.serverStorage.GetStatus()
	return &sf.ResourceStatus{
		Health: sf.OK_RH,
		State:  status.State(),
	}
}

// Persistent Object API

const fileShareRegistryPrefix = "SH"

const (
	fileShareCreateStartLogEntryType = iota
	fileShareCreateCompleteLogEntryType
	fileShareDeleteStartLogEntryType
	fileShareDeleteCompleteLogEntryType
)

type fileSharePersistentMetadata struct {
	FileSystemId   string  `json:"FileSystemId"`
	StorageGroupId *string `json:"StorageGroupId,omitempty"`
	MountRoot      *string `json:"MountRoot,omitempty"`
}

type fileSharePersistentCreateCompleteLogEntry struct {
	FileSharePath string `json:"FileSharePath"`
}

func (sh *FileShare) GetKey() string                       { return fileShareRegistryPrefix + sh.id }
func (sh *FileShare) GetProvider() PersistentStoreProvider { return sh.fileSystem.storageService }

func (sh *FileShare) GenerateMetadata() ([]byte, error) {
	return json.Marshal(fileSharePersistentMetadata{
		FileSystemId:   sh.fileSystem.id,
		StorageGroupId: &sh.storageGroup.id,
		MountRoot:      &sh.mountRoot,
	})
}

func (sh *FileShare) GenerateStateData(state uint32) ([]byte, error) {
	switch state {
	case fileShareCreateCompleteLogEntryType:
		entry := fileSharePersistentCreateCompleteLogEntry{
			FileSharePath: sh.mountRoot,
		}

		return json.Marshal(entry)
	}
	return nil, nil
}

func (sh *FileShare) Rollback(state uint32) error {
	switch state {
	case fileShareCreateStartLogEntryType:
		sh.fileSystem.deleteFileShare(sh)
	}

	return nil
}

// Persistent Object Recovery API

type fileShareRecoveryRegistry struct {
	storageService *StorageService
}

func NewFileShareRecoveryRegistry(s *StorageService) kvstore.Registry {
	return &fileShareRecoveryRegistry{storageService: s}
}

func (r *fileShareRecoveryRegistry) Prefix() string { return fileShareRegistryPrefix }

func (r *fileShareRecoveryRegistry) NewReplay(id string) kvstore.ReplayHandler {
	return &fileShareRecoveryReplayHandler{
		fileShareId:    id,
		storageService: r.storageService,
	}
}

type fileShareRecoveryReplayHandler struct {
	fileShareId      string
	lastLogEntryType uint32
	fileSystem       *FileSystem
	storageService   *StorageService
}

func (rh *fileShareRecoveryReplayHandler) Metadata(data []byte) error {
	metadata := &fileSharePersistentMetadata{}
	if err := json.Unmarshal(data, metadata); err != nil {
		return err
	}

	fileSystem := rh.storageService.findFileSystem(metadata.FileSystemId)
	storageGroup := rh.storageService.findStorageGroup(*metadata.StorageGroupId)

	fileSystem.shares = append(fileSystem.shares, FileShare{
		id:           rh.fileShareId,
		fileSystem:   fileSystem,
		storageGroup: storageGroup,
	})

	rh.fileSystem = fileSystem

	return nil
}

func (rh *fileShareRecoveryReplayHandler) Entry(t uint32, data []byte) error {
	return nil
}

func (rh *fileShareRecoveryReplayHandler) Done() error {
	return nil
}
