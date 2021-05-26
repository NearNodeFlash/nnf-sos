package nnf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	nnf "stash.us.cray.com/rabsw/nnf-ec/internal/manager-nnf"
	server "stash.us.cray.com/rabsw/nnf-ec/internal/manager-server"

	openapi "stash.us.cray.com/rabsw/rfsf-openapi/pkg/common"
	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"
)

const (
	StorageServiceRoot = "/redfish/v1/StorageServices/NNF"
)

// NewStorageServiceConnection will create a new connection to the NNF Storage Service. Will return
// the storage service capable of supporting various create and get operations. Will return nil if
// the service cannot be reached.
func NewStorageServiceConnection(address, port string) (*storageService, error) {
	ss := &storageService{
		address: address,
		port:    port,
		client:  http.Client{},
	}

	if _, err := ss.Get(); err != nil {

		var operr *net.OpError
		if errors.As(err, &operr) {
			if operr.Op == "read" { // Connection refused (not ready?)
				return nil, nil
			}
		}

		var dnserr *net.DNSError
		if errors.As(err, &dnserr) {
			if dnserr.IsTemporary || dnserr.IsTimeout || dnserr.IsNotFound {
				return nil, nil
			}
		}

		return nil, err
	}

	return ss, nil
}

type storageService struct {
	address string
	port    string
	client  http.Client
}

func (s *storageService) Get() (*sf.StorageServiceV150StorageService, error) {
	model := new(sf.StorageServiceV150StorageService)
	err := s.get(StorageServiceRoot, model)
	return model, err
}

func (s *storageService) GetCapacity() (*sf.CapacityCapacitySource, error) {
	model := new(sf.CapacityCapacitySource)
	err := s.get(fmt.Sprintf("%s/CapacitySource", StorageServiceRoot), model)

	return model, err
}

func (s *storageService) GetServer(odataid string) (*sf.EndpointV150Endpoint, error) {
	model := new(sf.EndpointV150Endpoint)
	err := s.get(odataid, model)

	return model, err
}

func (s *storageService) GetServers() ([]sf.EndpointV150Endpoint, error) {

	collection := new(sf.EndpointCollectionEndpointCollection)
	err := s.get(fmt.Sprintf("%s/Endpoints", StorageServiceRoot), collection)
	if err != nil {
		return nil, err
	}

	endpoints := make([]sf.EndpointV150Endpoint, len(collection.Members))
	for idx, ref := range collection.Members {
		err = s.get(ref.OdataId, &endpoints[idx])
		if err != nil {
			return nil, err
		}
	}

	return endpoints, nil
}

func (s *storageService) GetStoragePool(id string) (*sf.StoragePoolV150StoragePool, error) {
	model := new(sf.StoragePoolV150StoragePool)
	err := s.get(fmt.Sprintf("%s/StoragePools/%s", StorageServiceRoot, id), model)

	return model, err
}

func (s *storageService) CreateStoragePool(capacityBytes int64) (*sf.StoragePoolV150StoragePool, error) {
	model := new(sf.StoragePoolV150StoragePool)

	model.CapacityBytes = capacityBytes
	model.Oem = openapi.MarshalOem(nnf.AllocationPolicyOem{
		Policy:     nnf.SpareAllocationPolicyType,
		Compliance: nnf.RelaxedAllocationComplianceType,
	})

	err := s.post(fmt.Sprintf("%s/StoragePools", StorageServiceRoot), model)

	return model, err
}

func (s *storageService) CreateStorageGroup(pool *sf.StoragePoolV150StoragePool, endpoint *sf.EndpointV150Endpoint) (*sf.StorageGroupV150StorageGroup, error) {
	model := new(sf.StorageGroupV150StorageGroup)

	model.Links.StoragePool.OdataId = pool.OdataId
	model.Links.ServerEndpoint.OdataId = endpoint.OdataId

	err := s.post(fmt.Sprintf("%s/StorageGroups", StorageServiceRoot), model)

	return model, err
}

func (s *storageService) GetStorageGroup(odataid string) (*sf.StorageGroupV150StorageGroup, error) {
	model := new(sf.StorageGroupV150StorageGroup)
	err := s.get(odataid, model)

	return model, err
}

func (s *storageService) CreateFileSystem(pool *sf.StoragePoolV150StoragePool, fileSystem string) (*sf.FileSystemV122FileSystem, error) {
	model := new(sf.FileSystemV122FileSystem)

	model.Links.StoragePool.OdataId = pool.OdataId
	model.Oem = openapi.MarshalOem(server.FileSystemOem{
		Name: fileSystem,
	})

	err := s.post(fmt.Sprintf("%s/FileSystems", StorageServiceRoot), model)

	return model, err
}

func (s *storageService) GetFileSystem(odataid string) (*sf.FileSystemV122FileSystem, error) {
	model := new(sf.FileSystemV122FileSystem)
	err := s.get(odataid, model)

	return model, err
}

func (s *storageService) CreateFileShare(fileSystem *sf.FileSystemV122FileSystem, endpoint *sf.EndpointV150Endpoint, fileSharePath string) (*sf.FileShareV120FileShare, error) {
	model := new(sf.FileShareV120FileShare)

	model.FileSharePath = fileSharePath
	model.Links.FileSystem.OdataId = fileSystem.OdataId
	model.Links.Endpoint.OdataId = endpoint.OdataId

	err := s.post(fmt.Sprintf("%s/ExportedFileShares", fileSystem.OdataId), model)

	return model, err
}

func (s *storageService) get(path string, model interface{}) error {
	return s.do(http.MethodGet, path, model)
}

func (s *storageService) post(path string, model interface{}) error {
	return s.do(http.MethodPost, path, model)
}

func (s *storageService) do(method string, path string, model interface{}) error {
	url := fmt.Sprintf("http://%s:%s%s", s.address, s.port, path)

	body := []byte{}
	if method == http.MethodPost || method == http.MethodPatch {
		body, _ = json.Marshal(model)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	rsp, err := s.client.Do(req)
	if rsp != nil {
		defer rsp.Body.Close()
	}

	if err != nil {
		return err
	}

	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("Get request failed. Path: %s Status: %d (%s)", path, rsp.StatusCode, rsp.Status)
	}

	if err := json.NewDecoder(rsp.Body).Decode(model); err != nil {
		return err
	}

	return nil
}
