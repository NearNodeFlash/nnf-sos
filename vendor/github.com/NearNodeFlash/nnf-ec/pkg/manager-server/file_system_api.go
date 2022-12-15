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
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"k8s.io/mount-utils"

	"github.com/NearNodeFlash/nnf-ec/pkg/logging"
)

type FileSystemControllerApi interface {
	NewFileSystem(oem FileSystemOem) (FileSystemApi, error)
}

func NewFileSystemController(config *ConfigFile) FileSystemControllerApi {
	return &fileSystemController{config: config}
}

type fileSystemController struct {
	config *ConfigFile
}

// NewFileSystem -
func (c *fileSystemController) NewFileSystem(oem FileSystemOem) (FileSystemApi, error) {
	return FileSystemRegistry.NewFileSystem(oem)
}

var (
	FileSystemController FileSystemControllerApi
)

func Initialize() error {

	config, err := loadConfig()
	if err != nil {
		log.WithError(err).Errorf("Failed to load File System configuration")
		return err
	}

	FileSystemController = NewFileSystemController(config)

	return nil
}

type FileSystemOptions = map[string]interface{}

// FileSystemApi - Defines the interface for interacting with various file systems
// supported by the NNF element controller.
type FileSystemApi interface {
	New(oem FileSystemOem) (FileSystemApi, error)

	IsType(oem FileSystemOem) bool // Returns true if the provided oem fields match the file system type, false otherwise
	IsMockable() bool              // Returns true if the file system can be instantiated by the mock server, false otherwise

	Type() string
	Name() string

	Create(devices []string, opts FileSystemOptions) error
	Delete() error

	Mount(mountpoint string) error
	Unmount(mountpoint string) error

	GenerateRecoveryData() map[string]string
	LoadRecoveryData(data map[string]string)
	LoadDeviceList(devices []string)
}

// FileSystem - Represents an abstract file system, with individual operations
// defined by the underlying FileSystemApi implementation
type FileSystem struct {
	name    string
	devices []string
}

func (f *FileSystem) LoadDeviceList(devices []string) {
	f.devices = devices
}

func (f *FileSystem) mount(source string, target string, fstype string, options []string) error {

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	mounter := mount.New("")
	mounted, err := mounter.IsMountPoint(target)
	if err != nil {
		return err
	}

	if !mounted {
		if err := mounter.Mount(source, target, fstype, options); err != nil {
			log.Errorf("Mount failed: %v", err)
		}
	}

	return nil
}

func (f *FileSystem) Unmount(mountpoint string) error {
	if mountpoint == "" {
		return nil
	}

	mounter := mount.New("")
	mounted, err := mounter.IsMountPoint(mountpoint)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return err
	}

	if mounted {
		if err := mounter.Unmount(mountpoint); err != nil {
			log.Errorf("Unmount failed: %v", err)
			return err
		}
	}

	_ = os.Remove(mountpoint) // Attempt to remove the directory but don't fuss about it if not

	return nil
}

type FileSystemError struct {
	command string
	stdout  bytes.Buffer
	stderr  bytes.Buffer
	err     error
}

func (e *FileSystemError) Error() string {
	errorString := fmt.Sprintf("Error Running Command '%s'", e.command)
	if e.stdout.Len() > 0 {
		errorString += fmt.Sprintf(", StdOut: %s", e.stdout.String())
	}
	if e.stderr.Len() > 0 {
		errorString += fmt.Sprintf(", StdErr: %s", e.stderr.String())
	}
	if e.err != nil {
		errorString += fmt.Sprintf(", Internal Error: %s", e.err)
	}
	return errorString
}

func (e *FileSystemError) Unwrap() error {
	return e.err
}

func (*FileSystem) run(cmd string) ([]byte, error) {
	return logging.Cli.Trace2(logging.LogToStdout, cmd, func(cmd string) ([]byte, error) {

		fsError := FileSystemError{command: cmd}
		shellCmd := exec.Command("bash", "-c", cmd)
		shellCmd.Stdout = &fsError.stdout
		shellCmd.Stderr = &fsError.stderr
		err := shellCmd.Run()
		if err != nil {
			// Command failed, return stderr
			return nil, &fsError
		}
		// Command success, return stdout
		return fsError.stdout.Bytes(), nil
	})
}

type FileSystemOemMkfsMount struct {
	// The mkfs commandline, minus the "mkfs" command itself.
	Mkfs string `json:"Mkfs,omitempty"`

	// Arguments for the mount-utils library.
	Mount []string `json:"Mount,omitempty"`
}

type FileSystemOemLvm struct {
	// The pvcreate commandline, minus the "pvcreate" command.
	PvCreate string `json:"PvCreate,omitempty"`

	// The vgcreate commandline, minus the "vgcreate" command.
	VgCreate string `json:"VgCreate,omitempty"`

	// The lvcreate commandline, minus the "lvcreate" command.
	LvCreate string `json:"LvCreate,omitempty"`
}

type FileSystemOemZfs struct {
	// The zpool create commandline, minus the "zpool create" command.
	ZpoolCreate string `json:"ZpoolCreate,omitempty"`

	// For "zfs create", specify the args in the mkfs.lustre --mkfsoptions arg.
}

type FileSystemOemLustre struct {
	MgsNode    string `json:"MgsNode,omitempty"`
	TargetType string `json:"TargetType"`
	Index      int    `json:"Index"`
	BackFs     string `json:"BackFs"`
}

type FileSystemOemGfs2 struct {
	ClusterName string `json:"ClusterName"`
}

// File System OEM defines the structure that is expected to be included inside a
// Redfish / Swordfish FileSystemV122FileSystem
type FileSystemOem struct {
	Type   string              `json:"Type"`
	Name   string              `json:"Name"`
	Lustre FileSystemOemLustre `json:"Lustre,omitempty"`
	Gfs2   FileSystemOemGfs2   `json:"Gfs2,omitempty"`

	LvmCmd    FileSystemOemLvm       `json:"Lvm,omitempty"`
	MkfsMount FileSystemOemMkfsMount `json:"MkfsMount,omitempty"`
	ZfsCmd    FileSystemOemZfs       `json:"Zfs,omitempty"`
}

// File System Registry - Maintains a list of eligible file systems registered in the system.
type fileSystemRegistry []FileSystemApi

var (
	FileSystemRegistry fileSystemRegistry
)

func (r *fileSystemRegistry) RegisterFileSystem(fileSystem FileSystemApi) {

	// Sanity check provided FS has a valid type
	if len(fileSystem.Type()) == 0 {
		panic("File system has no type")
	}

	// Sanity check for duplicate file systems
	for _, fs := range *r {
		if fs.Type() == fileSystem.Type() {
			panic(fmt.Sprintf("File system '%s' already registered", fileSystem.Type()))
		}
	}

	*r = append(*r, fileSystem)
}

func (r *fileSystemRegistry) NewFileSystem(oem FileSystemOem) (FileSystemApi, error) {
	for _, fs := range *r {
		if fs.IsType(oem) {
			return fs.New(oem)
		}
	}

	return nil, nil
}

func setFileSystemPermissions(f FileSystemApi, opts FileSystemOptions) (err error) {
	const (
		UserID  = "userID"
		GroupID = "groupID"
	)

	userID := 0
	if _, exists := opts[UserID]; exists {
		userID = opts[UserID].(int)
	}

	groupID := 0
	if _, exists := opts[GroupID]; exists {
		groupID = opts[GroupID].(int)
	}

	// The owner/group of the file system has to be set while the file system is mounted.
	// We mount the file system here at a temporary location and then immediately unmount
	// it after the Chown() call.
	mountpath := "/mnt/nnf/client/" + f.Name()
	if err := f.Mount(mountpath); err != nil {
		return err
	}

	defer func() {
		unmountErr := f.Unmount(mountpath)
		if err == nil {
			err = unmountErr
		}
	}()

	if err := os.Chown(mountpath, userID, groupID); err != nil {
		return err
	}

	return nil
}
