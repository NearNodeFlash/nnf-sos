package switchtec

import (
	"errors"
	"fmt"
	"unsafe"
)

var (
	MrpcInterruptedError = errors.New("MRPC Status Interrupted")
	MrpcIoError          = errors.New("MRPC I/O Error")
)

func dumpGasRegs() {
	var regs mrpcRegs

	fmt.Printf("Input:       %08x\n", unsafe.Offsetof(regs.input))
	fmt.Printf("Output:      %08x\n", unsafe.Offsetof(regs.output))
	fmt.Printf("Command Reg: %08x\n", unsafe.Offsetof(regs.command))
	fmt.Printf("Status Reg:  %08x\n", unsafe.Offsetof(regs.status))
	fmt.Printf("Return Reg:  %08x\n", unsafe.Offsetof(regs.ret))
}

func (dev *Device) gasCommand(cmd Command, payload []byte) error {

	var regs mrpcRegs

	// Copy the payload to the MRPC input space
	if err := dev.ops.gasWriteMrpcPayload(dev, payload); err != nil {
		return err
	}

	// TODO: Permit retries
	offset := int(unsafe.Offsetof(regs.command))
	if err := dev.ops.gasWrite32(dev, offset, uint32(cmd)); err != nil {
		return err
	}

	// Poll for completion
	offset = int(unsafe.Offsetof(regs.status))

	for done := false; !done; {
		status, err := dev.ops.gasRead32(dev, offset)
		if err != nil {
			return err
		}

		switch status {
		case mrpcInProgressStatus:

			continue
		case mrpcDoneStatus:
			done = true

		case mrpcErrorStatus:
			offset = int(unsafe.Offsetof(regs.ret))
			ret, _ := dev.ops.gasRead32(dev, offset)
			return fmt.Errorf("MRPC Error: %#x", ret)

		case mrpcInterruptedStatus:
			return MrpcInterruptedError

		default:
			return MrpcIoError
		}
	}

	offset = int(unsafe.Offsetof(regs.ret))
	ret, _ := dev.ops.gasRead32(dev, offset)
	if ret != 0 {
		return fmt.Errorf("MRPC Response Error: %#x", ret)
	}

	return nil
}

// GASWrite will write at a given address bytes to the Global Address Space
func (dev *Device) GASWrite(data []byte, addr uint64) error {

	/*
		if dev.IsLocal() {
			gas, err := dev.NewGAS(false)
			if err != nil {
				return err
			}
			defer gas.Unmap()

			return gas.Write(data, addr)
		}

		const maxWriteLength = uint32(maxDataLength - (unsafe.Sizeof(uint32(0)) + unsafe.Sizeof(uint32(0))))

		cmd := struct {
			Offset uint32
			Length uint32 `countOf:"Data"`
			Data   [maxWriteLength]byte
		}{
			Offset: uint32(addr),
			Length: 0,
		}

		remaining := uint32(len(data))
		fmt.Printf("GAS Write: Writing %d bytes\n", remaining)

		for remaining != 0 {

			cmd.Length = remaining
			if cmd.Length > maxWriteLength {
				cmd.Length = maxWriteLength
			}

			copy(cmd.Data[0:cmd.Length], data[cmd.Offset:cmd.Offset+cmd.Length])

			if err := dev.RunCommand(GASWriteCommand, cmd, nil); err != nil {
				return err
			}

			remaining -= cmd.Length
			cmd.Offset += cmd.Length
		}
	*/

	return nil
}

func (dev *Device) GASRead(offset uint64, length uint64) ([]byte, error) {
	/*
		if dev.IsLocal() {
			gas, err := dev.NewGAS(true)
			if err != nil {
				return nil, err
			}

			return gas.Read(offset, length)
		}
	*/

	return nil, fmt.Errorf("Remote GAS Read Not Yet Implemented")
}
