/* -----------------------------------------------------------------
 * model_message.go -
 *
 * DMTF Message
 *
 * Author: Caleb Carlson
 *
 * © Copyright 2020 Hewlett Packard Enterprise Development LP
 *
 * ----------------------------------------------------------------- */

package openapi

//Message - Should be used by services to send in an Event.
// The MessageId should correspond to one of the predefined
// messages in their respective Message Registries. MessageArgs should
// be provided to splice into the predefined corresponding message string.
type Message struct {

	//Message - The human-readable message, if provided.
	Message string `json:"Message,omitempty"`

	//MessageArgs - This property shall contain the message substitution arguments
	// for the specific message to which this MessageId refers and shall be included
	// only if the MessageId is present. Any number and integer type arguments shall be converted to strings.
	MessageArgs []string `json:"MessageArgs,omitempty"`

	//MessageId - The key for this message used to find the message in a Message Registry.
	MessageId string `json:"MessageId"`

	//MessageSeverity - The severity of the message.
	MessageSeverity string `json:"MessageSeverity,omitempty"`

	//Oem - The OEM extension property.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	//RelatedProperties - This property shall contain an array of JSON Pointers
	//indicating the properties described by the message, if appropriate for the message.
	RelatedProperties []string `json:"RelatedProperties,omitempty"`

	//Resolution - Used to provide suggestions on how to resolve the situation that caused the error.
	Resolution string `json:"Resolution,omitempty"`

	// DEPRECATED
	// This property has been deprecated in favor of MessageSeverity,
	// which ties the values to the enumerations defined for the Health property within Status.
	//Severity - The severity of the errors.
	Severity string `json:"Severity,omitempty"`
}
