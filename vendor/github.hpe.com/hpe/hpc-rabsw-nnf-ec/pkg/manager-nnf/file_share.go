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
	"strings"

	"github.hpe.com/hpe/hpc-rabsw-nnf-ec/internal/kvstore"
	sf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/models"
)

type FileShare struct {
	id        string
	mountRoot string

	storageGroupId string
	fileSystemId   string

	storageService *StorageService
}

func (sh *FileShare) OdataId() string {
	fs := sh.storageService.findFileSystem(sh.fileSystemId)
	return fmt.Sprintf("%s/ExportedFileShares/%s", fs.OdataId(), sh.id)
}

func (sh *FileShare) OdataIdRef(ref string) sf.OdataV4IdRef {
	return sf.OdataV4IdRef{OdataId: fmt.Sprintf("%s%s", sh.OdataId(), ref)}
}

func (sh *FileShare) getStatus() *sf.ResourceStatus {
	sg := sh.storageService.findStorageGroup(sh.storageGroupId)
	status, _ := sg.serverStorage.GetStatus()
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
	fileShareUpdateStartLogEntryType
	fileShareUpdateCompleteLogEntryType
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

type fileSharePersistentUpdateCompleteLogEntry struct {
	FileSharePath string `json:"FileSharePath"`
}

func (sh *FileShare) GetKey() string {
	return fileShareRegistryPrefix + strings.Join([]string{sh.fileSystemId, sh.id}, ":")
}

func (sh *FileShare) GetProvider() PersistentStoreProvider {
	fs := sh.storageService.findFileSystem(sh.fileSystemId)
	return fs.storageService
}

func (sh *FileShare) GenerateMetadata() ([]byte, error) {
	return json.Marshal(fileSharePersistentMetadata{
		FileSystemId:   sh.fileSystemId,
		StorageGroupId: &sh.storageGroupId,
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
	case fileShareUpdateCompleteLogEntryType:
		entry := fileSharePersistentUpdateCompleteLogEntry{
			FileSharePath: sh.mountRoot,
		}

		return json.Marshal(entry)
	}
	return nil, nil
}

func (sh *FileShare) Rollback(state uint32) error {
	switch state {
	case fileShareCreateStartLogEntryType:
		fs := sh.storageService.findFileSystem(sh.fileSystemId)
		fs.deleteFileShare(sh)
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
	ids := strings.SplitN(id, ":", 2)

	return &fileShareRecoveryReplayHandler{
		fileShareId:    ids[1],
		fileSystemId:   ids[0],
		storageService: r.storageService,
	}
}

type fileShareRecoveryReplayHandler struct {
	fileShareId      string
	fileSystemId     string
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

	fileSystem.shares = append(fileSystem.shares, FileShare{
		id:             rh.fileShareId,
		fileSystemId:   metadata.FileSystemId,
		mountRoot:      *metadata.MountRoot,
		storageGroupId: *metadata.StorageGroupId,
		storageService: rh.storageService,
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
