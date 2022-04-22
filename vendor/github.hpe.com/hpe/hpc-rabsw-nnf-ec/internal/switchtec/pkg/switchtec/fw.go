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
	"fmt"
)

type FirmwarePartitionId uint8

const (
	Map0FirmwarePartitionId    FirmwarePartitionId = 0x0
	Map1FirmwarePartitionId    FirmwarePartitionId = 0x1
	Key0FirmwarePartitionId    FirmwarePartitionId = 0x2
	Key1FirmwarePartitionId    FirmwarePartitionId = 0x3
	BL20FirmwarePartitionId    FirmwarePartitionId = 0x4
	BL21FirmwarePartitionId    FirmwarePartitionId = 0x5
	Cfg0FirmwarePartitionId    FirmwarePartitionId = 0x6
	Cfg1FirmwarePartitionId    FirmwarePartitionId = 0x7
	Img0FirmwarePartitionId    FirmwarePartitionId = 0x8
	Img1FirmwarePartitionId    FirmwarePartitionId = 0x9
	NvlogFirmwarePartitionId   FirmwarePartitionId = 0xA
	SEEPROMFirmwarePartitionId FirmwarePartitionId = 0xFE
	InvalidFirmwarePartitionId FirmwarePartitionId = 0xFF
)

type FlashInfo struct {
	FirmwareVersion   uint32
	FlashSize         uint32
	DeviceId          uint16
	EccEnable         uint8
	Reserved0         uint8
	RunningBL2Flag    uint8
	RunningCfgFlag    uint8
	RunningImgFlag    uint8
	RunningKeyFlag    uint8
	RedundancyKeyFlag uint8
	RedundancyBL2Flag uint8
	RedundancyCfgFlag uint8
	RedundancyImgFlag uint8
	Reserved1         [11]uint32
	Map0              FlashPartInfo
	Map1              FlashPartInfo
	KeyMan0           FlashPartInfo
	KeyMan1           FlashPartInfo
	BL20              FlashPartInfo
	BL21              FlashPartInfo
	Cfg0              FlashPartInfo
	Cfg1              FlashPartInfo
	Img0              FlashPartInfo
	Img1              FlashPartInfo
	NVLog             FlashPartInfo
	Vendor            [8]FlashPartInfo
}

type FlashPartInfo struct {
	ImageCrc        uint32
	ImageLen        uint32
	ImageVer        uint16
	Valid           uint8
	Active          uint8
	PartitionStart  uint32
	PartitionEnd    uint32
	PartitionOffset uint32
	PartitionSizeDw uint32
	ReadOnly        uint8
	IsUsing         uint8
	Reserved        [2]uint8
}

type FirmwareMetadata struct {
	Magic              [4]uint8
	SubMagic           [4]uint8
	HeaderVersion      uint32
	SecureVersion      uint32
	HeaderLength       uint32
	MetadataLength     int32
	ImageLength        uint32
	Type               uint32
	Reserved           uint32
	Version            uint32
	Sequence           uint32
	Reserved1          uint32
	DateStr            [8]uint8
	TimeStr            [8]uint8
	ImageStr           [16]uint8
	Reserved2          [4]uint8
	ImageCrc           uint32
	PublicKeyModules   [512]uint8
	PublicKeyExponent  [4]uint8
	UartPort           uint8
	UartRate           uint8
	BistEnable         uint8
	BistGpioPinCfg     uint8
	BistGpioLevelCfg   uint8
	Reserved3          [3]uint8
	XmlVersion         uint32
	RealocatableImgLen uint32
	LinkAddr           uint32
	HeaderCrc          uint32
}

func versionToString(version uint32) string {
	major := version >> 24
	minor := (version >> 16) & 0xFF
	build := version & 0xFFFF

	return fmt.Sprintf("%X.%02X B%03X", major, minor, build)
}

// GetFirmwareVersion -
func (dev *Device) GetFirmwareVersion() (string, error) {
	flashInfo, err := dev.GetFlashInfo()
	if err != nil {
		return "", err
	}

	var partitionId FirmwarePartitionId

	if flashInfo.Img0.IsUsing != 0 {
		partitionId = Img0FirmwarePartitionId
	} else if flashInfo.Img1.IsUsing != 0 {
		partitionId = Img1FirmwarePartitionId
	} else {
		return "", fmt.Errorf("No firmware in use")
	}

	metadata, err := dev.GetFirmwareInfoMetadata(partitionId)

	return versionToString(metadata.Version), err
}

// GetFlashInfo -
func (dev *Device) GetFlashInfo() (*FlashInfo, error) {
	cmd := struct {
		SubCmd uint8
	}{
		SubCmd: uint8(PartitionInfoGetAllInfoSubCommand),
	}

	info := new(FlashInfo)

	err := dev.RunCommand(PartitionInfo, cmd, info)

	return info, err
}

func (dev *Device) GetFirmwareInfoMetadata(partitionId FirmwarePartitionId) (*FirmwareMetadata, error) {
	cmd := struct {
		SubCmd uint8
		PartId uint8
	}{
		SubCmd: uint8(PartitionInfoGetMetadataSubCommand),
		PartId: uint8(partitionId),
	}

	metadata := new(FirmwareMetadata)

	err := dev.RunCommand(PartitionInfo, cmd, metadata)

	return metadata, err
}
