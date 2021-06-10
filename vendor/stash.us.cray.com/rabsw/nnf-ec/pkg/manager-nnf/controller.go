package nnf

import (
	server "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-server"
)

type NnfControllerInterface interface {
	ServerControllerProvider() server.ServerControllerProvider
}

type NnfController struct {
	serverControllerProvider server.ServerControllerProvider
}

func NewNnfController() NnfControllerInterface {
	return &NnfController{server.DefaultServerControllerProvider{}}
}

func (c *NnfController) ServerControllerProvider() server.ServerControllerProvider {
	return c.serverControllerProvider
}

// TODO: See if mock makes sense here - really the lower-layers should provide the mocking
// of physical devices; NNF Controller might be fine building off that.
func NewMockNnfController() NnfControllerInterface {
	return &NnfController{serverControllerProvider: server.MockServerControllerProvider{}}
}
