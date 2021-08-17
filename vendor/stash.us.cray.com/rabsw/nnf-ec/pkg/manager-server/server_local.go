package server

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"stash.us.cray.com/rabsw/nnf-ec/pkg/common"
	"stash.us.cray.com/rabsw/nnf-ec/pkg/logging"

	"stash.us.cray.com/rabsw/nnf-ec/pkg/manager-server/nvme"
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

func (c *LocalServerController) NewStorage(pid uuid.UUID) *Storage {
	c.storage = append(c.storage, Storage{Id: pid, ctrl: c})
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

func (c *LocalServerController) GetStatus(s *Storage) StorageStatus {

	// We really shouldn't need to refresh on every GetStatus() call if we're correctly
	// tracking udev add/remove events. There should be a single refresh on launch (or
	// possibily a udev-info call to pull in the initial hardware?)
	if err := c.Discover(s, nil); err != nil {
		logrus.WithError(err).Errorf("Local Server Controller: Discovery Error")
		return StorageStatus_Error
	}

	// There should always be 1 or more namespces, so zero namespaces mean we are still starting.
	// Once we've recovered the expected number of namespaces (nsExpected > 0) we continue to
	// return a starting status until all namespaces are available.

	// TODO: We should check for an error status somewhere... as this stands we are not
	// ever going to return an error if refresh() fails.
	if (len(s.ns) == 0) ||
		((s.nsExpected > 0) && (len(s.ns) < s.nsExpected)) {
		return StorageStatus_Starting
	}

	if s.nsExpected == len(s.ns) {
		return StorageStatus_Ready
	}

	return StorageStatus_Error
}

func (c *LocalServerController) CreateFileSystem(s *Storage, fs FileSystemApi, opts FileSystemOptions) error {
	s.fileSystem = fs

	if err := fs.Create(s.Devices(), opts); err != nil {
		return err
	}

	return fs.Mount(opts["mountpoint"].(string))
}

func (c *LocalServerController) DeleteFileSystem(s *Storage) error {
	if err := s.fileSystem.Unmount(); err != nil {
		return err
	}

	return s.fileSystem.Delete()
}

func (c *LocalServerController) Discover(s *Storage, newStorageFunc func(*Storage)) error {
	var err error
	if devices, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
		err = c.discoverUsingSuppliedDevices(s, devices)
	} else {
		err = c.discoverViaNamespaces(newStorageFunc)
	}
	return err
}

func (c *LocalServerController) discoverUsingSuppliedDevices(s *Storage, devices string) error {

	// All devices will be placed into the same pool.
	devList := strings.Split(devices, ",")
	for idx, devSupplied := range devList {
		logrus.Info("Supplied device ", " idx ", idx, " dev ", devSupplied)
		byBytes := []byte(devSupplied)
		// Use the last 8 bytes as the unique Id.
		offset := len(byBytes) - 8
		if len(byBytes) < 8 {
			offset = 0
		}
		var myUuid uuid.UUID
		copy(myUuid[:], byBytes[offset:])
		sns := &StorageNamespace{
			id:        myUuid,
			path:      devSupplied,
			nsid:      idx,
			poolId:    s.Id,
			poolIdx:   idx,
			poolTotal: len(devList),
		}

		s.UpsertStorageNamespace(sns)
	}
	return nil
}

func (c *LocalServerController) discoverViaNamespaces(newStorageFunc func(*Storage)) error {
	nss, err := c.namespaces()
	if err != nil {
		return err
	}

	for _, ns := range nss {
		sns, err := c.newStorageNamespace(ns)

		if errors.Is(err, common.ErrNamespaceMetadata) {
			continue
		} else if err != nil {
			return err
		}

		s := c.findStorage(sns.poolId)
		if s == nil {
			s = c.NewStorage(sns.poolId)

			s.nsExpected = sns.poolTotal

			s.ns = append(s.ns, *sns)

			if newStorageFunc != nil {
				newStorageFunc(s)
			}

		} else {

			// We've identified a pool for this particular namespace
			// Add the namespace to the pool if it's not present.
			s.UpsertStorageNamespace(sns)

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

	// First we need to identify the NSID for the provided path.
	nsid, err := nvme.GetNamespaceId(path)
	if err != nil {
		return nil, err
	}

	// Retrieve the namespace GUID
	guidStr, err := c.run(fmt.Sprintf("nvme id-ns %s --namespace-id=%d | awk '/nguid/{printf $3}'", path, nsid))
	if err != nil {
		return nil, err
	}

	id, err := uuid.ParseBytes(guidStr)
	if err != nil {
		return nil, err
	}

	data, err := c.run(fmt.Sprintf("nvme get-feature %s --namespace-id=%d --feature-id=0x7F --raw-binary", path, nsid))
	if err != nil {
		return nil, err
	}

	meta, err := common.DecodeNamespaceMetadata(data[6:])
	if err != nil {
		return nil, err
	}

	return &StorageNamespace{
		id:        id,
		path:      path,
		nsid:      nsid,
		poolId:    meta.Id,
		poolIdx:   int(meta.Index),
		poolTotal: int(meta.Count),
	}, nil
}

func (c *LocalServerController) run(cmd string) ([]byte, error) {
	return logging.Cli.Trace(cmd, func(cmd string) ([]byte, error) {
		return exec.Command("bash", "-c", cmd).Output()
	})
}
