package switchtec

type SerialNumberSecurityVersionInfo struct {
	ChipSerial                uint32
	KeyManifestSecureVersiion uint32
	BL2SecureVersion          uint32
	MainSecureVersion         uint32
	SecureUnlockVersion       uint32
}

// GetDeviceId -
func (dev *Device) GetDeviceId() (uint32, error) {
	return dev.ops.getDeviceID(dev)
}

// GetSerialNumber -
func (dev *Device) GetSerialNumber() (uint32, error) {
	rsp := new(SerialNumberSecurityVersionInfo)

	err := dev.RunCommand(SerialNumberSecVersion, nil, rsp)
	return rsp.ChipSerial, err
}

