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
	"github.com/go-logr/logr"
)

type vgsOutput struct {
	Report []vgsReport `json:"report"`
}

type vgsReport struct {
	LV []vgsVolumeGroup `json:"vg"`
}

type vgsVolumeGroup struct {
	Name    string `json:"vg_name"`
	PVCount string `json:"pv_count"`
	LVCount string `json:"lv_count"`
	Attrs   string `json:"vg_attr"`
	Size    string `json:"vg_size"`
}

func vgsListVolumes(ctx context.Context, log logr.Logger) ([]vgsVolumeGroup, error) {
	output, err := command.Run("vgs --reportformat json", log)
	if err != nil {
		return nil, fmt.Errorf("could not list volume groups: %w", err)
	}

	vgsOutput := vgsOutput{}

	if err := json.Unmarshal([]byte(output), &vgsOutput); err != nil {
		return nil, err
	}

	// If there are multiple reports, combine all the volume groups into a single list
	volumeGroups := []vgsVolumeGroup{}
	for _, report := range vgsOutput.Report {
		volumeGroups = append(volumeGroups, report.LV...)
	}

	return volumeGroups, nil
}
