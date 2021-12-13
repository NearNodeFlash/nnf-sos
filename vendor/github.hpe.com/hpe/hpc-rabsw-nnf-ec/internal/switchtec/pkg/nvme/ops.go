package nvme

type ops interface {
	close() error

	submitAdminPassthru(dev *Device, cmd *AdminCmd, data []byte) error
}
