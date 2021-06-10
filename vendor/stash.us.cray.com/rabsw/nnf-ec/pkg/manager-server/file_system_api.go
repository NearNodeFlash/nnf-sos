package server

import (
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"

	"stash.us.cray.com/rabsw/nnf-ec/pkg/logging"
)

type FileSystemControllerApi interface {
	NewFileSystem(typ string, name string) FileSystemApi
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
	Name() string

	Create(devices []string, opts FileSystemOptions) error
	Delete() error

	Mount(mountpoint string) error
	Unmount() error
}

// NewFileSystem -
func (c *fileSystemController) NewFileSystem(typ string, name string) FileSystemApi {

	switch strings.ToLower(typ) {
	case (&FileSystemZfs{}).Type():
		return NewFileSystemZfs(name)
	case (&FileSystemLvm{}).Type():
		return NewFileSystemLvm(name)
	}

	return nil
}

// FileSystem - Represents an abstract file system, with individual operations
// defiend by the underlying FileSystemApi implementation
type FileSystem struct {
	name string
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
}
