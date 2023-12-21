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

import "context"

type BlockDevice interface {
	// Create the block device (e.g., LVM pvcreate, vgcreate, and lvcreate)
	Create(ctx context.Context, complete bool) (bool, error)

	// Destroy the block device (e.g., LVM pvremove, vgremove, and lvremove)
	Destroy(ctx context.Context) (bool, error)

	// Activate the block device (e.g., LVM lockstart and lvchange --activate y)
	Activate(ctx context.Context) (bool, error)

	// Deactivate the block device (e.g., LVM lockstop and lvchange --activate n)
	Deactivate(ctx context.Context) (bool, error)

	// Get device /dev path
	GetDevice() string

	// Check if the block device has already been formatted for a file system
	CheckFormatted() (bool, error)
}
