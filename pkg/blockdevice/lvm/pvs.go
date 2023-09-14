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

package lvm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/NearNodeFlash/nnf-sos/pkg/command"
)

type pvsOutput struct {
	Report []pvsReport `json:"report"`
}

type pvsReport struct {
	LV []pvsPhysicalVolume `json:"pv"`
}

type pvsPhysicalVolume struct {
	Name   string `json:"pv_name"`
	VGName string `json:"vg_name"`
	Attrs  string `json:"pv_attr"`
	Size   string `json:"pv_size"`
}

func pvsListVolumes(ctx context.Context) ([]pvsPhysicalVolume, error) {
	output, err := command.Run("pvs --reportformat json")
	if err != nil {
		return nil, fmt.Errorf("could not list physical volumes: %w", err)
	}

	pvsOutput := pvsOutput{}

	if err := json.Unmarshal([]byte(output), &pvsOutput); err != nil {
		return nil, err
	}

	// If there are multiple reports, combine all the physical volumes into a single list
	physicalVolumes := []pvsPhysicalVolume{}
	for _, report := range pvsOutput.Report {
		physicalVolumes = append(physicalVolumes, report.LV...)
	}

	return physicalVolumes, nil
}
