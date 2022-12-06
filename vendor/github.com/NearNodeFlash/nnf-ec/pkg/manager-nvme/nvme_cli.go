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

package nvme

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/HewlettPackard/structex"
	"github.com/NearNodeFlash/nnf-ec/internal/switchtec/pkg/nvme"

	"github.com/NearNodeFlash/nnf-ec/pkg/logging"
	fabric "github.com/NearNodeFlash/nnf-ec/pkg/manager-fabric"
)

func NewCliNvmeController() NvmeController {
	return &cliNvmeController{}
}

type cliNvmeController struct{}

func (cliNvmeController) NewNvmeDeviceController() NvmeDeviceController {
	return &cliNvmeDeviceController{}
}

type cliNvmeDeviceController struct{}

func (ctrl cliNvmeDeviceController) Initialize() error { return nil }
func (ctrl cliNvmeDeviceController) Close() error      { return nil }

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
		path:    fmt.Sprintf("%#04x@%s", pdfid, *spath),
		command: "switchtec-nvme",
		pdfid:   pdfid,
	}, nil
}

type cliDevice struct {
	path    string // Path to device
	command string // Command to manage the device at path
	pdfid   uint16 // PDFID of the device, or zero if none
}

func (d *cliDevice) dev() string {
	return d.path
}

// IsDirectDevice -
func (d *cliDevice) IsDirectDevice() bool { return false }

// IdentifyController -
func (d *cliDevice) IdentifyController(controllerId uint16) (*nvme.IdCtrl, error) {
	if controllerId != 0 {
		panic("Identify Controller: non-zero controller ID not yet supported")
	}

	rsp, err := d.run(fmt.Sprintf("id-ctrl %s --output-format=binary", d.dev()))
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

	rsp, err := d.run(fmt.Sprintf("id-ns %s --namespace-id=%d %s --output-format=binary", d.dev(), namespaceId, opts))
	if err != nil {
		return nil, err
	}

	ns := new(nvme.IdNs)

	err = structex.DecodeByteBuffer(bytes.NewBuffer([]byte(rsp)), ns)

	return ns, err
}

func (d *cliDevice) ListSecondary() (*nvme.SecondaryControllerList, error) {
	rsp, err := d.run(fmt.Sprintf("list-secondary %s --output-format=binary", d.dev()))
	if err != nil {
		return nil, err
	}

	ls := new(nvme.SecondaryControllerList)

	err = structex.DecodeByteBuffer(bytes.NewBuffer([]byte(rsp)), ls)

	return ls, err
}

func (d *cliDevice) AssignControllerResources(controllerId uint16, resourceType SecondaryControllerResourceType, numResources uint32) error {
	rt := map[SecondaryControllerResourceType]int{VQResourceType: 0, VIResourceType: 1}[resourceType]
	rsp, err := d.run(fmt.Sprintf("virt-mgmt %s --cntlid=%d --rt=%d --nr=%d --act=8", d.dev(), controllerId, rt, numResources))
	if err != nil {
		return err
	}

	if !strings.HasPrefix(rsp, "success") {
		return fmt.Errorf(strings.TrimRight(rsp, "\n"))
	}

	return nil
}

func (d *cliDevice) OnlineController(controllerId uint16) error {
	rsp, err := d.run(fmt.Sprintf("virt-mgmt %s --cntlid=%d --act=9", d.dev(), controllerId))
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

	rsp, err := d.run(fmt.Sprintf("list-ns %s --all", d.dev()))
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

func (d *cliDevice) ListAttachedControllers(namespaceId nvme.NamespaceIdentifier) ([]uint16, error) {
	// Example Output:
	// 	  # nvme list-ctrl --namespace-id=1 /dev/nvme1
	//    num of ctrls present: 1
	//    [   0]:0x1

	// NOTE: Binary format would be create here as it returns an nvme.CtrlList; but this format is missing
	//       from the latest nvme-cli

	rsp, err := d.run(fmt.Sprintf("list-ctrl %s --namespace-id=%d", d.dev(), namespaceId))
	if err != nil {
		return nil, err
	}

	controllerIds := make([]uint16, 0)
	scanner := bufio.NewScanner(strings.NewReader(rsp))
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "num of ctrls prsent: ") {

			controllerId, err := strconv.ParseUint(line[strings.Index(line, ":0x")+len(":0x"):len(line)-1], 16, 16)
			if err != nil {
				return nil, err
			}

			controllerIds = append(controllerIds, uint16(controllerId))
		}
	}

	return controllerIds, nil
}

func (d *cliDevice) CreateNamespace(sizeInSectors uint64, sectorSizeIndex uint8) (nvme.NamespaceIdentifier, nvme.NamespaceGloballyUniqueIdentifier, error) {
	// Example Command
	//    # nvme create-ns /dev/nvme2 --nsze=468843606 --ncap=468843606 --flbas=3 --nmic=1 
	//    create-ns: Success, created nsid:1

	rsp, err := d.run(fmt.Sprintf("create-ns %s --nsze=%d --ncap=%d --flbas=%d --nmic=1", d.dev(), sizeInSectors, sizeInSectors, sectorSizeIndex))
	if err != nil {
		return 0, nvme.NamespaceGloballyUniqueIdentifier{}, err
	}

	regex := regexp.MustCompile(`^create-ns: Success, created nsid:(?P<NSID>\d+)\n`)
	substrings := regex.FindStringSubmatch(rsp)
	if len(substrings) != 2 {
		return 0, nvme.NamespaceGloballyUniqueIdentifier{}, fmt.Errorf("NSID not found in response output '%s'", rsp)
	}

	nsid, err := strconv.Atoi(substrings[1])
	if err != nil {
		return 0, nvme.NamespaceGloballyUniqueIdentifier{}, err
	}

	ns, err := d.IdentifyNamespace(nvme.NamespaceIdentifier(nsid))
	if err != nil {
		return 0, nvme.NamespaceGloballyUniqueIdentifier{}, err
	}

	return nvme.NamespaceIdentifier(nsid), ns.GloballyUniqueIdentifier, nil
}

func (d *cliDevice) DeleteNamespace(namespaceId nvme.NamespaceIdentifier) error {
	// Example Command
	//    # nvme delete-ns --namespace-id=1 /dev/nvme2
	//    delete-ns: Success, deleted nsid:1

	rsp, err := d.run(fmt.Sprintf("delete-ns %s --namespace-id=%d", d.dev(), namespaceId))
	if err != nil {
		return err
	}

	regex := regexp.MustCompile(`^delete-ns: Success, deleted nsid:(?P<NSID>\d+)\n`)
	substrings := regex.FindStringSubmatch(rsp)
	if len(substrings) != 2 {
		return fmt.Errorf("NSID not found in response output '%s'", rsp)
	}

	nsid, err := strconv.Atoi(substrings[1])
	if err != nil {
		return err
	}

	if namespaceId != nvme.NamespaceIdentifier(nsid) {
		return fmt.Errorf("Deleted NSID (%d) does not match expected NSID (%d)", nsid, namespaceId)
	}

	return nil
}

func (d *cliDevice) FormatNamespace(namespaceID nvme.NamespaceIdentifier) error {
	// Example Command
	//    # nvme format --force --namespace-id=1 /dev/nvme2
	//    Success formatting namespace:1

	rsp, err := d.run(fmt.Sprintf("format %s --force --namespace-id=%d", d.dev(), namespaceID))
	if err != nil {
		return err
	}

	regex := regexp.MustCompile(`Success formatting namespace:(\d+)\n`)
	substrings := regex.FindStringSubmatch(rsp)
	if len(substrings) != 2 {
		return fmt.Errorf("Namespace not found in response output '%s'", rsp)
	}

	nsid, err := strconv.Atoi(substrings[1])
	if err != nil {
		return err
	}

	if namespaceID != nvme.NamespaceIdentifier(nsid) {
		return fmt.Errorf("Formatted NSID (%d) does not match expected NSID (%d)", nsid, namespaceID)
	}

	return nil
}

func (d *cliDevice) WaitFormatComplete(namespaceID nvme.NamespaceIdentifier) error {
	idns := &nvme.IdNs{}
	idns.Utilization = 1 // something other than 0 to get the loop going below

	for idns.Utilization > 0 {
		idns, err := d.IdentifyNamespace(namespaceID)
		if err != nil {
			return err
		}

		fmt.Printf("Formatting, Utilization(bytes): %d              \r", idns.Utilization) // wipe out straggling digits
	}

	fmt.Printf("Formatting, Utilization(bytes): %d              \r", idns.Utilization)
	return nil
}

func (d *cliDevice) AttachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error {
	// Example Command
	//    # nvme attach-ns /dev/nvme2 --namespace-id=1 --controllers=0x41
	//    attach-ns: Success, nsid:1
	if len(controllers) != 1 {
		panic("Only a single controller can be attached at a time by the CLI Device Driver")
	}

	rsp, err := d.run(fmt.Sprintf("attach-ns %s --namespace-id=%d --controllers=%d", d.dev(), namespaceId, controllers[0]))
	if err != nil {
		return err
	}

	regex := regexp.MustCompile(`^attach-ns: Success, nsid:(?P<NSID>\d+)\n`)
	substrings := regex.FindStringSubmatch(rsp)
	if len(substrings) != 2 {
		return fmt.Errorf("NSID not found in response output '%s'", rsp)
	}

	nsid, err := strconv.Atoi(substrings[1])
	if err != nil {
		return err
	}

	if namespaceId != nvme.NamespaceIdentifier(nsid) {
		return fmt.Errorf("Attached NSID (%d) does not match expected NSID (%d)", nsid, namespaceId)
	}

	return nil
}

func (d *cliDevice) DetachNamespace(namespaceId nvme.NamespaceIdentifier, controllers []uint16) error {
	// Example Command
	//    # nvme attach-ns /dev/nvme2 --namespace-id=1 --controllers=0x41
	//    attach-ns: Success, nsid:1
	if len(controllers) != 1 {
		panic("Only a single controller can be detached at a time by the CLI Device Driver")
	}

	rsp, err := d.run(fmt.Sprintf("detach-ns %s --namespace-id=%d --controllers=%d", d.dev(), namespaceId, controllers[0]))
	if err != nil {
		return err
	}

	regex := regexp.MustCompile(`^detach-ns: Success, nsid:(?P<NSID>\d+)\n`)
	substrings := regex.FindStringSubmatch(rsp)
	if len(substrings) != 2 {
		return fmt.Errorf("NSID not found in response output '%s'", rsp)
	}

	nsid, err := strconv.Atoi(substrings[1])
	if err != nil {
		return err
	}

	if namespaceId != nvme.NamespaceIdentifier(nsid) {
		return fmt.Errorf("Detached NSID (%d) does not match expected NSID (%d)", nsid, namespaceId)
	}

	return nil
}

func (*cliDevice) SetNamespaceFeature(namespaceId nvme.NamespaceIdentifier, data []byte) error {
	return nil
}

func (*cliDevice) GetNamespaceFeature(namespaceId nvme.NamespaceIdentifier) ([]byte, error) {
	return nil, nil
}

func (d *cliDevice) GetWearLevelAsPercentageUsed() (uint8, error) {
	rsp, err := d.run(fmt.Sprintf("smart-log %s --output-format=binary", d.dev()))
	if err != nil {
		return 0, err
	}

	log := new(nvme.SmartLog)

	err = structex.DecodeByteBuffer(bytes.NewBuffer([]byte(rsp)), log)

	return log.PercentageUsed, err
}

func (d *cliDevice) run(cmd string) (string, error) {

	rsp, err := logging.Cli.Trace(cmd, func(cmd string) ([]byte, error) {
		return exec.Command("bash", "-c", fmt.Sprintf("/usr/sbin/%s %s", d.command, cmd)).Output()
	})

	return string(rsp), err
}
