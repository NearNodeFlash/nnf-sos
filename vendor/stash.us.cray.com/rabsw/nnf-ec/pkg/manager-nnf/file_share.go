package nnf

import (
	"encoding/json"
	"fmt"
	"strconv"

	"stash.us.cray.com/rabsw/nnf-ec/internal/kvstore"
	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
	server "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-server"
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
func (fs *FileSystem) createFileShare(sg *StorageGroup, mountRoot string) *FileShare {
	var fileShareId = -1
	for _, fileShare := range fs.shares {
		id, _ := strconv.Atoi(fileShare.id)

		if fileShareId <= id {
			fileShareId = id
		}
	}

	fileShareId = fileShareId + 1

	return &FileShare{
		id:           strconv.Itoa(fileShareId),
		storageGroup: sg,
		mountRoot:    mountRoot,
		fileSystem:   fs,
	}
}

func (sh *FileShare) initialize(mountpoint string) error {

	opts := server.FileSystemOptions{
		"mountpoint": mountpoint,
	}

	return sh.storageGroup.serverStorage.CreateFileSystem(sh.fileSystem.fsApi, opts)
}

func (sh *FileShare) getStatus() *sf.ResourceStatus {

	return &sf.ResourceStatus{
		Health: sf.OK_RH,
		State:  sh.storageGroup.serverStorage.GetStatus().State(),
	}
}

// Persistent Object API

const fileShareRegistryPrefix = "SH"

const (
	fileShareCreateStartLogEntryType = iota
	fileShareCreateCompleteLogEntryType
	fileShareDeleteCompleteLogEntryType
)

type fileSharePersistentMetadata struct {
	FileSystemId   string  `json:"FileSystemId"`
	StorageGroupId *string `json:"StorageGroupId,omitempty"`
	MountRoot      *string `json:"MountRoot,omitempty"`
}

func (sh *FileShare) GetKey() string                       { return fileShareRegistryPrefix }
func (sh *FileShare) GetProvider() PersistentStoreProvider { return sh.fileSystem.storageService }

func (sh *FileShare) GenerateMetadata() ([]byte, error) {
	return json.Marshal(fileSharePersistentMetadata{
		FileSystemId:   sh.fileSystem.id,
		StorageGroupId: &sh.storageGroup.id,
		MountRoot:      &sh.mountRoot,
	})
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

	fileSystem.shares = append(rh.fileSystem.shares, FileShare{
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
