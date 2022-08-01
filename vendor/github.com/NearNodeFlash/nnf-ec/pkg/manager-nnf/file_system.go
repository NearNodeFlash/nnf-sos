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

package nnf

import (
	"encoding/json"
	"fmt"
	"strconv"

	server "github.com/NearNodeFlash/nnf-ec/pkg/manager-server"
	"github.com/NearNodeFlash/nnf-ec/pkg/persistent"
	sf "github.com/NearNodeFlash/nnf-ec/pkg/rfsf/pkg/models"
)

type FileSystem struct {
	id          string
	accessModes []string

	fsApi server.FileSystemApi
	fsOem server.FileSystemOem

	shares []FileShare

	storagePoolId  string
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

// Create a file share object with the provided variables and add it to the file systems list of file
// shares. If an ID is not provided, an unused one will be used. If an ID is provided, the caller must check
// that the ID does not already exist.
func (fs *FileSystem) createFileShare(id string, sg *StorageGroup, mountRoot string) *FileShare {

	if len(id) == 0 {
		var fileShareId = -1
		for _, fileShare := range fs.shares {
			if id, err := strconv.Atoi(fileShare.id); err == nil {
				if fileShareId <= id {
					fileShareId = id
				}
			}
		}

		fileShareId = fileShareId + 1
		id = strconv.Itoa(fileShareId)
	}

	sg.fileShareId = id

	fs.shares = append(fs.shares, FileShare{
		id:             id,
		storageGroupId: sg.id,
		mountRoot:      mountRoot,
		fileSystemId:   fs.id,
		storageService: fs.storageService,
	})

	return &fs.shares[len(fs.shares)-1]
}

func (fs *FileSystem) deleteFileShare(sh *FileShare) {

	sg := fs.storageService.findStorageGroup(sh.storageGroupId)
	sg.fileShareId = ""

	for shareIdx, share := range fs.shares {
		if share.id == sh.id {
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

	server.FileSystemOem `json:",inline"`
}

func (fs *FileSystem) GetKey() string                       { return fileSystemRegistryPrefix + fs.id }
func (fs *FileSystem) GetProvider() PersistentStoreProvider { return fs.storageService }

func (fs *FileSystem) GenerateMetadata() ([]byte, error) {
	return json.Marshal(fileSystemPersistentMetadata{
		StoragePoolId:  fs.storagePoolId,
		FileSystemType: fs.fsApi.Type(),
		FileSystemName: fs.fsApi.Name(),
		FileSystemOem:  fs.fsOem,
	})
}

func (fs *FileSystem) GenerateStateData(state uint32) ([]byte, error) {
	return nil, nil
}

func (fs *FileSystem) Rollback(state uint32) error {
	switch state {
	case fileSystemCreateStartLogEntryType:
		fs.storageService.deleteFileSystem(fs)
	}
	return nil
}

// Persistent Object Storage API

const (
	fileSystemCreateStartLogEntryType = iota
	fileSystemCreateCompleteLogEntryType
	fileSystemDeleteStartLogEntryType
	fileSystemDeleteCompleteLogEntryType
)

type fileSystemRecoveryRegistry struct {
	storageService *StorageService
}

func NewFileSystemRecoveryRegistry(s *StorageService) persistent.Registry {
	return &fileSystemRecoveryRegistry{storageService: s}
}

func (*fileSystemRecoveryRegistry) Prefix() string { return fileSystemRegistryPrefix }

func (r *fileSystemRecoveryRegistry) NewReplay(id string) persistent.ReplayHandler {
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

	fsApi, err := server.FileSystemController.NewFileSystem(metadata.FileSystemOem)
	if err != nil {
		return fmt.Errorf("File System %s Replay: Failed to create api %s", rh.fileSystemId, err)
	}

	storagePool := rh.storageService.findStoragePool(metadata.StoragePoolId)
	if storagePool == nil {
		return fmt.Errorf("File System %s Replay: Failed to find storage pool %s", rh.fileSystemId, metadata.StoragePoolId)
	}

	rh.storageService.createFileSystem(rh.fileSystemId, storagePool, fsApi, metadata.FileSystemOem)

	return nil
}

func (rh *fileSystemRecoveryReplyHandler) Entry(t uint32, data []byte) error {
	rh.lastLogEntryType = t
	return nil
}

func (rh *fileSystemRecoveryReplyHandler) Done() (bool, error) {

	if rh.lastLogEntryType == fileSystemDeleteCompleteLogEntryType {
		rh.storageService.deleteFileSystem(rh.storageService.findFileSystem(rh.fileSystemId))
		return true, nil
	}

	return false, nil
}
