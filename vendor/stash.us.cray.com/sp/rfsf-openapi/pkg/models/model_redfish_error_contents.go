/* -----------------------------------------------------------------
 * model_redfish_error_contents -
 *
 * DMTF RedfishErrorContents model
 *
 * Author: Caleb Carlson
 *
 * © Copyright 2020 Cray Inc.
 *
 * ----------------------------------------------------------------- */

package openapi

//RedfishErrorContents - The properties that describe an error from a Redfish Service.
type RedfishErrorContents struct {

	//Code - A string indicating a specific MessageId from the message registry.
	Code string `json:"code"`

	//Message - A human-readable error message corresponding to the message in the message registry.
	Message string `json:"message"`

	//MessageExtendedInfo - An array of message objects describing one or more error message(s).
	MessageExtendedInfo []map[string]interface{} `json:"@Message.ExtendedInfo,omitempty"`
}
