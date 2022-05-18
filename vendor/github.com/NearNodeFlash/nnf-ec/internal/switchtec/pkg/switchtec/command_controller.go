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

package switchtec

import (
	"sync"

	"go.chromium.org/luci/common/runtime/goroutine"
)

// CommandController - The Command Controller is reponsibile for synchronizing access
// to the managed switchtec devices for commands. This is really just an implementation
// of a re-entrant locking mechanism, but the proud Go developers are enthroned on
// their high horses and think use of a re-entrent lock is some indiciation of program
// deficiency - so refuse to implement it in their language. They're also idiots.
//
// The Switchtec controller is a perfect use case for re-entrant locks - all commands
// consist of a request/response pair that needs to be serialized and non-interruptable,
// and there are certain command sets that must also be serialized and
// non-interruptable.
//
// For example, the collection of commands related to NVMe Admin Passthrough -
// which consists of 3 commands: Start, Transfer Data, and Finish. Taken
// individually, each command request/response cannot be interrupted, and the
// entire sequence of three commands cannot be interrupted. Failure to do so
// will cause the Switchtec device to raise an error that the sequence was
// corrupted.
//
// So, the lesson is to never think your programming language is too good for
// something that has been implemented in every programming language before
// yours.

type commandController struct {
	mainLock  sync.Mutex   // Main lock that represents ownership of the device
	innerLock sync.Mutex   // Inner lock, used to manage the workings of the lock
	cond      *sync.Cond   // Conditional used to signal when the lock /might/ be available
	id        goroutine.ID // The is the owning goroutine id
	count     int          // Current number of nested locks owned by the locker

}

func NewCommandController() *commandController {
	ctrl := new(commandController)

	ctrl.id = goroutine.ID(^uint64(0))
	ctrl.count = 0
	ctrl.cond = sync.NewCond(&ctrl.innerLock)

	return ctrl
}

func (c *commandController) Lock() {
	id := goroutine.CurID()

	c.innerLock.Lock()

	for {
		// Check if there is no current owner of the lock and break to
		// acquire it.
		if c.count == 0 {
			c.id = id
			break
		}

		// Check if the current goroutine already has the lock and break to
		// acquire it; this is the case of re-entrent locking.
		if c.id == id {
			break
		}

		// Wait on the inner lock to become available so we can re-evaluate the lock conditions
		// Note the behavior wait - it automatically releases the lock when called, and auto-
		// matically acquires the lock on return.
		c.cond.Wait()
	}

	// At this point we've settled on this goroutine acquring the master lock for the whole
	// controller. Increment the number of lock calls currently held by the goroutine. For
	// very first time it is requested, lock the main mutex.
	c.count++
	if c.count == 1 {
		// Note: Technically this is not needed since the inner-lock controls all the magic;
		// but it also acts as a useful tool to validate the main recursive locking behavior.
		c.mainLock.Lock()

	}

	c.innerLock.Unlock()
}

func (c *commandController) Unlock() {
	c.innerLock.Lock()

	// Decrement the current count of locks held by this controller. When the lock
	// count reaches zero, unlock the main mutext and signal to all waiting routines
	// the availablity of controller.
	c.count--
	if c.count == 0 {
		c.mainLock.Unlock()
		c.cond.Signal()
	}

	c.innerLock.Unlock()
}
