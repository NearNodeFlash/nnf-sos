/*
 * Copyright 2021-2023 Hewlett Packard Enterprise Development LP
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

package controller

import (
	"math"
	"testing"
)

const (
	valid    bool  = true
	invalid  bool  = false
	dontCare int64 = 0

	KB int64 = 1000
	MB int64 = KB * 1000
	GB int64 = MB * 1000
	TB int64 = GB * 1000
	PB int64 = TB * 1000

	KiB int64 = 1024
	MiB int64 = KiB * 1024
	GiB int64 = MiB * 1024
	TiB int64 = GiB * 1024
	PiB int64 = TiB * 1024
)

var capacityTests = []struct {
	capacity      string // #DW directive
	validCommand  bool   // expected parse error result compared with nil
	expectedBytes int64
}{
	{"1000", valid, 1000},   // No units -> bytes
	{"20000", valid, 20000}, // No units -> bytes

	{"0.1MB", valid, int64(0.1 * float64(MB))},
	{"1.1PB", valid, int64(math.Round(1.1 * float64(PB)))},
	{"1.1KB", valid, int64(1.1 * float64(KB))},
	{"1.5MiB", valid, int64(1.5 * float64(MiB))},

	{".1MB", valid, int64(.1 * float64(MB))},
	{".1GiB", valid, int64(math.Round(.1 * float64(GiB)))},

	{"", invalid, dontCare},
	{"MB", invalid, dontCare},
	{"0BM", invalid, dontCare},
	{"0EB", invalid, dontCare},

	{"0MB", valid, 0},
	{"1KB", valid, 1 * KB},
	{"1MB", valid, 1 * MB},
	{"1GB", valid, 1 * GB},
	{"1TB", valid, 1 * TB},
	{"1PB", valid, 1 * PB},

	{"0MiB", valid, 0 * MiB},
	{"1KiB", valid, 1 * KiB},
	{"1MiB", valid, 1 * MiB},
	{"1GiB", valid, 1 * GiB},
	{"1TiB", valid, 1 * TiB},
	{"1PiB", valid, 1 * PiB},

	{"111KB", valid, 111 * KB},
	{"2345MB", valid, 2345 * MB},
	{"987GB", valid, 987 * GB},
	{"04TB", valid, 04 * TB},
	{"1000PB", valid, 1000 * PB},

	{"111KiB", valid, 111 * KiB},
	{"2345MiB", valid, 2345 * MiB},
	{"987GiB", valid, 987 * GiB},
	{"04TiB", valid, 04 * TiB},
	{"1000PiB", valid, 1000 * PiB},

	{"111KiB", valid, 111 * KiB},
	{"2345MiB", valid, 2345 * MiB},
	{"987GiB", valid, 987 * GiB},
	{"04TiB", valid, 04 * TiB},
	{"1000PiB", valid, 1000 * PiB},
}

func TestGetCapacity(t *testing.T) {
	for index, tt := range capacityTests {
		bytes, err := getCapacityInBytes(tt.capacity)

		validCapacity := err == nil
		if validCapacity != tt.validCommand {
			t.Errorf("TestCapacity(%s)(%d): expect_valid(%v) err(%v)", tt.capacity, index, tt.validCommand, err)
			continue
		}

		if bytes != tt.expectedBytes {
			t.Errorf("TestCapacity(%s)(%d): %d reported, expected %d", tt.capacity, index, bytes, tt.expectedBytes)
		}
	}
}
