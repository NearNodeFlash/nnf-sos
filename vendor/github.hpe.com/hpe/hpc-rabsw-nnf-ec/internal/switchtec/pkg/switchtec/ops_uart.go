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
	"bytes"
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/pkg/term"
	"github.com/sigurn/crc8"
)

/*
 * Example of uart operations
 *
 * GAS Write:
 *
 * command: gaswr -c -s <offset> 0x<byte str> <crc>
 *
 * case 1: success
 * input:    gaswr -c -s 0x5 0xaabbccddeeff 0x84
 * output:   gas_reg_write() success
 * 	     CRC: [0x84/0x84]
 *	     0x00000000:1212>
 *
 * case 2: success
 * input:    gaswr -c -s 0x135c10 0x00000008 0xbc
 * output:   [PFF] cs addr: 0x0304, not hit
 *           gas_reg_write() success
 * 	     CRC: [0xbc/0xbc]
 *	     0x00000000:2172>
 *

 * case 3: crc error
 * input:    gaswr -c -s 0x5 0xaabbccddeeff 0xb
 * output:   gas_reg_write() CRC Error
 * 	     CRC: [0x84/0x0b]
 *	     0x00000000:0000>
 *
 * case 4: out of range
 * input:    gaswr -c -s 0x5135c00 0x00000000 0xe9
 * output:   Error with gas_reg_write(): 0x63006, Offset:0x5135c00
 *           CRC:[0xe9/0xe9]
 *           0x00000000:084d>
 *
 * GAS Read:
 *
 * command: gasrd -c -s <offset> <byte count>
 *
 * case 1: success
 * input:    gasrd -c -s 0x3 5
 * output:   gas_reg_read <0x3> [5 Byte]
 * 	     00 58 00 00 00
 * 	     CRC: 0x37
 *           0x00000000:1204>
 *
 * case 2: success
 * input:    gasrd -c -s 0x135c00 4
 * output:   gas_reg_read <0x135c00> [4 Byte]
 *           [PFF] cs addr: 0x0300,not hit
 * 	     00 00 00 00
 * 	     CRC: 0xb6
 * 	     0x00000000:0d93>
 *
 * case 3: out of range
 * input:    gasrd -c -s 0x5135c00 4
 * output:   gas_reg_read <0x5135c00> [4 Byte]
 *           No access beyond the Total GAS Section
 * 	     ...
 * 	     ...
 * 	     0x00000000:0d93>
 */
const (
	uartMaxWriteBytes = 100 // this matches switchtec, but it can be up to 1023
	uartMaxReadBytes  = 1024
)

var (
	mrpc mrpcRegs
)

type uartOps struct {
	term *term.Term
}

// OpenUart will open a UART device and configure
// the device for UART operations.
func (dev *Device) OpenUart(name string) error {

	t, err := term.Open(name, term.Speed(230400), term.RawMode)
	if err != nil {
		return err
	}

	/*
		readTimeout, _ := time.ParseDuration("1s")
		t.SetReadTimeout(readTimeout)
	*/

	ops := uartOps{term: t}

	if err := ops.detectPrompt(); err != nil {
		return err
	}

	/*
		if err := ops.cliControl("pscdbg 0 all\r"); err != nil {
			return err
		}
	*/

	if err := ops.cliControl("echo 0\r"); err != nil {
		return err
	}

	dev.ops = &ops

	return nil
}

func (ops *uartOps) getDeviceID(dev *Device) (uint32, error) {
	var regs sysInfoRegs

	offset := gasSysInfoOffset + int(unsafe.Offsetof(regs.deviceID))
	size := int(unsafe.Sizeof(regs.deviceID))
	reg, err := ops.gasRead(dev, offset, size)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(reg), nil
}

func (ops *uartOps) loadPartitionInfo(dev *Device) error {
	var regs topRegs

	offset := gasTopCfgOffset + int(unsafe.Offsetof(regs.partitionId))
	size := int(unsafe.Sizeof(regs.partitionId))
	reg, err := ops.gasRead(dev, offset, size)
	if err != nil {
		return err
	}

	dev.partition = uint8(binary.LittleEndian.Uint32(reg))

	offset = gasTopCfgOffset + int(unsafe.Offsetof(regs.partitionCount))
	size = int(unsafe.Sizeof(regs.partitionCount))
	reg, err = ops.gasRead(dev, offset, size)
	if err != nil {
		return err
	}

	dev.partitionCount = uint8(binary.LittleEndian.Uint32(reg))

	return nil
}

func (ops *uartOps) detectPrompt() error {

	// Permit up to 1 second to read the prompt
	timeout, _ := time.ParseDuration("1s")

	// Temporarily set a read timeout to make sure
	// we drain the entire read buffer

	readTimeout, _ := time.ParseDuration("100ms")
	ops.term.SetReadTimeout(readTimeout)
	defer ops.term.SetReadTimeout(0)

	if err := ops.sendCommand("\r", nil, 0); err != nil {
		return err
	}

	var rsp string = ""
	start := time.Now()
	for true {
		var buf [32]byte
		cnt, err := ops.term.Read(buf[:])

		fmt.Printf("DETECT PROMPT: %d bytes, err %v\n", cnt, err)

		rsp += string(buf[:cnt])

		// Standard Prompt is "0x12345678:ABCD>"
		if cnt == 0 && strings.LastIndex(rsp, ":")+5 == strings.LastIndex(rsp, ">") {
			break
		}

		if time.Now().Sub(start) > timeout {
			return fmt.Errorf("Prompt detect timeout")
		}
	}

	fmt.Println("PROMPT DETECTED!")
	return nil
}

func (ops *uartOps) cliControl(str string) error {
	if err := ops.sendCommand(str, nil, 0); err != nil {
		return err
	}

	if _, err := ops.readResponseLn(); err != nil {
		return err
	}

	return nil
}

func (ops *uartOps) sendCommand(str string, data []byte, crc uint32) error {

	cmd := str

	if len(data) != 0 {
		for i := 0; i < len(data); i++ {
			cmd += fmt.Sprintf("%02x", data[len(data)-1-i])
		}

		cmd += fmt.Sprintf(" %#x\r", crc)
	}

	bytes := []byte(cmd)

	dump := func(data []byte) {
		var ascii [16]byte

		size := len(data)
		for i := 0; i < size; i++ {
			b := data[i]

			if i%16 == 0 {
				fmt.Printf("\n%08x: ", i)
			}

			fmt.Printf("%02x ", b)
			if b >= ' ' && b <= '~' {
				ascii[i%16] = b
			} else {
				ascii[i%16] = uint8('.')
			}

			if ((i+1)%8 == 0) || (i+1 == size) {
				fmt.Printf(" ")
				if (i+1)%16 == 0 {
					fmt.Printf("| %s", string(ascii[:]))
				} else if i+1 == size {
					ascii[(i+1)%16] = 0
					if (i+1)%16 <= 8 {
						fmt.Printf(" ")
					}
					for j := (i + 1) % 16; j < 16; j++ {
						fmt.Printf("   ")
						ascii[j] = ' '
					}
					fmt.Printf("| %s", ascii[:])
				}
			}
		}
		fmt.Printf("\n")
	}

	dump(bytes)

	_, err := ops.term.Write(bytes)
	return err
}

func (ops *uartOps) readResponseLn() (string, error) {
	var rsp strings.Builder
	rsp.Grow(uartMaxReadBytes)

	for true {
		var buf [uartMaxReadBytes]byte
		cnt, err := ops.term.Read(buf[:])
		if err != nil {
			return "", err
		}

		rsp.Write(buf[:cnt])

		indexer := func() func(rune) bool {
			index := 0
			lastColonIndex := -1
			lastGreaterThanIndex := 1

			return func(r rune) bool {

				if r == ':' {
					lastColonIndex = index
				} else if r == '>' {
					lastGreaterThanIndex = index
				}

				if lastColonIndex+5 == lastGreaterThanIndex {
					return true
				}

				index++
				return false
			}
		}

		if strings.IndexFunc(rsp.String(), indexer()) > 0 {
			break
		}
	}

	return rsp.String(), nil
}

func (ops *uartOps) submitCommand(dev *Device, cmd Command, buf []byte) error {
	//fmt.Printf("Submit Command: %02x: %v %d\n", cmd, buf, len(buf))
	return dev.gasCommand(cmd, buf)
}

func (ops *uartOps) readResponse(dev *Device, response []byte) error {
	rsp, err := ops.gasReadMrpcPayload(dev, len(response))
	copy(response, rsp)
	return err
}

func (ops *uartOps) close(dev *Device) error {

	if err := ops.term.Close(); err != nil {
		return err
	}

	return nil
}

func (ops *uartOps) gasMap(dev *Device, readonly bool) error {
	// no-op
	return nil
}

func (ops *uartOps) gasUnmap(dev *Device) error {
	// no-op
	return nil
}

func (ops *uartOps) gasWrite(dev *Device, offset int, data []byte) error {
	addr := make([]byte, 4)

	binary.BigEndian.PutUint32(addr, uint32(offset))

	table := crc8.MakeTable(crc8.CRC8)
	crc := crc8.Checksum(addr, table)
	for i := len(data); i > 0; i-- {
		crc = crc8.Update(crc, data[i-1:i], table)
	}

	// Usage: gaswr [-c] [-s] [-v] <address> <data0> [data1] [data2] [ ... ] [CRC]
	// Option:
	// 		-c: Enable CRC check for address and data
	//		-s: Enable byte mode, data is HEX byte stream, e.g., 0x1234abcde or 1234abcde
	//			Otherwise enable DW mode, data is DW
	//      -v: Print more details
	if err := ops.sendCommand(fmt.Sprintf("gaswr -c -s %#x 0x", offset), data, uint32(crc)); err != nil {
		return err
	}

	_, err := ops.readResponseLn()
	if err != nil {
		return err
	}

	return nil
}

func (ops *uartOps) gasWriteMrpcPayload(dev *Device, data []byte) error {
	offset := gasMrpcOffset + int(unsafe.Offsetof(mrpc.input))
	for n := 0; n != len(data); {
		cnt := len(data) - n
		if cnt > uartMaxWriteBytes {
			cnt = uartMaxWriteBytes
		}

		if err := ops.gasWrite(dev, offset, data[n:n+cnt]); err != nil {
			return err
		}

		offset += cnt
		n += cnt
	}

	return nil
}

func (ops *uartOps) gasWrite32(dev *Device, offset int, data uint32) error {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, data)
	return ops.gasWrite(dev, offset, b)
}

var (
	// Case 1: Success
	// Input:  gasrd -c -s 0x3 5
	// Output: gas_reg_read <0x3> [5 byte]
	//         00 58 00 00 00
	//         CRC: 0x37
	gasReadRegExp1 = regexp.MustCompile(
		`<0x(?P<offset>[[:xdigit:]]*)> \[(?P<length>\d*) Byte\]\s*((?P<data>[[:xdigit:]]{2}\s*)*)[^:]*: 0x(?P<crc>[[:xdigit:]]{2})`)
)

func (ops *uartOps) gasRead(dev *Device, offset int, length int) ([]byte, error) {
	//fmt.Printf("GAS Read: %x %d\n", offset, length)

	// Command: gasrd -c -s <offset> <length>
	err := ops.sendCommand(fmt.Sprintf("gasrd -c -s 0x%x %d\r", offset, length), nil, 0)
	if err != nil {
		return nil, err
	}

	rsp, err := ops.readResponseLn()
	if err != nil {
		return nil, err
	}

	loadResponse := func(tokens []string) (offset int, length int, data []byte, crc uint8) {

		//fmt.Printf("RSP DATA: '%s'", tokens[0])
		o, err := strconv.ParseInt(tokens[1], 16, 32)
		if err != nil {
			panic(err)
		}

		l, err := strconv.ParseInt(tokens[2], 10, 32)
		if err != nil {
			panic(err)
		}

		datasplitter := func(r rune) bool {
			return r == ' ' || r == '\r' || r == '\n'
		}

		data = make([]byte, l)
		for i, t := range strings.FieldsFunc(tokens[3], datasplitter) {
			//fmt.Printf("  %d: %s\n", i, t)
			if len(t) == 0 {
				continue
			}
			d, err := strconv.ParseUint(t[0:2], 16, 8)
			if err != nil {
				panic(err)
			}
			data[i] = byte(d)
		}

		c, err := strconv.ParseUint(tokens[len(tokens)-1], 16, 8)
		if err != nil {
			panic(err)
		}

		return int(o), int(l), data, uint8(c)
	}

	if match := gasReadRegExp1.FindStringSubmatch(rsp); match != nil {
		//fmt.Println("SUBSTRING MATCHED!!!")

		rspOffset, rspLength, rspData, rspCrc := loadResponse(match)

		if rspOffset != offset {
			return nil, fmt.Errorf("Response offset incorrect")
		}

		if rspLength != length {
			return nil, fmt.Errorf("Response length incorrect")
		}

		rspAddr := make([]byte, 4)
		binary.BigEndian.PutUint32(rspAddr, uint32(rspOffset))

		table := crc8.MakeTable(crc8.CRC8)
		crc := crc8.Checksum(rspAddr, table)
		crc = crc8.Update(crc, rspData[:], table)

		if rspCrc != crc {
			return rspData, fmt.Errorf("Response CRC incorrect. Expected: %02x Actual: %02x", crc, rspCrc)
		}

		//fmt.Printf("GAS Read Complete: %v (%d)\n", rspData, rspLength)

		return rspData, nil
	}

	return nil, fmt.Errorf("Not yet implemented")
}

func (ops *uartOps) gasReadMrpcPayload(dev *Device, size int) ([]byte, error) {
	var buf bytes.Buffer

	offset := gasMrpcOffset + int(unsafe.Offsetof(mrpc.output))
	for n := 0; n < size; {
		cnt := size - n
		if cnt > uartMaxReadBytes {
			cnt = uartMaxReadBytes
		}

		b, err := ops.gasRead(dev, offset, cnt)
		if err != nil {
			return buf.Bytes(), err
		}

		buf.Write(b)

		offset += cnt
		n += cnt
	}

	return buf.Bytes(), nil
}

func (ops *uartOps) gasRead32(dev *Device, offset int) (uint32, error) {
	ret, err := ops.gasRead(dev, offset, 4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(ret), err
}
