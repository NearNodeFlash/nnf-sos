/*
 * Copyright 2020, 2021, 2022 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nvme

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/NearNodeFlash/nnf-ec/pkg/persistent"
	"github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/nvme"
)

// NVME Mock Persistence - Handles the various persistent aspects of an NVMe device, specifically namespaces
// and attached controllers. This uses a unique key-value store consisting of a key per device. Each NVMe
// command is saved into the device's ledger. On boot, we recover the database and replay all the actions
// that occurred on the drive.

const (
	mockNvmePersistenceRegistryPrefix = "MOCK_"
)

const (
	mockNvmePersistenceNamespaceCreate uint32 = iota
	mockNvmePersistenceNamespaceDelete
	mockNvmePersistenceAttachController
	mockNvmePersistenceDetachController
)

var errDeviceNotFound = errors.New("Device Not Found")

type MockNvmePersistenceManager struct {
	store   *persistent.Store
	replays []mockNvmePersistenceReplay
}

// Initialize the Mock NVMe Persistent Manager - This opens the key-value store for access and registers
// the Mock NVMe Persistence Registry to handle any Replays in the database.
func (mgr *MockNvmePersistenceManager) initialize() (err error) {

	mgr.store, err = persistent.Open("mock.db", false)
	if err != nil {
		panic(err)
	}

	mgr.store.Register([]persistent.Registry{newMockNvmePersistenceRegistry(mgr)})

	if err := mgr.store.Replay(); err != nil {
		panic(err)
	}

	return
}

// Close -
func (mgr *MockNvmePersistenceManager) close() error {
	return mgr.store.Close()
}

// Load will load the provided device with any recorded events so the device appears to have
// all the NVMe Namespaces with the attached Controllers.
func (mgr *MockNvmePersistenceManager) load(dev *mockDevice) error {

	dev.allocatedCapacity = 0
	for replayIdx := range mgr.replays {
		replay := &mgr.replays[replayIdx]
		if dev.id() == replay.id {
			for _, namespace := range replay.data.namespaces {
				ns := dev.findNamespace(invalidNamespaceId)
				if ns == nil {
					return fmt.Errorf("Unable to allocate free namepace")
				}

				ns.id = nvme.NamespaceIdentifier(namespace.NamespaceId)
				ns.capacity = namespace.Capacity
				ns.guid = namespace.GUID

				for _, ctrlId := range namespace.controllerIds {
					ns.attachedControllers[ctrlId] = &dev.controllers[ctrlId]
				}

				dev.allocatedCapacity += ns.capacity
			}

			return nil
		}
	}

	return errDeviceNotFound
}

// New will create the provided device in the Mock NVMe Persistence database.
func (mgr *MockNvmePersistenceManager) new(dev *mockDevice) error {
	metadata, _ := json.Marshal(&mockNvmeDevicePersistentMetadata{
		FabricId: dev.fabricId,
		SwitchId: dev.switchId,
		PortId:   dev.portId,
		// TODO: Serial Number? Might be useful to confirm.
	})

	ledger, err := mgr.store.NewKey(mockNvmePersistenceRegistryPrefix+dev.id(), metadata)
	if err != nil {
		return err
	}

	return ledger.Close(false)
}

func (mgr *MockNvmePersistenceManager) recordCreateNamespace(dev *mockDevice, ns *mockNamespace) {
	ledger, err := mgr.store.OpenKey(mockNvmePersistenceRegistryPrefix+dev.id())
	if err != nil {
		panic(err)
	}

	data, _ := json.Marshal(&mockNvmeDevicePersistentNamespaceData{
		NamespaceId: uint32(ns.id),
		GUID:        ns.guid,
		Capacity:    ns.capacity,
	})

	if err := ledger.Log(mockNvmePersistenceNamespaceCreate, data); err != nil {
		panic(err)
	}

	ledger.Close(false)
}

func (mgr *MockNvmePersistenceManager) recordDeleteNamespace(dev *mockDevice, ns *mockNamespace) {
	ledger, err := mgr.store.OpenKey(mockNvmePersistenceRegistryPrefix+dev.id())
	if err != nil {
		panic(err)
	}

	data, _ := json.Marshal(&mockNvmeDevicePersistentNamespaceData{
		NamespaceId: uint32(ns.id),
		GUID:        ns.guid,
		Capacity:    0,
	})

	if err := ledger.Log(mockNvmePersistenceNamespaceDelete, data); err != nil {
		panic(err)
	}

	ledger.Close(false)
}

func (mgr *MockNvmePersistenceManager) recordAttachController(dev *mockDevice, ns *mockNamespace, ctrlId uint16) {
	ledger, err := mgr.store.OpenKey(mockNvmePersistenceRegistryPrefix+dev.id())
	if err != nil {
		panic(err)
	}

	data, _ := json.Marshal(&mockNvmeDevicePersistentControllerData{
		NamespaceId:  uint32(ns.id),
		ControllerId: ctrlId,
	})

	if err := ledger.Log(mockNvmePersistenceAttachController, data); err != nil {
		panic(err)
	}

	ledger.Close(false)
}

func (mgr *MockNvmePersistenceManager) recordDetachController(dev *mockDevice, ns *mockNamespace, ctrlId uint16) {
	ledger, err := mgr.store.OpenKey(mockNvmePersistenceRegistryPrefix+dev.id())
	if err != nil {
		panic(err)
	}

	data, _ := json.Marshal(&mockNvmeDevicePersistentControllerData{
		NamespaceId:  uint32(ns.id),
		ControllerId: ctrlId,
	})

	if err := ledger.Log(mockNvmePersistenceDetachController, data); err != nil {
		panic(err)
	}

	ledger.Close(false)
}

// Mock NVME Persistence Registry - Handles database entries prefixed with "MOCK_" string.
type mockNvmePersistenceRegistry struct {
	mgr *MockNvmePersistenceManager
}

func newMockNvmePersistenceRegistry(mgr *MockNvmePersistenceManager) persistent.Registry {
	return &mockNvmePersistenceRegistry{mgr: mgr}
}

func (reg *mockNvmePersistenceRegistry) Prefix() string {
	return mockNvmePersistenceRegistryPrefix
}

func (reg *mockNvmePersistenceRegistry) NewReplay(id string) persistent.ReplayHandler {
	reg.mgr.replays = append(reg.mgr.replays, mockNvmePersistenceReplay{id: id})
	return &reg.mgr.replays[len(reg.mgr.replays)-1]
}

// Mock NVMe Persistence Reply - Handles replays from the Mock NVMe Persistence Registry - that
// is any reply that is prefixed with the parent registries prefix value (i.e. "MOCK_")
type mockNvmePersistenceReplay struct {
	id   string
	data mockNvmeDevicePersistentReplyData
}

func (r *mockNvmePersistenceReplay) Metadata(data []byte) error {
	r.data.id = r.id
	return json.Unmarshal(data, &r.data.metadata)
}

// Entry - Handles each entry logged in the database
func (r *mockNvmePersistenceReplay) Entry(t uint32, data []byte) error {
	switch t {
	case mockNvmePersistenceNamespaceCreate:
		r.data.namespaces = append(r.data.namespaces, mockNvmeDevicePersistentNamespaceReplyData{})
		if err := json.Unmarshal(data, &r.data.namespaces[len(r.data.namespaces)-1]); err != nil {
			return err
		}

	case mockNvmePersistenceNamespaceDelete:
		namespace := &mockNvmeDevicePersistentNamespaceData{}
		if err := json.Unmarshal(data, namespace); err != nil {
			return err
		}

		for nsIdx, ns := range r.data.namespaces {
			if namespace.NamespaceId == ns.NamespaceId {
				copy(r.data.namespaces[nsIdx:], r.data.namespaces[nsIdx+1:])
				r.data.namespaces = r.data.namespaces[:len(r.data.namespaces)-1]
				break
			}
		}

	case mockNvmePersistenceAttachController:
		controller := &mockNvmeDevicePersistentControllerData{}
		if err := json.Unmarshal(data, controller); err != nil {
			return err
		}

		ns := r.findNamespace(controller.NamespaceId)
		ns.controllerIds = append(ns.controllerIds, controller.ControllerId)

	case mockNvmePersistenceDetachController:
		controller := &mockNvmeDevicePersistentControllerData{}
		if err := json.Unmarshal(data, controller); err != nil {
			return err
		}
		ns := r.findNamespace(controller.NamespaceId)

		for ctrlIdIdx, ctrlId := range ns.controllerIds {
			if controller.ControllerId == ctrlId {
				copy(ns.controllerIds[ctrlIdIdx:], ns.controllerIds[ctrlIdIdx+1:])
				ns.controllerIds = ns.controllerIds[:len(ns.controllerIds)-1]
			}
		}
	}

	return nil
}

func (*mockNvmePersistenceReplay) Done() (bool, error) {

	return false, nil
}

func (r *mockNvmePersistenceReplay) findNamespace(id uint32) *mockNvmeDevicePersistentNamespaceReplyData {
	for nsIdx, ns := range r.data.namespaces {
		if ns.NamespaceId == id {
			return &r.data.namespaces[nsIdx]
		}
	}

	panic(fmt.Sprintf("Could not find namespace %d", id))
}

// Mock NVMe Device Persistent Metadata - Data structure that describes the metadata saved for a
// mock NVMe device. This describes the devices location (fabric,switch,port tuple) and the
type mockNvmeDevicePersistentMetadata struct {
	FabricId string `json:"FabricId"`
	SwitchId string `json:"SwitchId"`
	PortId   string `json:"PortId"`
}

// Mock NVMe Device Persistent Namespace Data - Data structure that describes the creation of
// an NVMe Namespace for the parent NVMe device, or the deletion of an NVMe namespace.
type mockNvmeDevicePersistentNamespaceData struct {
	NamespaceId uint32   `json:"NamespaceId"`
	Capacity    uint64   `json:"Capacity"`
	GUID        [16]byte `json:"GUID"`
}

// Mock NVMe Device Persistent Controller Data - Data structure that describes the actions
// of an NVMe Controller over the lifetime of the device. Can be used to track either attach
// or detach actions for a particular NVMe namespace.
type mockNvmeDevicePersistentControllerData struct {
	NamespaceId  uint32 `json:"NamespaceId"`
	ControllerId uint16 `json:"ControllerId"`
}

type mockNvmeDevicePersistentReplyData struct {
	id         string
	metadata   mockNvmeDevicePersistentMetadata
	namespaces []mockNvmeDevicePersistentNamespaceReplyData
}

type mockNvmeDevicePersistentNamespaceReplyData struct {
	mockNvmeDevicePersistentNamespaceData
	controllerIds []uint16
}
