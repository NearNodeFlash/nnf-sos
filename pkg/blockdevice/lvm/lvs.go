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

type lvsOutput struct {
	Report []lvsReport `json:"report"`
}

type lvsReport struct {
	LV []lvsLogicalVolume `json:"lv"`
}

type lvsLogicalVolume struct {
	Name   string `json:"lv_name"`
	VGName string `json:"vg_name"`
	Attrs  string `json:"lv_attr"`
	Size   string `json:"lv_size"`
}

func lvsListVolumes(ctx context.Context) ([]lvsLogicalVolume, error) {
	output, err := command.Run("lvs --reportformat json")
	if err != nil {
		return nil, fmt.Errorf("could not list logical volumes: %w", err)
	}

	lvsOutput := lvsOutput{}

	if err := json.Unmarshal([]byte(output), &lvsOutput); err != nil {
		return nil, err
	}

	// If there are multiple reports, combine all the logical volumes into a single list
	logicalVolumes := []lvsLogicalVolume{}
	for _, report := range lvsOutput.Report {
		logicalVolumes = append(logicalVolumes, report.LV...)
	}

	return logicalVolumes, nil
}
