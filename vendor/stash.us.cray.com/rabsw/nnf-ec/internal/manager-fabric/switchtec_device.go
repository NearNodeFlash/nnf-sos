package fabric

import (
	"os"
	"strconv"

	"stash.us.cray.com/rabsw/switchtec-fabric/pkg/switchtec"
)

type SwitchtecController struct{}

func NewSwitchtecController() SwitchtecControllerInterface {
	return &SwitchtecController{}
}

func (SwitchtecController) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (SwitchtecController) Open(path string) (SwitchtecDeviceInterface, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	dev, err := switchtec.Open(path)
	return &SwitchtecDevice{dev: dev}, err
}

type SwitchtecDevice struct {
	dev *switchtec.Device
}

func (d *SwitchtecDevice) Device() *switchtec.Device {
	return d.dev
}

func (d *SwitchtecDevice) Path() *string {
	panic("Switchtec Device does not support Path getter")
}
func (d *SwitchtecDevice) Close() {
	d.dev.Close()
}

func (d *SwitchtecDevice) Identify() (int32, error) {
	return d.dev.Identify()
}

func (d *SwitchtecDevice) GetFirmwareVersion() (string, error) {
	return d.dev.GetFirmwareVersion()
}

func (d *SwitchtecDevice) GetModel() (string, error) {
	id, err := d.dev.GetDeviceId()
	return strconv.Itoa(int(id)), err
}

func (d *SwitchtecDevice) GetManufacturer() (string, error) {
	return "Microsemi", nil
}

func (d *SwitchtecDevice) GetSerialNumber() (string, error) {
	sn, err := d.dev.GetSerialNumber()
	return strconv.Itoa(int(sn)), err
}

func (d *SwitchtecDevice) GetPortStatus() ([]switchtec.PortLinkStat, error) {
	return d.dev.LinkStat()
}

func (d *SwitchtecDevice) EnumerateEndpoint(id uint8, f func(epPort *switchtec.DumpEpPortDevice) error) error {
	return d.dev.GfmsEpPortDeviceEnumerate(id, f)
}

func (d *SwitchtecDevice) Bind(hostPhysPortId, hostLogPortId uint8, pdfid uint16) error {
	return d.dev.Bind(uint8(d.dev.ID()), hostPhysPortId, hostLogPortId, pdfid)
}
