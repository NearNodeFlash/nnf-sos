package nvme

import (
	"stash.us.cray.com/rabsw/nnf-ec/internal/switchtec/pkg/nvme"
)

type NvmeController interface {
	NewNvmeDeviceController() NvmeDeviceController
}

type NvmeDeviceController interface {
	Initialize() error
	Close() error
	
	NewNvmeDevice(fabricId, switchId, portId string) (NvmeDeviceApi, error)
}

// NvmeDeviceApi -
type NvmeDeviceApi interface {
	IdentifyController(controllerId uint16) (*nvme.IdCtrl, error)
	IdentifyNamespace(namespaceId nvme.NamespaceIdentifier) (*nvme.IdNs, error)

	ListSecondary() (*nvme.SecondaryControllerList, error)

	AssignControllerResources(
		controllerId uint16,
		resourceType SecondaryControllerResourceType,
		numResources uint32) error

	OnlineController(controllerId uint16) error

	ListNamespaces(controllerId uint16) ([]nvme.NamespaceIdentifier, error)
	ListAttachedControllers(namespaceId nvme.NamespaceIdentifier) ([]uint16, error)

	CreateNamespace(capacityBytes uint64, metadata []byte) (nvme.NamespaceIdentifier, error)
	DeleteNamespace(namespaceId nvme.NamespaceIdentifier) error

	AttachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error
	DetachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error

	SetNamespaceFeature(namespaceId nvme.NamespaceIdentifier, data []byte) error
	GetNamespaceFeature(namespaceId nvme.NamespaceIdentifier) ([]byte, error)
}

// SecondaryControllersInitFunc -
type SecondaryControllersInitFunc func(count uint8)

// SecondaryControllerHandlerFunc -
type SecondaryControllerHandlerFunc func(controllerId uint16, controllerOnline bool, virtualFunctionNumber uint16, numVQResourcesAssinged, numVIResourcesAssigned uint32) error

// SecondaryControllerResourceType -
type SecondaryControllerResourceType int

const (
	VQResourceType SecondaryControllerResourceType = iota
	VIResourceType
)
