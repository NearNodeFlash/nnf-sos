/* -----------------------------------------------------------------
 * model_redfish_error.go -
 *
 * DMTF RedfishError model
 *
 * Author: Caleb Carlson
 *
 * © Copyright 2020 Cray Inc.
 *
 * ----------------------------------------------------------------- */

package openapi

//RedfishError - The error payload from a Redfish Service.
type RedfishError struct {

	//Error - The properties that describe an error from a Redfish Service.
	Error RedfishErrorContents `json:"error"`
}
