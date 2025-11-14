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

package blockdevice

import (
	"context"

	"github.com/go-logr/logr"
)

type MockBlockDevice struct {
	Log logr.Logger
}

// Check that Mock implements the BlockDevice interface
var _ BlockDevice = &MockBlockDevice{}

func (m *MockBlockDevice) Create(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Created mock block device")

	return true, nil
}

func (m *MockBlockDevice) Destroy(ctx context.Context) (bool, error) {
	m.Log.Info("Destroyed mock block device")

	return true, nil
}

func (m *MockBlockDevice) Activate(ctx context.Context) (bool, error) {
	m.Log.Info("Activated mock block device")

	return true, nil
}

func (m *MockBlockDevice) Deactivate(ctx context.Context, full bool) (bool, error) {
	m.Log.Info("Deactivated mock block device")

	return true, nil
}

func (m *MockBlockDevice) GetDevice() string {
	return "/dev/mock"
}

func (m *MockBlockDevice) CheckFormatted() (bool, error) {
	return true, nil
}

func (m *MockBlockDevice) CheckExists(ctx context.Context) (bool, error) {
	return true, nil
}

func (m *MockBlockDevice) CheckHealth(ctx context.Context) (bool, error) {
	return true, nil
}

func (m *MockBlockDevice) CheckReady(ctx context.Context) (bool, error) {
	return true, nil
}

func (m *MockBlockDevice) Repair(ctx context.Context) error {
	return nil
}

func (m *MockBlockDevice) PreActivate(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PreActivate")

	return true, nil
}

func (m *MockBlockDevice) PostActivate(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PostActivate")

	return true, nil
}

func (m *MockBlockDevice) PreDeactivate(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PreDeactivate")

	return true, nil
}

func (m *MockBlockDevice) PostDeactivate(ctx context.Context, complete bool) (bool, error) {
	if complete {
		return false, nil
	}

	m.Log.Info("Ran PostDeactivate")

	return true, nil
}
