package server

import (
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"

	"stash.us.cray.com/rabsw/nnf-ec/internal/logging"
)

type FileSystemControllerApi interface {
	NewFileSystem(string) FileSystemApi
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
	}

	FileSystemController = NewFileSystemController(config)

	return nil
}

// FileSystemApi - Defines the interface for interacting with various file systems
// supported by the NNF element controller.
type FileSystemApi interface {
	Name() string

	Create(devices []string, opts FileSystemCreateOptions) error
	Delete() error
}

// NewFileSystem - Will allocate a new File System defined by the provided
// 'name', or nil if no file system of that name can be found.
func (c *fileSystemController) NewFileSystem(name string) FileSystemApi {
	for _, fs := range c.config.FileSystems {
		if fs.Name == strings.ToLower(name) {
			return &FileSystem{
				name: fs.Name,
				ops:  fs.Operations,
			}
		}
	}

	return nil
}

// FileSystem - Represents an abstract file system, with individual operations
// defiend by the underlying FileSystemApi implementation
type FileSystem struct {
	name     string
	commands map[string]string
	ops      []FileSystemOperationConfig
}

type FileSystemCreateOptions = map[string]string

func (fs *FileSystem) Name() string { return fs.name }

func (fs *FileSystem) Create(devices []string, opts FileSystemCreateOptions) error {

	op, err := fs.findOperation("create")
	if err != nil {
		return err
	}

	cmd := op.make(map[string]string{
		"name":    fs.name,
		"devices": strings.Join(devices, " "),
	}, &opts)

	if _, err := fs.run(cmd); err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) Delete() error {
	op, err := fs.findOperation("delete")
	if err != nil {
		return err
	}

	cmd := op.make(map[string]string{
		"name": fs.name,
	}, nil)

	if _, err := fs.run(cmd); err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) findOperation(cmd string) (*FileSystemOperationConfig, error) {
	for idx, op := range fs.ops {
		if op.Label == cmd {
			return &fs.ops[idx], nil
		}
	}

	return nil, NewOperationNotFoundError(cmd)
}

func (*FileSystem) run(cmd string) ([]byte, error) {
	return logging.Cli.Trace(cmd, func(cmd string) ([]byte, error) {
		return exec.Command("bash", "-c", cmd).Output()
	})
}

// Make takes a list of arguments and options for the particular operation.
// The returned format is a string taking the format
// [cmd] [optional args] [required args]
//
func (conf *FileSystemOperationConfig) make(args map[string]string, opts *map[string]string) string {
	var builder strings.Builder

	// Write the command first
	builder.WriteString(conf.Command)

	// Options are placed after the command, prior any arguments
	for _, option := range conf.Options {
		for key, opt := range option {
			if val, ok := (*opts)[key]; ok {

				builder.WriteString(" ")

				key := fmt.Sprintf("{%s}", key)
				fmt.Fprint(&builder, strings.ReplaceAll(opt, key, val))

				delete(*opts, key)
			}
		}
	}

	// Itemized arguments are placed at the end of the command, in order as
	// they appear in the configuration.
	for _, arg := range conf.Arguments {

		// Arguments are tokenized by brackets, if present.
		param := strings.Trim(arg, "{}")

		builder.WriteString(" ")
		if param == arg {
			builder.WriteString(arg)
		} else {
			if val, ok := args[param]; ok {
				builder.WriteString(strings.Replace(arg, arg, val, 1))
				delete(args, param)
			} else {
				// TODO: Argument not found, what to do?
			}

		}
	}

	if len(args) == 0 {
		return builder.String()
	}

	// Common arguments are placed at the end, in non-specific order
	// i.e. zpool export {name}
	//      {name} is replaced by the arg[name] value
	cmd := builder.String()
	for key, val := range args {
		wildcard := fmt.Sprintf("{%s}", key)
		cmd = strings.ReplaceAll(cmd, wildcard, val)
	}

	// TODO: No more wildcards should be in the command string
	return cmd
}

type FileSystemError struct {
	cmd    string
	reason string
}

func (err *FileSystemError) Error() string {
	return fmt.Sprintf("%s Failed: %s", err.cmd, err.reason)
}

func NewOperationNotFoundError(cmd string) error {
	return &FileSystemError{cmd: cmd, reason: fmt.Sprintf("Operation '%s' Not Found", cmd)}
}

// File System OEM defines the structure that is expected to be included inside a
// Redfish / Swordfish FileSystemV122FileSystem
type FileSystemOem struct {
	Name string `json:"Name"`
}
