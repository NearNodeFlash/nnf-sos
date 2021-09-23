package messageregistry

import (
	"net/http"
)

type Api interface {
	RedfishV1RegistriesGet(w http.ResponseWriter, r *http.Request)
	RedfishV1RegistriesRegistryIdGet(w http.ResponseWriter, r *http.Request)
	RedfishV1RegistriesRegistryIdRegistryGet(w http.ResponseWriter, r *http.Request)
}
