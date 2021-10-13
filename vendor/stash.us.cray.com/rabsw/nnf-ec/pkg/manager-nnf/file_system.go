package nnf

import (
	"encoding/json"
	"fmt"

	server "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-server"
	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"

	"stash.us.cray.com/rabsw/nnf-ec/internal/kvstore"
)

type FileSystem struct {
	id          string
	accessModes []string

	fsApi  server.FileSystemApi
	shares []FileShare

	storagePool    *StoragePool
	storageService *StorageService
}

func (fs *FileSystem) OdataId() string {
	return fmt.Sprintf("%s/FileSystems/%s", fs.storageService.OdataId(), fs.id)
}

func (fs *FileSystem) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", fs.OdataId(), ref)}
}

func (fs *FileSystem) findFileShare(id string) *FileShare {
	for fileShareIdx, fileShare := range fs.shares {
		if fileShare.id == id {
			return &fs.shares[fileShareIdx]
		}
	}

	return nil
}

func (fs *FileSystem) deleteFileShare(id string) {
	for shareIdx, share := range fs.shares {
		if share.id == id {
			fs.shares = append(fs.shares[:shareIdx], fs.shares[shareIdx+1:]...)
			break
		}
	}
}

// Persistent Object API

const fileSystemRegistryPrefix = "FS"

type fileSystemPersistentMetadata struct {
	StoragePoolId  string `json:"StoragePoolId"`
	FileSystemType string `json:"FileSystemType"`
	FileSystemName string `json:"FileSystemName"`
}

func (fs *FileSystem) GetKey() string                       { return fileSystemRegistryPrefix + fs.id }
func (fs *FileSystem) GetProvider() PersistentStoreProvider { return fs.storageService }

func (fs *FileSystem) GenerateMetadata() ([]byte, error) {
	return json.Marshal(fileSystemPersistentMetadata{
		StoragePoolId:  fs.storagePool.id,
		FileSystemType: fs.fsApi.Type(),
		FileSystemName: fs.fsApi.Name(),
	})
}

func (fs *FileSystem) GenerateStateData(state uint32) ([]byte, error) {
	return nil, nil
}

func (fs *FileSystem) Rollback(state uint32) error {
	return nil
}

// Persistent Object Storage API

const (
	fileSystemCreateStartLogEntryType = iota
	fileSystemCreateCompleteLogEntryType
	fileSystemDeleteCompleteLogEntryType
)

type fileSystemRecoveryRegistry struct {
	storageService *StorageService
}

func NewFileSystemRecoveryRegistry(s *StorageService) kvstore.Registry {
	return &fileSystemRecoveryRegistry{storageService: s}
}

func (*fileSystemRecoveryRegistry) Prefix() string { return fileSystemRegistryPrefix }

func (r *fileSystemRecoveryRegistry) NewReplay(id string) kvstore.ReplayHandler {
	return &fileSystemRecoveryReplyHandler{
		fileSystemId:   id,
		storageService: r.storageService,
	}
}

type fileSystemRecoveryReplyHandler struct {
	fileSystemId     string
	lastLogEntryType uint32
	storageService   *StorageService
}

// Metadata handles the metadata TLV type for the replay
func (rh *fileSystemRecoveryReplyHandler) Metadata(data []byte) error {
	metadata := &fileSystemPersistentMetadata{}
	if err := json.Unmarshal(data, metadata); err != nil {
		return err
	}

	storagePool := rh.storageService.findStoragePool(metadata.StoragePoolId)

	rh.storageService.fileSystems = append(rh.storageService.fileSystems, FileSystem{
		id:             rh.fileSystemId,
		storagePool:    storagePool,
		storageService: rh.storageService,
	})

	return nil
}

func (rh *fileSystemRecoveryReplyHandler) Entry(t uint32, data []byte) error {
	return nil
}

func (rh *fileSystemRecoveryReplyHandler) Done() error {
	return nil
}
