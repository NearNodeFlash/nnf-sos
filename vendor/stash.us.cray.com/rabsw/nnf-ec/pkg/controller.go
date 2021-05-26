/*
 * Near Node Flash
 *
 * This file contains the declaration of the Near-Node Flash
 * Element Controller.
 *
 * Author: Nate Roiger
 *
 * Copyright 2020 Hewlett Packard Enterprise Development LP
 */

package nnf

import (
	"flag"

	ec "stash.us.cray.com/rabsw/ec"

	fabric "stash.us.cray.com/rabsw/nnf-ec/internal/manager-fabric"
	nnf "stash.us.cray.com/rabsw/nnf-ec/internal/manager-nnf"
	nvme "stash.us.cray.com/rabsw/nnf-ec/internal/manager-nvme"
)

const (
	Name = "nnf-ec"
	Port = 50057
	Version = "v2"
)

type Options struct {
	mock bool // Enable mock interfaces for Switches, NVMe, and NNF
	cli  bool // Enable CLI commands instead of binary
}

func newDefaultOptions() *Options {
	return &Options{mock: false, cli: false}
}

func NewMockOptions() *Options {
	return &Options{mock: true, cli: false}
}

func BindFlags(fs *flag.FlagSet) *Options {
	opts := newDefaultOptions()

	fs.BoolVar(&opts.mock, "mock", opts.mock, "Enable mock (simulated) environment.")
	fs.BoolVar(&opts.cli, "cli", opts.cli, "Enable CLI interfaces with devices, instead of raw binary.")

	nvme.BindFlags(fs)

	return opts
}

// NewController - Create a new NNF Element Controller with the desired mocking behavior
func NewController(opts *Options) *ec.Controller {
	if opts == nil {
		// ajf - don't check this in, but I don't yet know how to provide options for unit tests
		//		opts = newDefaultOptions()
		opts = NewMockOptions()
	}

	switchCtrl := fabric.NewSwitchtecController()
	nvmeCtrl := nvme.NewSwitchtecNvmeController()
	nnfCtrl := nnf.NewNnfController()

	if opts.cli {
		switchCtrl = fabric.NewSwitchtecCliController()
		nvmeCtrl = nvme.NewCliNvmeController()
	}

	if opts.mock {
		switchCtrl = fabric.NewMockSwitchtecController()
		nvmeCtrl = nvme.NewMockNvmeController()
		nnfCtrl = nnf.NewMockNnfController()
	}

	return &ec.Controller{
		Name:    Name,
		Port:    Port,
		Version: Version,
		Routers: NewDefaultApiRouters(switchCtrl, nvmeCtrl, nnfCtrl),
	}
}

// NewDefaultApiRouters -
func NewDefaultApiRouters(switchCtrl fabric.SwitchtecControllerInterface, nvmeCtrl nvme.NvmeController, nnfCtrl nnf.NnfControllerInterface) ec.Routers {

	routers := []ec.Router{
		fabric.NewDefaultApiRouter(fabric.NewDefaultApiService(), switchCtrl),
		nvme.NewDefaultApiRouter(nvme.NewDefaultApiService(), nvmeCtrl),
		nnf.NewDefaultApiRouter(nnf.NewDefaultApiService(nnf.NewDefaultStorageService()), nnfCtrl),
	}

	return routers
}
