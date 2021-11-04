package fabric

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"

	sf "stash.us.cray.com/rabsw/nnf-ec/pkg/rfsf/pkg/models"
)

//go:embed config_default.yaml
var configFileDefault []byte

//go:embed config_dp1a.yaml
var configFileDP1a []byte

type ConfigFile struct {
	Version  string
	Metadata struct {
		Name string
	}
	ManagementConfig ManagementConfig `yaml:"managementConfig"`
	Switches         []SwitchConfig

	ManagementPortCount int
	UpstreamPortCount   int
	DownstreamPortCount int
}

type ManagementConfig struct {
	PrimaryDevice   int32 `yaml:"primaryDevice"`
	SecondaryDevice int32 `yaml:"secondaryDevice"`
}

type SwitchConfig struct {
	Id       string
	Metadata struct {
		Name string
	}
	pciGen int
	Ports  []PortConfig

	ManagementPortCount int
	UpstreamPortCount   int
	DownstreamPortCount int
}

type PortConfig struct {
	Id    string
	Name  string
	Type  string
	Port  int
	Width int
}

func loadConfig() (*ConfigFile, error) {

	var config = new(ConfigFile)
	var configFile = findConfig()

	if err := yaml.Unmarshal(configFile, config); err != nil {
		return config, err
	}

	// For usability we convert the port index to a string - this
	// allows for easier comparisons for functions receiving portId
	// as a string. We also tally the number of each port type
	for switchIdx := range config.Switches {
		s := &config.Switches[switchIdx]
		for _, p := range s.Ports {

			switch p.getPortType() {
			case sf.MANAGEMENT_PORT_PV130PT:
				s.ManagementPortCount++
				if s.UpstreamPortCount != 0 {
					return nil, fmt.Errorf("management USP must be listed first in config.yaml")
				}
			case sf.UPSTREAM_PORT_PV130PT:
				s.UpstreamPortCount++
			case sf.DOWNSTREAM_PORT_PV130PT:
				s.DownstreamPortCount++
			case sf.INTERSWITCH_PORT_PV130PT:
				continue
			default:
				return nil, fmt.Errorf("unhandled port type %s in config.yaml", p.Type)
			}
		}

		config.ManagementPortCount += s.ManagementPortCount
		config.UpstreamPortCount += s.UpstreamPortCount
		config.DownstreamPortCount += s.DownstreamPortCount
	}

	// Only a single management endpoint for ALL switches (but unique ports)
	if config.ManagementPortCount != len(config.Switches) {
		if !config.IsManagementRoutingEnabled() {
			return nil, fmt.Errorf("Switch Ports: Expected %d Management Ports, Received: %d", len(config.Switches), config.ManagementPortCount)
		}
	}

	return config, nil
}

func findConfig() []byte {
	// Detect which configuration we are running. DP1a is using port 32, which is a x4 port which
	// maps to all drives (9 locally, 9 over the fabric interconnect). We identify this configuration
	// using the known PCIe topology for DP1a, which has the device listed at address 0b:00.1.
	devicePath := "/sys/bus/pci/devices/0000:0b:00.1/device"
	if _, err := os.Stat(devicePath); !os.IsNotExist(err) {
		device, _ := ioutil.ReadFile(devicePath)
		if string(device) == "0x4200\n" {
			return configFileDP1a
		}
	}

	return configFileDefault
}

// Method returns True if the management configuration has routing enabled, False otherwise.
// Management Routing is when one PAX is controlled through another device - this is needed
// on bringup configurations where there is not a Rabbit-P that is connected to both systems.
func (c *ConfigFile) IsManagementRoutingEnabled() bool {
	return c.ManagementConfig.PrimaryDevice != c.ManagementConfig.SecondaryDevice
}

// func (c *ConfigFile) findSwitch(switchId string) (*SwitchConfig, error) {
// 	for _, s := range c.Switches {
// 		if s.Id == switchId {
// 			return &s, nil
// 		}
// 	}

// 	return nil, fmt.Errorf("Switch %s not found", switchId)
// }

// func (c *ConfigFile) findSwitchPort(switchId string, portId string) (*PortConfig, error) {
// 	s, err := c.findSwitch(switchId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, port := range s.Ports {
// 		if portId == port.Id {
// 			return &port, nil
// 		}
// 	}

// 	return nil, fmt.Errorf("Switch %s Port %s not found", switchId, portId)
// }

func (p *PortConfig) getPortType() sf.PortV130PortType {
	switch p.Type {
	case "InterswitchPort":
		return sf.INTERSWITCH_PORT_PV130PT
	case "UpstreamPort":
		return sf.UPSTREAM_PORT_PV130PT
	case "DownstreamPort":
		return sf.DOWNSTREAM_PORT_PV130PT
	case "ManagementPort":
		return sf.MANAGEMENT_PORT_PV130PT
	default:
		return sf.UNCONFIGURED_PORT_PV130PT
	}
}
