/* -----------------------------------------------------------------
 * model_message_registry.go -
 *
 * DMTF MessageRegistry
 *
 * Author: Caleb Carlson
 *
 * © Copyright 2020 Hewlett Packard Enterprise Development LP
 *
 * ----------------------------------------------------------------- */

package openapi

type MessageRegistry struct {
	OdataType string `json:"@odata.type"`

	//Actions - The available actions for this resource.
	Actions map[string]interface{} `json:"Actions,omitempty"`

	//Description - Description of the Message Registry
	Description string `json:"Description,omitempty"`

	//Id - Id of the Message Registry
	Id string `json:"Id"`

	//Language - The RFC5646-conformant language code for the Message Registry.
	Language string `json:"Language"`

	//Messages - The message keys contained in the Message Registry.
	Messages map[string]MessageProperty `json:"Messages"`

	//Name - The Name of the Message Registry.
	Name string `json:"Name"`

	//Oem - The OEM extension property.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	//OwningEntity - The organization or company that publishes this Message Registry.
	OwningEntity string `json:"OwningEntity"`

	//RegistryPrefix - The single-word prefix that is used in forming and decoding MessageIds.
	RegistryPrefix string `json:"RegistryPrefix"`

	//RegistryVersion - The Message Registry version in the middle portion of a MessageId.
	RegistryVersion string `json:"RegistryVersion"`
}
