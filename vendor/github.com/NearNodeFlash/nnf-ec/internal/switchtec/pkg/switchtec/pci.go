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

package switchtec

import "fmt"

const (
	// PCIe Capabilities Registers - all definitions derived from
	// the PCI Express Base Specification, Rev 4.0 Version 1.0

	// 7.5.1.1.11 Capabilities Pointer (Offset 34h) - points to a linked
	// list of capabilities implemented by this function. Bottom two bits
	// are reserved and must be set to 00b.
	pcieCapabilitiesPointer     uint16 = 0x34
	pcieCapabilitiesPointerMask uint8  = 0xFC

	// 7.5.3.1 PCI Express Capability List Register (Offset 00h)
	pcieNextCapabilityPointerOffset uint16 = 0x01

	// 7.5.3.2 PCI Express Capabilities Register (Offset 02h)
	pcieCapabilitiesVersionMask uint16 = 0xFC

	// 7.5.3 PCI Express Capability Structure - this structure is a mechanism
	// for identifying and managaging PCI Express device.
	pcieExpressCapabilitiesId uint8 = 0x10

	// 7.5.3.4 Device Control Register (Offset 08h)
	// PCI Express Base Specification, Rev 4.0 Version 1.0
	pcieDeviceControlRegisterOffset                         uint16 = 0x08
	pcieDeviceControlRegisterInitiateFunctionLevelResetFlag uint16 = 0x8000
)

func (dev *Device) VfReset(pdfid uint16) error {

	// Read the capabilities pointer
	capabilityOffset, err := dev.CsrRead8(pdfid, pcieCapabilitiesPointer)
	if err != nil {
		return fmt.Errorf("vf reset - failed to read register address %#04x", pcieCapabilitiesPointer)
	}

	// Bottom two bits are reserved and must be set to 00b.
	capabilityOffset &= pcieCapabilitiesPointerMask

	// Loop through the PCI capabilities until the PCI Express Capabilities ID is found
	offset := uint16(capabilityOffset)
	for offset != 0 {
		capId, err := dev.CsrRead8(pdfid, offset)
		if err != nil {
			return fmt.Errorf("vf reset - failed to read register address %#04x", offset)
		}

		if capId == pcieExpressCapabilitiesId {
			break
		}

		offset += pcieNextCapabilityPointerOffset
		nextOffset, err := dev.CsrRead8(pdfid, offset)
		if err != nil {
			return fmt.Errorf("vf reset - failed to read register address %#04x", offset)
		}

		offset = uint16(nextOffset)
	}

	if offset == 0 {
		return fmt.Errorf("vf reset - cannot find capability register '%#02x'", pcieExpressCapabilitiesId)
	}

	// We've located the start of the PCIe Express Capability structure. Read in
	// the Device Control register and toggle the reset flag.
	offset += pcieDeviceControlRegisterOffset
	ctrl, err := dev.CsrRead16(pdfid, offset)
	if err != nil {
		return fmt.Errorf("vf reset - failed to read register %#04x", offset)
	}

	ctrl |= pcieDeviceControlRegisterInitiateFunctionLevelResetFlag
	if err := dev.CsrWrite16(pdfid, offset, ctrl); err != nil {
		return fmt.Errorf("vf reset - failed to write register %#04x", offset)
	}

	return nil
}
