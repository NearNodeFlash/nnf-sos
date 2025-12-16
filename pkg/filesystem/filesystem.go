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

package filesystem

import "context"

type FileSystem interface {
	// Create the file system (e.g., mkfs)
	Create(ctx context.Context, complete bool) (bool, error)

	// Destroy the file system (e.g., wipefs)
	Destroy(ctx context.Context) (bool, error)

	// Activate the file system (e.g., mount Lustre target)
	Activate(ctx context.Context, complete bool) (bool, error)

	// Deactivate the file system (e.g., unmount Lustre target)
	Deactivate(ctx context.Context) (bool, error)

	// Mount the file system
	Mount(ctx context.Context, path string, complete bool) (bool, error)

	// Unmount the file system
	Unmount(ctx context.Context, path string) (bool, error)

	// Run any commands on the OS before the file system is mounted
	PreMount(ctx context.Context, path string, complete bool) (bool, error)

	// Run any commands against the file system after it has been mounted
	PostMount(ctx context.Context, path string, complete bool) (bool, error)

	// Run any commands against the file system before it is unmounted
	PreUnmount(ctx context.Context, path string, complete bool) (bool, error)

	// Run any commands on the OS after the file system is unmounted
	PostUnmount(ctx context.Context, complete bool) (bool, error)

	// Run any commands against the file system after it has been activated
	PostActivate(ctx context.Context, complete bool) (bool, error)

	// Run any commands against the activated file system before it is deactivated
	PreDeactivate(ctx context.Context, complete bool) (bool, error)

	// Run any commands against the mounted file system after it has been fully created. PostSetup
	// is only run on the Rabbit during the Setup phase
	PostSetup(ctx context.Context, complete bool) (bool, error)

	// Run any commands against the mounted file system before it is torn down. PreTeardown
	// is only run on the Rabbit during the Teardown phase
	PreTeardown(ctx context.Context, complete bool) (bool, error)
}
