/*
 * Copyright 2020-2024 Hewlett Packard Enterprise Development LP
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

/*
 * Near Node Flash
 *
 * This file contains the declaration of the Near-Node Flash
 * Element Controller.
 *
 * Author: Nate Roiger
 *
 */

package nnf

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"

	ec "github.com/NearNodeFlash/nnf-ec/pkg/ec"
	event "github.com/NearNodeFlash/nnf-ec/pkg/manager-event"
	fabric "github.com/NearNodeFlash/nnf-ec/pkg/manager-fabric"
	msgreg "github.com/NearNodeFlash/nnf-ec/pkg/manager-message-registry"
	nnf "github.com/NearNodeFlash/nnf-ec/pkg/manager-nnf"
	nvme "github.com/NearNodeFlash/nnf-ec/pkg/manager-nvme"
	telemetry "github.com/NearNodeFlash/nnf-ec/pkg/manager-telemetry"
	"github.com/NearNodeFlash/nnf-ec/pkg/persistent"
)

const (
	Name    = "nnf-ec"
	Port    = 50057
	Version = "v2"
)

type Options struct {
	mock                 bool   // Enable mock interfaces for Switches, NVMe, and NNF
	cli                  bool   // Enable CLI commands instead of binary
	persistence          bool   // Enable persistent object storage; used during crash/reboot recovery
	json                 string // Initialize the element controller with the provided json file
	direct               string // Enable direct management of NVMe devices matching this regexp pattern
	InitializeAndExit    bool   // Initialize all controllers then exit without starting the http server (mfg use)
	deleteUnknownVolumes bool   // Delete volumes not represented by a storage pool at the end of initialization
}

func (o *Options) DeleteUnknownVolumes() bool {
	return o.deleteUnknownVolumes
}

func newDefaultOptions() *Options {
	return &Options{mock: false, cli: false, persistence: true}
}

func NewMockOptions(persistence bool) *Options {
	return &Options{mock: true, cli: false, persistence: persistence}
}

func BindFlags(fs *flag.FlagSet) *Options {
	opts := newDefaultOptions()

	fs.BoolVar(&opts.mock, "mock", opts.mock, "Enable mock (simulated) environment.")
	fs.BoolVar(&opts.cli, "cli", opts.cli, "Enable CLI interfaces with devices, instead of raw binary.")
	fs.BoolVar(&opts.persistence, "persistence", opts.persistence, "Enable persistent object storage (used during crash/reboot recovery)")
	fs.StringVar(&opts.json, "json", "", "Initialize database with provided json file")
	fs.StringVar(&opts.direct, "direct", opts.direct, "Enable direct management of NVMe block devices matching this regexp pattern. Implies Mock.")
	fs.BoolVar(&opts.InitializeAndExit, "initializeAndExit", opts.InitializeAndExit, "Initialize all hardware controllers, then exit without starting the http server. Useful in hardware bringup")
	fs.BoolVar(&opts.deleteUnknownVolumes, "deleteUnknownVolumes", opts.deleteUnknownVolumes, "Delete volumes not represented by storage pools")

	nvme.BindFlags(fs)

	return opts
}

// NewController - Create a new NNF Element Controller with the desired mocking behavior
func NewController(opts *Options) *ec.Controller {
	if opts == nil {
		return nil
	}

	switchCtrl := fabric.NewSwitchtecController()
	nvmeCtrl := nvme.NewSwitchtecNvmeController()
	nnfCtrl := nnf.NewNnfController(opts.persistence)

	if opts.cli {
		switchCtrl = fabric.NewSwitchtecCliController()
		nvmeCtrl = nvme.NewCliNvmeController()
	}

	if len(opts.direct) != 0 {
		switchCtrl = fabric.NewMockSwitchtecController()
		nvmeCtrl = nvme.NewDirectDeviceNvmeController(opts.direct)
	} else if opts.mock {
		switchCtrl = fabric.NewMockSwitchtecController()
		nvmeCtrl = nvme.NewMockNvmeController(opts.persistence)

		if _, ok := os.LookupEnv("NNF_SUPPLIED_DEVICES"); ok {
			// Keep the real NnfController.
			log.Infof("NNF_SUPPLIED_DEVICES: %s", os.Getenv("NNF_SUPPLIED_DEVICES"))
		} else {
			nnfCtrl = nnf.NewMockNnfController(opts.persistence)
		}
	}

	if len(opts.json) != 0 {
		persistent.StorageProvider = persistent.NewJsonFilePersistentStorageProvider(opts.json)
	}

	return ec.NewController(Name, Port, Version, NewDefaultApiRouters(switchCtrl, nvmeCtrl, nnfCtrl, opts.deleteUnknownVolumes))
}

// NewDefaultApiRouters -
func NewDefaultApiRouters(switchCtrl fabric.SwitchtecControllerInterface, nvmeCtrl nvme.NvmeController, nnfCtrl nnf.NnfControllerInterface, nnfUnknownVolumes bool) ec.Routers {

	routers := []ec.Router{
		fabric.NewDefaultApiRouter(fabric.NewDefaultApiService(), switchCtrl),
		nvme.NewDefaultApiRouter(nvme.NewDefaultApiService(), nvmeCtrl),
		nnf.NewDefaultApiRouter(nnf.NewDefaultApiService(nnf.NewDefaultStorageService(nnfUnknownVolumes)), nnfCtrl),
		telemetry.NewDefaultApiRouter(telemetry.NewDefaultApiService()),
		event.NewDefaultApiRouter(event.NewDefaultApiService()),
		msgreg.NewDefaultApiRouter(msgreg.NewDefaultApiService()),
	}

	return routers
}
