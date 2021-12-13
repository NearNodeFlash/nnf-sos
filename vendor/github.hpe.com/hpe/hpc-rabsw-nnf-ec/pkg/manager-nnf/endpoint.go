package nnf

import (
	"fmt"

	server "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/manager-server"

	sf "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/rfsf/pkg/models"
)

type Endpoint struct {
	id           string
	name         string
	controllerId uint16
	state        sf.ResourceState

	fabricId string

	// This is the Server Controller used for managing the endpoint
	serverCtrl server.ServerControllerApi

	config         *ServerConfig
	storageService *StorageService
}

func (ep *Endpoint) OdataId() string {
	return fmt.Sprintf("%s/Endpoints/%s", ep.storageService.OdataId(), ep.id)
}
