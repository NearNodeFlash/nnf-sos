package api

// TODO: Obsolete this because the volumes are no longer represented by the namespace manager
//       All code has moved into the nnf-manager.
type NvmeApi interface {
	GetVolumes(controllerId string) ([]string, error)
}

var NvmeInterface NvmeApi

func RegisterNvmeInterface(api NvmeApi) {
	NvmeInterface = api
}
