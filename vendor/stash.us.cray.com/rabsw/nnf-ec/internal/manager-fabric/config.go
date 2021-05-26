package fabric

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v2"

	sf "stash.us.cray.com/rabsw/rfsf-openapi/pkg/models"
)

//go:embed config.yaml
var configFile []byte

type ConfigFile struct {
	Version  string
	Metadata struct {
		Name string
	}
	Switches []SwitchConfig

	ManagementPortCount int
	UpstreamPortCount   int
	DownstreamPortCount int
}

type SwitchConfig struct {
	Id       string
	Metadata struct {
		Name string
	}
	Ports []PortConfig

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
		return nil, fmt.Errorf("isconfigured Switch Ports: Expected %d Management Ports, Received: %d", len(config.Switches), config.ManagementPortCount)
	}

	return config, nil
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
