package server

import (
	"os/exec"

	log "github.com/sirupsen/logrus"

	"stash.us.cray.com/rabsw/nnf-ec/pkg/logging"
)

type FileSystemControllerApi interface {
	NewFileSystem(oem FileSystemOem) FileSystemApi
}

func NewFileSystemController(config *ConfigFile) FileSystemControllerApi {
	return &fileSystemController{config: config}
}

type fileSystemController struct {
	config *ConfigFile
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
	Type() string
	IsType(oem FileSystemOem) bool
	Name() string

	Create(devices []string, opts FileSystemOptions) error
	Delete() error

	Mount(mountpoint string) error
	Unmount() error
}

// NewFileSystem -
func (c *fileSystemController) NewFileSystem(oem FileSystemOem) (fsApi FileSystemApi) {

	if (&FileSystemZfs{}).IsType(oem) {
		fsApi = NewFileSystemZfs(oem)
	} else if (&FileSystemLvm{}).IsType(oem) {
		fsApi = NewFileSystemLvm(oem)
	} else if (&FileSystemXfs{}).IsType(oem) {
		fsApi = NewFileSystemXfs(oem)
	} else if (&FileSystemLustre{}).IsType(oem) {
		fsApi = NewFileSystemLustre(oem)
	}
	if fsApi != nil {
		log.Info("New file system", " type ", fsApi.Type())
	}
	return fsApi
}

// FileSystem - Represents an abstract file system, with individual operations
// defined by the underlying FileSystemApi implementation
type FileSystem struct {
	name       string
	devices    []string
	mountpoint string
}

func (*FileSystem) run(cmd string) ([]byte, error) {
	return logging.Cli.Trace(cmd, func(cmd string) ([]byte, error) {
		return exec.Command("bash", "-c", cmd).Output()
	})
}

// File System OEM defines the structure that is expected to be included inside a
// Redfish / Swordfish FileSystemV122FileSystem
type FileSystemOem struct {
	Type string `json:"Type"`
	Name string `json:"Name"`
	// The following are used by Lustre, ignored for others.
	MgsNode    string `json:"MgsNode,omitempty"`
	TargetType string `json:"TargetType,omitempty"`
}
