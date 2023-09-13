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

package ports

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
)

const maxPort = math.MaxUint16

// Validate will validate the provided list of ports.
func Validate(ports []intstr.IntOrString) error {

	for _, port := range ports {

		switch port.Type {
		case intstr.Int:
			if port.IntVal <= 0 {
				return fmt.Errorf("port value '%d' must be a positive value", port.IntVal)
			}
			if port.IntVal > maxPort {
				return fmt.Errorf("port value '%d' exceeds maximum port size %d", port.IntVal, maxPort)
			}
		case intstr.String:

			// Expect the string value of the form "START-END"
			parsed := strings.SplitN(port.StrVal, "-", 2)
			if len(parsed) != 2 {
				return fmt.Errorf("port range '%s' invalid", port.StrVal)
			}

			start, err := strconv.Atoi(parsed[0])
			if err != nil {
				return fmt.Errorf("port range '%s' starting value '%s' failed to parse", port.StrVal, parsed[0])
			}

			if start <= 0 {
				return fmt.Errorf("port range '%s' starting value '%d' must be a positive value", port.StrVal, start)
			}

			if start > maxPort {
				return fmt.Errorf("port range '%s' starting value '%d' exceeds maximum port size %d", port.StrVal, start, maxPort)
			}

			end, err := strconv.Atoi(parsed[1])
			if err != nil {
				return fmt.Errorf("port range '%s' ending value '%s' failed to parse", port.StrVal, parsed[1])
			}

			if end <= 0 {
				return fmt.Errorf("port range '%s' ending value '%d' must be a positive value", port.StrVal, end)
			}

			if end > maxPort {
				return fmt.Errorf("port range '%s' ending value '%d' exceeds maximum port size %d", port.StrVal, end, maxPort)
			}

			if start >= end {
				return fmt.Errorf("port range '%s' invalid", port.StrVal)
			}
		}
	}

	// TODO: Do we wish to validate the port order? Check for overlaps?

	return nil
}

const (
	InvalidPort = uint16(0)
)

// NewPortIterator will return a new iterator for enumerating ports in the system
// configuration.
func NewPortIterator(ports []intstr.IntOrString) *portIterator {
	return &portIterator{
		ports:    ports,
		index:    0,
		subindex: 0,
	}
}

type portIterator struct {
	ports    []intstr.IntOrString
	index    int // index into ports array
	subindex int // if port is string type, index into ranged port value "FIRST-LAST"
}

// Next returns the next port from the iterator, or InvalidPort when done or an error has occurred.
func (itr *portIterator) Next() uint16 {
	if itr.index >= len(itr.ports) {
		return InvalidPort
	}

	current := itr.ports[itr.index]
	if current.Type == intstr.Int {
		itr.index++
		itr.subindex = 0
		return uint16(current.IntVal)
	}

	parsed := strings.SplitN(current.StrVal, "-", 2)
	if len(parsed) != 2 {
		return InvalidPort
	}

	first, err := strconv.ParseUint(parsed[0], 10, 16)
	if err != nil {
		return InvalidPort
	}

	last, err := strconv.ParseUint(parsed[1], 10, 16)
	if err != nil {
		return InvalidPort
	}

	port := first + uint64(itr.subindex)
	itr.subindex++

	if port == last {
		itr.index++
		itr.subindex = 0
	}

	return uint16(port)
}
