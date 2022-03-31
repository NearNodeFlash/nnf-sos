package server

import (
	"bytes"
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"

	"github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/logging"
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
	Unmount() error
}

// FileSystem - Represents an abstract file system, with individual operations
// defined by the underlying FileSystemApi implementation
type FileSystem struct {
	name       string
	devices    []string
	mountpoint string
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
	return logging.Cli.Trace(cmd, func(cmd string) ([]byte, error) {

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

// File System OEM defines the structure that is expected to be included inside a
// Redfish / Swordfish FileSystemV122FileSystem
type FileSystemOem struct {
	Type string `json:"Type"`
	Name string `json:"Name"`
	// The following are used by Lustre, ignored for others.
	MgsNode    string `json:"MgsNode,omitempty"`
	TargetType string `json:"TargetType,omitempty"`
	Index      int    `json:"Index,omitempty"`
	BackFs     string `json:"BackFs,omitempty"`
	// The following is used by GFS2, ignored for others.
	ClusterName string `json:"ClusterName,omitempty"`
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
