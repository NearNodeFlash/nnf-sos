package nvme

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/HewlettPackard/structex"
	"stash.us.cray.com/rabsw/switchtec-fabric/pkg/nvme"

	"stash.us.cray.com/rabsw/nnf-ec/pkg/logging"
	fabric "stash.us.cray.com/rabsw/nnf-ec/pkg/manager-fabric"
)

func NewCliNvmeController() NvmeController {
	return &cliNvmeController{}
}

type cliNvmeController struct{}

func (cliNvmeController) NewNvmeDeviceController() NvmeDeviceController {
	return &cliNvmeDeviceController{}
}

type cliNvmeDeviceController struct{}

func (cliNvmeDeviceController) NewNvmeDevice(fabricId, switchId, portId string) (NvmeDeviceApi, error) {
	spath := fabric.GetSwitchPath(fabricId, switchId)
	if spath == nil {
		return nil, fmt.Errorf("Switch Path Not Found")
	}

	pdfid, err := fabric.GetPortPDFID(fabricId, switchId, portId, 0)
	if err != nil {
		return nil, err
	}

	return &cliDevice{
		path:  *spath,
		pdfid: pdfid,
	}, nil
}

type cliDevice struct {
	path  string // Path to switchtec device managing this device
	pdfid uint16 // PDFID of the device
}

func (d *cliDevice) dev() string {
	return fmt.Sprintf("%#04x@%s", d.pdfid, d.path)
}

// IdentifyController -
func (d *cliDevice) IdentifyController(controllerId uint16) (*nvme.IdCtrl, error) {
	if controllerId != 0 {
		panic("Identify Controller: non-zero controller ID not yet supported")
	}

	rsp, err := d.command(fmt.Sprintf("id-ctrl %s --output-format=binary", d.dev()))
	if err != nil {
		return nil, err
	}

	ctrl := new(nvme.IdCtrl)

	err = structex.DecodeByteBuffer(bytes.NewBuffer([]byte(rsp)), ctrl)

	return ctrl, err
}

// IdentifyNamespace -
func (d *cliDevice) IdentifyNamespace(namespaceId nvme.NamespaceIdentifier) (*nvme.IdNs, error) {
	opts := ""
	if namespaceId != CommonNamespaceIdentifier {
		opts = "--force"
	}

	rsp, err := d.command(fmt.Sprintf("id-ns %s --namespace-id=%d %s --output-format=binary", d.dev(), namespaceId, opts))
	if err != nil {
		return nil, err
	}

	ns := new(nvme.IdNs)

	err = structex.DecodeByteBuffer(bytes.NewBuffer([]byte(rsp)), ns)

	return ns, err
}

func (d *cliDevice) ListSecondary() (*nvme.SecondaryControllerList, error) {
	rsp, err := d.command(fmt.Sprintf("list-secondary %s --output-format=binary", d.dev()))
	if err != nil {
		return nil, err
	}

	ls := new(nvme.SecondaryControllerList)

	err = structex.DecodeByteBuffer(bytes.NewBuffer([]byte(rsp)), ls)

	return ls, err
}

func (d *cliDevice) AssignControllerResources(controllerId uint16, resourceType SecondaryControllerResourceType, numResources uint32) error {
	rt := map[SecondaryControllerResourceType]int{VQResourceType: 0, VIResourceType: 1}[resourceType]
	rsp, err := d.command(fmt.Sprintf("virt-mgmt %s --cntlid=%d --rt=%d --nr=%d --act=8", d.dev(), controllerId, rt, numResources))
	if err != nil {
		return err
	}

	if !strings.HasPrefix(rsp, "success") {
		return fmt.Errorf(strings.TrimRight(rsp, "\n"))
	}

	return nil
}

func (d *cliDevice) OnlineController(controllerId uint16) error {
	rsp, err := d.command(fmt.Sprintf("virt-mgmt %s --cntlid=%d --act=9", d.dev(), controllerId))
	if err != nil {
		return err
	}

	if !strings.HasPrefix(rsp, "success") {
		return fmt.Errorf(strings.TrimRight(rsp, "\n"))
	}
	return nil
}

func (d *cliDevice) ListNamespaces(controllerId uint16) ([]nvme.NamespaceIdentifier, error) {
	if controllerId != 0 {
		panic("List Namespaces: non-zero controller ID not yet supported")
	}

	rsp, err := d.command(fmt.Sprintf("list-ns %s --all", d.dev()))
	if err != nil {
		return nil, err
	}

	nsids := make([]nvme.NamespaceIdentifier, 0)

	scanner := bufio.NewScanner(strings.NewReader(rsp))
	for scanner.Scan() {
		line := scanner.Text()

		colIdx := strings.Index(line, ":")
		if colIdx != -1 {
			nsid, err := strconv.ParseUint(strings.TrimRight(line[colIdx+1:], "\n"), 0, 32)
			if err != nil {
				return nil, err
			}

			nsids = append(nsids, nvme.NamespaceIdentifier(nsid))
		}
	}

	return nsids, scanner.Err()
}

func (d *cliDevice) CreateNamespace(capacityBytes uint64, metadata []byte) (nvme.NamespaceIdentifier, error) {
	return ^nvme.NamespaceIdentifier(0), nil
}

func (d *cliDevice) DeleteNamespace(namespaceId nvme.NamespaceIdentifier) error {
	return nil
}

func (d *cliDevice) AttachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error {
	return nil
}

func (*cliDevice) DetachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error {
	return nil
}

func (*cliDevice) SetNamespaceFeature(namespaceId nvme.NamespaceIdentifier, data []byte) error {
	return nil
}

func (*cliDevice) GetNamespaceFeature(namespaceId nvme.NamespaceIdentifier) ([]byte, error) {
	return nil, nil
}

func (*cliDevice) command(cmd string) (string, error) {
	cmd = fmt.Sprintf("switchtec-nvme %s", cmd)

	rsp, err := logging.Cli.Trace(cmd, func(cmd string) ([]byte, error) {
		return exec.Command("bash", "-c", fmt.Sprintf("/usr/sbin/%s", cmd)).Output()
	})

	return string(rsp), err
}
