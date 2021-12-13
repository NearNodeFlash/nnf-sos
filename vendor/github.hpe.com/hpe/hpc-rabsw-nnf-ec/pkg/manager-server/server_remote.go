package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	openapi "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/common"
	sf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/models"
)

const (
	RemoteStorageServiceId   = "NNFServer"
	RemoteStorageServicePort = 60050
)

type StoragePoolOem struct {
	Namespaces []StorageNamespace `json:"Namespaces"`
}

type RemoteServerController struct {
	client  http.Client
	Address string
}

func NewRemoteServerController(opts ServerControllerOptions) ServerControllerApi {
	return &RemoteServerController{
		Address: opts.Address,
		client:  http.Client{Timeout: 0},
	}
}

func (c *RemoteServerController) Connected() bool {
	_, err := c.client.Get(c.Url(""))
	return err == nil
}

func (c *RemoteServerController) GetServerInfo() ServerInfo {
	return ServerInfo{}
}

func (c *RemoteServerController) NewStorage(pid uuid.UUID, expectedNamespaces []StorageNamespace) *Storage {

	model := sf.StoragePoolV150StoragePool{
		Id:  pid.String(),
		Oem: openapi.MarshalOem(&StoragePoolOem{Namespaces: expectedNamespaces}),
	}

	req, _ := json.Marshal(model)

	rsp, err := c.client.Post(
		c.Url("/StoragePools"),
		"application/json",
		bytes.NewBuffer(req),
	)

	if rsp != nil {
		defer rsp.Body.Close()
	}

	if err != nil {
		log.WithError(err).Errorf("New Server Storage: Http Error")
		return nil
	}

	if err := json.NewDecoder(rsp.Body).Decode(&model); err != nil {
		log.WithError(err).Errorf("New Server Storage: Failed to decode JSON response")
		return nil
	}

	return &Storage{
		Id:   pid,
		ctrl: c,
	}
}

func (c *RemoteServerController) Delete(s *Storage) error {
	req, err := http.NewRequest(http.MethodDelete, c.Url(fmt.Sprintf("/StoragePools/%s", s.Id.String())), nil)
	if err != nil {
		return err
	}

	rsp, err := c.client.Do(req)
	if rsp != nil {
		defer rsp.Body.Close()
	}

	if err != nil {
		return err
	}

	return nil
}

func (c *RemoteServerController) GetStatus(s *Storage) (StorageStatus, error) {

	rsp, err := c.client.Get(
		c.Url(fmt.Sprintf("/StoragePools/%s", s.Id.String())),
	)
	if rsp != nil {
		defer rsp.Body.Close()
	}

	if err != nil {
		log.WithError(err).Errorf("Get Status: Http Error")
		return StorageStatus_Error, err
	}

	model := sf.StoragePoolV150StoragePool{}
	if err := json.NewDecoder(rsp.Body).Decode(&model); err != nil {
		log.WithError(err).Errorf("Get Status: Failed to decode JSON response")
		return StorageStatus_Error, err
	}

	switch model.Status.State {
	case sf.STARTING_RST:
		return StorageStatus_Starting, nil
	case sf.ENABLED_RST:
		return StorageStatus_Ready, nil
	default:
		return StorageStatus_Error, nil
	}
}

func (c *RemoteServerController) CreateFileSystem(s *Storage, f FileSystemApi, opts FileSystemOptions) error {

	model := sf.FileSystemV122FileSystem{
		StoragePool: sf.OdataV4IdRef{OdataId: fmt.Sprintf("/redfish/v1/StorageServices/%s/StoragePools/%s", RemoteStorageServiceId, s.Id.String())},
		Oem:         map[string]interface{}{"FileSystem": FileSystemOem{Type: f.Type(), Name: f.Name()}},
	}

	req, err := json.Marshal(model)
	if err != nil {
		return err
	}

	rsp, err := c.client.Post(
		c.Url("/FileSystems"), // /redfish/v1/StorageServices/{StorageServiceId}/FileSystems
		"application/json",
		bytes.NewBuffer(req),
	)

	if rsp != nil {
		defer rsp.Body.Close()
	}

	if err != nil {
		log.WithError(err).Errorf("Create File System: Http Error")
		return err
	}

	if err := json.NewDecoder(rsp.Body).Decode(&model); err != nil {
		log.WithError(err).Errorf("Create File System: Failed to decode JSON response")
		return err
	}

	return c.createMountPoint(&model, opts)
}

func (c *RemoteServerController) createMountPoint(fs *sf.FileSystemV122FileSystem, opts FileSystemOptions) error {

	mp, ok := opts["mountpoint"].(string)
	if !ok {
		return fmt.Errorf("Mountpoint not present in controller options")
	}

	model := sf.FileShareV120FileShare{
		FileSharePath: mp,
		Oem:           opts,
	}

	req, _ := json.Marshal(model)

	rsp, err := c.client.Post(
		c.Url(fmt.Sprintf("/FileSystems/%s/ExportedFileShares", fs.Id)),
		"application/json",
		bytes.NewBuffer(req),
	)

	if rsp != nil {
		defer rsp.Body.Close()
	}

	if err != nil {
		log.WithError(err).Errorf("Create Mount Point: Http Error")
		return err
	}

	if err := json.NewDecoder(rsp.Body).Decode(&model); err != nil {
		log.WithError(err).Errorf("Create Mount Point: Failed to decode JSON response")
		return err
	}

	return nil
}

func (r *RemoteServerController) DeleteFileSystem(s *Storage) error {
	return nil
}

func (r *RemoteServerController) Url(path string) string {
	return fmt.Sprintf("http://%s:%d/redfish/v1/StorageServices/%s%s", r.Address, RemoteStorageServicePort, RemoteStorageServiceId, path)
}
