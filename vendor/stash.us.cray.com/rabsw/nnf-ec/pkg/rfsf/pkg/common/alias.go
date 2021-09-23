/*
 * Redfish / Swordfish Defines
 *
 * This contains the defintions of aliasied Redfish / Swordfish models in
 * use for the various services in the openapi package. This provides a
 * single location to define the Redfish / Swordfish models in use.

 * Author: Nate Roiger
 *
 * Copyright 2020 Hewlett Packard Enterprise Development LP
 */

package openapi

import (
	models "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
)

const (
	OK = models.OK_RH
)

const (
	ENABLED = models.ENABLED_RST
)

const (
	ON  = models.ON_RPST
	OFF = models.OFF_RPST
)

// Chassis - The Chassis schema represents the physical components of a system.  This resource represents the sheet-metal confined spaces and logical zones such as racks, enclosures, chassis and all other containers.  Subsystems, such as sensors, that operate outside of a system's data plane are linked either directly or indirectly through this resource.  A subsystem that operates outside of a system's data plane are not accessible to software that runs on the system.
type Chassis = models.ChassisV1140Chassis

// Drive - The Drive schema represents a single physical drive for a system, including links to associated volumes.
type Drive = models.DriveV1110Drive

// DriveLinks - The links to other resources that are related to this resource.
type DriveLinks = models.DriveV1110Links

// EventDestination - The EventDestination schema defines the target of an event subscription, including the event types and context to provide to the target in the Event payload.
type EventDestination = models.EventDestinationV190EventDestination

// EventDestinationCollection - A Collection of EventDestination Resource instances.
type EventDestinationCollection = models.EventDestinationCollectionEventDestinationCollection

// EventService - The EventService schema contains properties for managing event subscriptions and generates the events sent to subscribers.  The resource has links to the actual collection of subscriptions, which are called event destinations.
type EventService = models.EventServiceV170EventService

// ManagerCollection - The collection of Manager resource instances.
type ManagerCollection = models.ManagerCollectionManagerCollection

// Manager - In Redfish, a manager is a systems management entity that can implement or provide access to a Redfish service.  Examples of managers are BMCs, enclosure managers, management controllers, and other subsystems that are assigned manageability functions.  An implementation can have multiple managers, which might be directly accessible through a Redfish-defined interface.
type Manager = models.ManagerV1100Manager

// ManagerActions - The available actions for this resource.
type ManagerActions = models.ManagerV1100Actions

// OdataIdRef - A reference to a resource.
type OdataIdRef = models.OdataV4IdRef

// RedfishError - The error payload from a Redfish Service.
type RedfishError = models.RedfishError

// ResourceBlockCollection - The collection of ResourceBlock resource instances.
type ResourceBlockCollection = models.ResourceBlockCollectionResourceBlockCollection

// ResourceBlock - The ResourceBlock schema contains definitions resource blocks, its components, and affinity to composed devices.
type ResourceBlock = models.ResourceBlockV133ResourceBlock

// ResourceIdentifier - Any additional identifiers for a resource.
type ResourceIdentifier = models.ResourceIdentifier

// ResourcePowerState - Power state of a resource
type ResourcePowerState = models.ResourcePowerState

// ResourceStatus - The status and health of a resource and its children.
type ResourceStatus = models.ResourceStatus

// StoragePool - A container of data storage.
type StoragePool = models.StoragePoolV150StoragePool

// StorageServices - Collection of resources that are managed and exposed to hosts as a group.
type StorageServices = models.StorageServiceV150StorageService

// VolumeCollection - A Collection of Volume resource instances.
type VolumeCollection = models.VolumeCollectionVolumeCollection

// Volume - Volume contains properties used to describe a volume, virtual disk, LUN, or other logical storage entity for any system.
type Volume = models.VolumeV161Volume
