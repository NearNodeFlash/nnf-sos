/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
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
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/NearNodeFlash/nnf-sos/pkg/command"
	"github.com/go-logr/logr"
)

type NvmeDevice struct {
	DevicePath string
	NSID       uint32
	NQN        string
}

type nvmeListVerboseNamespaces struct {
	Device string `json:"NameSpace"`
	NSID   uint32 `json:"NSID"`
}

type nvmeListVerboseControllers struct {
	Namespaces []nvmeListVerboseNamespaces `json:"Namespaces"`
}

type nvmeListVerboseDevice struct {
	SubsystemNQN string                       `json:"SubsystemNQN"`
	Controllers  []nvmeListVerboseControllers `json:"Controllers"`
}

type nvmeListVerboseDevices struct {
	Devices []nvmeListVerboseDevice `json:"Devices"`
}

func NvmeListDevices(log logr.Logger) ([]NvmeDevice, error) {
	devices := []NvmeDevice{}

	data, err := command.Run("nvme list -v --output-format=json", log)
	if err != nil {
		return nil, err
	}

	foundDevices := nvmeListVerboseDevices{}
	if err := json.Unmarshal([]byte(data), &foundDevices); err != nil {
		return nil, err
	}

	for _, device := range foundDevices.Devices {
		for _, controller := range device.Controllers {
			for _, namespace := range controller.Namespaces {
				devices = append(devices, NvmeDevice{DevicePath: "/dev/" + namespace.Device, NSID: namespace.NSID, NQN: device.SubsystemNQN})
			}
		}
	}

	return devices, nil
}

func NvmeRescanDevices(log logr.Logger) error {
	devices, err := ioutil.ReadDir("/dev/")
	if err != nil {
		return fmt.Errorf("could not read /dev: %w", err)
	}

	nvmeRegex, _ := regexp.Compile("nvme[0-9]+$")
	for _, device := range devices {
		if match := nvmeRegex.MatchString(device.Name()); match {
			nvmeDevice := "/dev/" + device.Name()
			if _, err := command.Run("nvme ns-rescan "+nvmeDevice, log); err != nil {
				return fmt.Errorf("could not rescan NVMe device: %w", err)
			}
		}
	}

	return nil
}
