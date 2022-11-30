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
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"regexp"
)

// DeviceList returns the devices that match the provided regexp in Model Number, Serial Number,
// or Node Qualifying Name (NQN). Returned paths are of the form /dev/nvme[0-9]+.
func DeviceList(r string) ([]string, error) {

	deviceRegexp, err := regexp.Compile(r)
	if err != nil {
		return nil, err
	}

	// Perform an initial discovery of the /dev/nvme*... devices
	devicePath := "/dev"
	rawDeviceRegexp := regexp.MustCompile(`^nvme\d+$`)

	files, err := ioutil.ReadDir(devicePath)
	if err != nil {
		return nil, err
	}

	devices := make([]string, 0)
	for _, file := range files {
		if rawDeviceRegexp.MatchString(file.Name()) {

			// Check Serial Number or SUBNQN for the matching device regexp
			cmd := fmt.Sprintf("nvme id-ctrl %s | grep -e sn -e subnqn -e mn", path.Join(devicePath, file.Name()))
			rsp, err := exec.Command("bash", "-c", cmd).Output()
			if err != nil {
				continue
			}

			if deviceRegexp.Match(rsp) {
				devices = append(devices, path.Join(devicePath, file.Name()))
			}
		}
	}

	return devices, nil
}
