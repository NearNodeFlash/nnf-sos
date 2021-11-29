package server

import "github.com/google/uuid"

type MockServerControllerProvider struct{}

func (MockServerControllerProvider) NewServerController(ServerControllerOptions) ServerControllerApi {
	return NewMockServerController()
}

func NewMockServerController() ServerControllerApi {
	return &MockServerController{}
}

type MockServerController struct {
	storage []Storage
}

func (c *MockServerController) Connected() bool { return true }

func (m *MockServerController) GetServerInfo() ServerInfo {
	return ServerInfo{
		LNetNids: []string{"1.2.3.4@tcp0"},
	}
}

func (c *MockServerController) NewStorage(pid uuid.UUID, expectedNamespaces []StorageNamespace) *Storage {
	c.storage = append(c.storage, Storage{
		Id:                 pid,
		expectedNamespaces: expectedNamespaces,
		ctrl:               c,
	})
	return &c.storage[len(c.storage)-1]
}

func (c *MockServerController) Delete(s *Storage) error {
	for storageIdx, storage := range c.storage {
		if storage.Id == s.Id {
			c.storage = append(c.storage[:storageIdx], c.storage[storageIdx+1:]...)
			break
		}
	}

	return nil
}

func (*MockServerController) GetStatus(s *Storage) (StorageStatus, error) {
	return StorageStatus_Ready, nil
}

func (*MockServerController) CreateFileSystem(s *Storage, fs FileSystemApi, opts FileSystemOptions) error {
	s.fileSystem = fs

	if fs.IsMockable() {
		if err := fs.Create(s.Devices(), opts); err != nil {
			return err
		}

		mountPoint := opts["mountpoint"].(string)
		if mountPoint == "" {
			return nil
		}

		return fs.Mount(mountPoint)
	}

	return nil
}

func (*MockServerController) DeleteFileSystem(s *Storage) error {
	if s.fileSystem.IsMockable() {
		if err := s.fileSystem.Unmount(); err != nil {
			return err
		}

		return s.fileSystem.Delete()
	}
	return nil
}
