package server

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/logging"

	"github.hpe.com/hpe/hpc-rabsw-nnf-ec/internal/switchtec/pkg/nvme"
)

type DefaultServerControllerProvider struct{}

func (DefaultServerControllerProvider) NewServerController(opts ServerControllerOptions) ServerControllerApi {
	if opts.Local {
		return &LocalServerController{}
	}
	return NewRemoteServerController(opts)
}

type LocalServerController struct {
	storage []Storage
}

func (c *LocalServerController) Connected() bool { return true }

func (c *LocalServerController) GetServerInfo() ServerInfo {
	serverInfo := ServerInfo{}

	output, err := c.run("lctl list_nids")
	if err != nil {
		return serverInfo
	}

	for _, nid := range strings.Split(string(output), "\n") {
		if strings.Contains(nid, "@") {
			serverInfo.LNetNids = append(serverInfo.LNetNids, nid)
		}
	}

	return serverInfo
}

func (c *LocalServerController) NewStorage(pid uuid.UUID, expectedNamespaces []StorageNamespace) *Storage {
	c.storage = append(c.storage, Storage{
		Id:                   pid,
		expectedNamespaces:   expectedNamespaces,
		discoveredNamespaces: make([]StorageNamespace, 0),
		ctrl:                 c,
	})
	return &c.storage[len(c.storage)-1]
}

func (c *LocalServerController) Delete(s *Storage) error {
	for storageIdx, storage := range c.storage {
		if storage.Id == s.Id {
			c.storage = append(c.storage[:storageIdx], c.storage[storageIdx+1:]...)
			break
		}
	}

	return nil
}

func (c *LocalServerController) GetStatus(s *Storage) (StorageStatus, error) {
	if len(s.discoveredNamespaces) < len(s.expectedNamespaces) {
		if err := c.Discover(s); err != nil {
			return StorageStatus_Error, err
		}
	}

	if len(s.discoveredNamespaces) < len(s.expectedNamespaces) {
		return StorageStatus_Starting, nil
	}

	return StorageStatus_Ready, nil
}

func (c *LocalServerController) CreateFileSystem(s *Storage, fs FileSystemApi, opts FileSystemOptions) error {
	s.fileSystem = fs

	if err := fs.Create(s.Devices(), opts); err != nil {
		return err
	}

	return nil
}

func (c *LocalServerController) DeleteFileSystem(s *Storage) error {
	return s.fileSystem.Delete()
}

func (c *LocalServerController) MountFileSystem(s *Storage, opts FileSystemOptions) error {
	mountPoint := opts["mountpoint"].(string)
	if mountPoint == "" {
		return nil
	}

	if err := s.fileSystem.Mount(mountPoint); err != nil {
		return err
	}

	return nil
}

func (c *LocalServerController) UnmountFileSystem(s *Storage) error {
	if err := s.fileSystem.Unmount(); err != nil {
		return err
	}

	return nil
}

func (c *LocalServerController) Discover(s *Storage) error {
	var err error
	if devices, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
		err = c.discoverUsingSuppliedDevices(s, devices)
	} else {
		err = c.discoverViaNamespaces(s)
	}
	return err
}

func (c *LocalServerController) discoverUsingSuppliedDevices(s *Storage, devices string) error {

	// All devices will be placed into the same pool.
	devList := strings.Split(devices, ",")
	for idx, devSupplied := range devList {
		log.Info("Supplied device ", " idx ", idx, " dev ", devSupplied)
		byBytes := []byte(devSupplied)
		// Use the last 8 bytes as the unique Id.
		offset := len(byBytes) - 8
		if len(byBytes) < 8 {
			offset = 0
		}
		var myUuid uuid.UUID
		copy(myUuid[:], byBytes[offset:])
		guid := nvme.NamespaceGloballyUniqueIdentifier{}
		guid.Parse(myUuid.String())

		s.discoveredNamespaces = append(s.discoveredNamespaces, StorageNamespace{
			Id:    nvme.NamespaceIdentifier(idx),
			Guid:  guid,
			path:  devSupplied,
			debug: false,
		})
	}

	s.expectedNamespaces = s.discoveredNamespaces

	return nil
}

func (c *LocalServerController) discoverViaNamespaces(s *Storage) error {
	paths, err := c.namespaces()
	if err != nil {
		return err
	}

	for _, path := range paths {

		sns, err := c.newStorageNamespace(path)
		if err != nil {
			return err
		}

		discovered := false
		for _, ns := range s.discoveredNamespaces {
			if ns.Equals(sns) {
				discovered = true
				break
			}
		}

		if discovered {
			continue
		}

		expected := false
		for _, ns := range s.expectedNamespaces {
			if ns.Equals(sns) {
				expected = true
				break
			}
		}

		if expected {
			log.Infof("Discovered namespace %s", sns.String())
			s.discoveredNamespaces = append(s.discoveredNamespaces, *sns)
		}
	}

	return nil
}

func (c *LocalServerController) findStorage(pid uuid.UUID) *Storage {
	for idx, p := range c.storage {
		if p.Id == pid {
			return &c.storage[idx]
		}
	}

	return nil
}

func (c *LocalServerController) namespaces() ([]string, error) {
	nss, err := c.run("ls -A1 /dev/nvme* | { grep -E 'nvme[0-9]+n[0-9]+' || :; }")

	// The above will return an err if zero namespaces exist. In
	// this case, ENOENT is returned and we should instead return
	// a zero length array.
	if exit, ok := err.(*exec.ExitError); ok {
		if syscall.Errno(exit.ExitCode()) == syscall.ENOENT {
			return make([]string, 0), nil
		}
	}

	return strings.Fields(string(nss)), err
}

func (c *LocalServerController) newStorageNamespace(path string) (*StorageNamespace, error) {

	// First we need to identify the NSID for the provided path. The output is something like
	// "NVME Identify Namespace 1:" so we awk the 4th value, which gives us "1:", then trim
	// the colon below.
	nsidStr, err := c.run(fmt.Sprintf("nvme id-ns %s | awk '/NVME Identify Namespace/{printf $4}'", path))
	if err != nil {
		return nil, err
	}

	nsid, _ := strconv.Atoi(strings.TrimSuffix(string(nsidStr[:]), ":"))

	// Retrieve the namespace GUID. The output is something like "nguid   : 00000000000000008ce38ee205e70401"
	// so we awk the third parameter.
	guidStr, err := c.run(fmt.Sprintf("nvme id-ns %s --namespace-id=%d | awk '/nguid/{printf $3}'", path, nsid))
	if err != nil {
		return nil, err
	}

	guid := nvme.NamespaceGloballyUniqueIdentifier{}
	guid.Parse(string(guidStr))

	return &StorageNamespace{
		Id:   nvme.NamespaceIdentifier(nsid),
		Guid: guid,
		path: path,
	}, nil
}

func (c *LocalServerController) run(cmd string) ([]byte, error) {
	return logging.Cli.Trace(cmd, func(cmd string) ([]byte, error) {
		return exec.Command("bash", "-c", cmd).Output()
	})
}
