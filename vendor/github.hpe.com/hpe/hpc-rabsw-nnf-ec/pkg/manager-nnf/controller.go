package nnf

import (
	server "github.hpe.com/hpe/hpc-rabsw-nnf-ec/pkg/manager-server"
)

type NnfControllerInterface interface {
	ServerControllerProvider() server.ServerControllerProvider
	PersistentControllerProvider() PersistentControllerApi
}

type NnfController struct {
	serverControllerProvider server.ServerControllerProvider
	persistentController     PersistentControllerApi
}

func NewNnfController(persistence bool) NnfControllerInterface {
	return &NnfController{
		server.DefaultServerControllerProvider{},
		getPersistentControllerApi(persistence),
	}
}

func (c *NnfController) ServerControllerProvider() server.ServerControllerProvider {
	return c.serverControllerProvider
}

func (c *NnfController) PersistentControllerProvider() PersistentControllerApi {
	return c.persistentController
}

// TODO: See if mock makes sense here - really the lower-layers should provide the mocking
// of physical devices; NNF Controller might be fine building off that.
func NewMockNnfController(persistence bool) NnfControllerInterface {
	return &NnfController{
		serverControllerProvider: server.MockServerControllerProvider{},
		persistentController:     getPersistentControllerApi(persistence),
	}
}

func getPersistentControllerApi(persistence bool) PersistentControllerApi {
	if persistence {
		return NewDefaultPersistentController()
	}
	return NewMockPersistentController()
}
