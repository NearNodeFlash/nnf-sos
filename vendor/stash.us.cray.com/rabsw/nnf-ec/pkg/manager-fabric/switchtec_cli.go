package fabric

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"stash.us.cray.com/rabsw/switchtec-fabric/pkg/switchtec"

	"stash.us.cray.com/rabsw/nnf-ec/pkg/logging"
)

type SwitchtecCliController struct{}

func NewSwitchtecCliController() SwitchtecControllerInterface {
	return &SwitchtecCliController{}
}

func (c SwitchtecCliController) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (c SwitchtecCliController) Open(path string) (SwitchtecDeviceInterface, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	if _, err := os.Stat("/usr/local/bin/switchtec"); os.IsNotExist(err) {
		return nil, err
	}

	return &SwitchtecCliDevice{path: path}, nil
}

type SwitchtecCliDevice struct {
	path string
	id   int
}

func (d *SwitchtecCliDevice) Device() *switchtec.Device {
	panic("Switchtec CLI Device doens't support Device getter")
}

func (d *SwitchtecCliDevice) Path() *string { return &d.path }

func (d *SwitchtecCliDevice) Close() {}

func (d *SwitchtecCliDevice) Identify() (int32, error) {
	rsp, err := d.run(fmt.Sprintf("fabric gfms-dump %s --type=PAX | awk '/PAX ID: [0-9]+/{printf $3}'", d.path))
	if err != nil {
		return -1, err
	}

	d.id, err = strconv.Atoi(rsp)
	return int32(d.id), err
}

func (d *SwitchtecCliDevice) GetFirmwareVersion() (string, error) {
	return d.run(fmt.Sprintf("info %s | awk '/FW Version: /{printf $3,$4}' OFS=' '", d.path))
}

func (d *SwitchtecCliDevice) GetModel() (string, error) {
	return d.run(fmt.Sprintf("info %s | awk '/Device ID: /{printf $3}'", d.path))
}

func (d *SwitchtecCliDevice) GetManufacturer() (string, error) {
	return "Microchip", nil
}

func (d *SwitchtecCliDevice) GetSerialNumber() (string, error) {
	return d.run(fmt.Sprintf("mfg info %s | awk '/Chip Serial:/{printf $3}'", d.path))
}

func (d *SwitchtecCliDevice) GetPortStatus() ([]switchtec.PortLinkStat, error) {
	rsp, err := d.run(fmt.Sprintf("status %s --pax=%d", d.path, d.id))
	if err != nil {
		return nil, err
	}

	stats := make([]switchtec.PortLinkStat, 0)
	var stat *switchtec.PortLinkStat = nil

	scanner := bufio.NewScanner(strings.NewReader(rsp))
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " \n\r")
		colIdx := strings.Index(line, ":")
		if colIdx == -1 || colIdx+1 >= len(line) {
			continue
		}

		key := strings.TrimSpace(line[:colIdx])
		values := strings.Split(strings.TrimSpace(line[colIdx+1:]), " ")

		switch key {

		case "Phys Port ID":

			physPortId, _ := strconv.Atoi(values[0])

			stats = append(stats, switchtec.PortLinkStat{
				PhysPortId:      uint8(physPortId),
				CfgLinkWidth:    0,
				NegLinkWidth:    0,
				LinkUp:          false,
				LinkGen:         4,
				LinkState:       switchtec.PortLinkState_Unknown,
				CurLinkRateGBps: 0,
			})

			stat = &stats[len(stats)-1]

		case "Status":
			stat.LinkUp = values[0] == "UP"
		case "LTSSM":
			stat.LinkState = switchtec.PortLinkState_L0
		case "Max-Width":
			maxLinkWidth, _ := strconv.Atoi(values[0][1:])
			stat.CfgLinkWidth = uint8(maxLinkWidth)
		case "Neg Width":
			negLinkWidth, _ := strconv.Atoi(values[0][1:])
			stat.NegLinkWidth = uint8(negLinkWidth)
		}
	}

	return stats, scanner.Err()
}

func (d *SwitchtecCliDevice) EnumerateEndpoint(physPortId uint8, handlerFunc func(epPort *switchtec.DumpEpPortDevice) error) error {
	rsp, err := d.run(fmt.Sprintf("fabric gfms-dump %s --type=EP_PORT --ep_pid=%d", d.path, physPortId))
	if err != nil {
		return err
	}

	functions := make([]switchtec.DumpEpPortAttachedDeviceFunction, 0)
	var function *switchtec.DumpEpPortAttachedDeviceFunction = nil

	scanner := bufio.NewScanner(strings.NewReader(rsp))
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " \n\r")

		if strings.HasSuffix(line, "(Not attached)") {

			return handlerFunc(
				&switchtec.DumpEpPortDevice{
					Hdr: switchtec.DumpEpPortHeader{
						PhysicalPort: physPortId,
						Typ:          uint8(switchtec.NoneEpPortType),
					},
				})

		}

		colIdx := strings.Index(line, ":")
		if colIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colIdx])

		// Process a new entry that is 'Function \d+ (SRIOV-[PV]F)'
		if strings.HasPrefix(key, "Function") && (strings.HasSuffix(key, "(SRIOV-PF)") || strings.HasSuffix(key, "(SRIOV-VF)")) {
			functions = append(functions, switchtec.DumpEpPortAttachedDeviceFunction{})
			function = &functions[len(functions)-1]

			id, _ := strconv.Atoi(strings.Split(key, " ")[1])
			function.FunctionID = uint16(id)
			function.VFNum = uint8(id)
			function.SRIOVCapPF = 0

			if strings.HasSuffix(key, "(SRIOV-PF)") {
				function.SRIOVCapPF = 1
			}

			continue
		}

		values := strings.Split(strings.TrimSpace(line[colIdx+1:]), " ")

		switch key {
		case "Function 0 (SRIOV-PF)":
			function.FunctionID = 0
			function.SRIOVCapPF = 1
		case "PDFID":
			pdfid, _ := strconv.ParseUint(values[0], 0, 64)
			function.PDFID = uint16(pdfid)
		case "Binding":
			if values[0] == "Bound" {
				function.Bound = 1
			}
		}
	}

	return handlerFunc(&switchtec.DumpEpPortDevice{
		Hdr: switchtec.DumpEpPortHeader{
			PhysicalPort: physPortId,
			Typ:          uint8(switchtec.DeviceEpPortType),
		},
		Ep: switchtec.DumpEpPortEp{
			Hdr: switchtec.DumpEpPortAttachmentHeader{
				FunctionCount: uint16(len(functions)),
			},
			Functions: functions,
		},
	})
}

func (d *SwitchtecCliDevice) Bind(hostPhysPortId, hostLogPortId uint8, pdfid uint16) error {
	// Usage: switchtec fabric gfms-bind <device> --host_sw_idx=<NUM> --phys_port_id=<NUM> --log_port_id=<NUM> --pdfid=<STR> [OPTIONS]
	rsp, err := d.run(fmt.Sprintf("fabric gfms-bind %s --pax=%d --host_sw_idx=%d --phys_port_id=%d --log_port_id=%d --pdfid=%#04x", d.path, d.id, d.id, hostPhysPortId, hostLogPortId, pdfid))
	if err != nil {
		return err
	}

	if strings.Contains(rsp, "Failed") {
		return fmt.Errorf("Bind operation failed '%s'", rsp)
	}

	return nil
}

func (d *SwitchtecCliDevice) run(cmd string) (string, error) {
	cmd = fmt.Sprintf("switchtec %s", cmd)

	rsp, err := logging.Cli.Trace(cmd, func(cmd string) ([]byte, error) {
		return exec.Command("bash", "-c", fmt.Sprintf("/usr/local/bin/%s", cmd)).Output()
	})

	return string(rsp), err
}
