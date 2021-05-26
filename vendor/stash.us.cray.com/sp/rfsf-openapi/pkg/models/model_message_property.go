/* -----------------------------------------------------------------
 * model_message_property.go -
 *
 * DMTF Message
 *
 * Author: Caleb Carlson
 *
 * © Copyright 2020 Hewlett Packard Enterprise Development LP
 *
 * ----------------------------------------------------------------- */

package openapi

type MessageProperty struct {

	//ArgDescriptions - The MessageArg descriptions, in order, used for this message.
	ArgDescriptions []string `json:"ArgDescriptions,omitempty"`

	//ArgLongDescriptions - The MessageArg normative descriptions, in order, used for this message.
	ArgLongDescriptions []string `json:"ArgLongDescriptions,omitempty"`

	//ClearingLogic - The clearing logic associated with this message.
	// The properties within indicate that what messages are cleared by this message as well as under what conditions.
	ClearingLogic ClearingLogic `json:"ClearingLogic,omitempty"`

	//Description - A short description of how and when to use this message.
	Description string `json:"Description"`

	//LongDescription - The normative language that describes this message's usage.
	LongDescription string `json:"LongDescription,omitempty"`

	//Message - This property shall contain the message to display.
	// If a %<integer> is included in part of the string, it shall represent a
	// string substitution for any MessageArgs that accompany the message, in order.
	Message string `json:"Message"`

	//MessageSeverity - The severity of the message.
	MessageSeverity string `json:"MessageSeverity"`

	//NumberOfArgs - This property shall contain the number of arguments that are
	// substituted for the locations marked with %<integer> in the message.
	NumberOfArgs uint32 `json:"NumberOfArgs"`

	//Oem - The OEM extension property.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	//RelatedProperties - A set of properties described by the message.
	RelatedProperties []string `json:"RelatedProperties,omitempty"`

	//Resolution - Used to provide suggestions on how to resolve the situation that caused the error.
	Resolution string `json:"Resolution"`

	// DEPRECATED
	//Severity - This property has been deprecated in favor of MessageSeverity,
	// which ties the values to the enumerations defined for the Health property within Status.
	Severity string `json:"Severity,omitempty"`
}

type ClearingLogic struct {

	//ClearsAll - An indication of whether all prior conditions and messages are cleared,
	// provided the ClearsIf condition is met.
	ClearsAll bool `json:"ClearsAll,omitempty"`

	//ClearsIf - The condition when the event is cleared.
	ClearsIf string `json:"ClearsIf,omitempty"`

	//ClearsMessage - The array of MessageIds that this message clears when the other conditions are met.
	ClearsMessage string `json:"ClearsMessage,omitempty"`
}
