/*
 * Swordfish API
 *
 * This contains the definition of the Swordfish extensions to a Redfish service.
 *
 * API version: v1.2.c
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"time"
)

// ManagerAccountV162ManagerAccount - The ManagerAccount schema defines the user accounts that are owned by a manager.  Changes to a manager account might affect the current Redfish service connection if this manager is responsible for the Redfish service.
type ManagerAccountV162ManagerAccount struct {

	// The OData description of a payload.
	OdataContext string `json:"@odata.context,omitempty"`

	// The current ETag of the resource.
	OdataEtag string `json:"@odata.etag,omitempty"`

	// The unique identifier for a resource.
	OdataId string `json:"@odata.id"`

	// The type of a resource.
	OdataType string `json:"@odata.type"`

	// The account types.
	AccountTypes []ManagerAccountV162AccountTypes `json:"AccountTypes"`

	Actions ManagerAccountV162Actions `json:"Actions,omitempty"`

	Certificates OdataV4IdRef `json:"Certificates,omitempty"`

	// The description of this resource.  Used for commonality in the schema definitions.
	Description string `json:"Description,omitempty"`

	// An indication of whether an account is enabled.  An administrator can disable it without deleting the user information.  If `true`, the account is enabled and the user can log in.  If `false`, the account is disabled and, in the future, the user cannot log in.
	Enabled bool `json:"Enabled,omitempty"`

	// The identifier that uniquely identifies the resource within the collection of similar resources.
	Id string `json:"Id"`

	Links ManagerAccountV162Links `json:"Links,omitempty"`

	// An indication of whether the account service automatically locked the account because the lockout threshold was exceeded.  To manually unlock the account before the lockout duration period, an administrator can change the property to `false` to clear the lockout condition.
	Locked bool `json:"Locked,omitempty"`

	// The name of the resource or array member.
	Name string `json:"Name"`

	// The OEM account types.
	OEMAccountTypes []string `json:"OEMAccountTypes,omitempty"`

	// The OEM extension.
	Oem map[string]interface{} `json:"Oem,omitempty"`

	// The password.  Use this property with a PATCH or PUT to write the password for the account.  This property is `null` in responses.
	Password string `json:"Password,omitempty"`

	// An indication of whether the service requires that the password for this account be changed before further access to the account is allowed.
	PasswordChangeRequired bool `json:"PasswordChangeRequired,omitempty"`

	// Indicates the date and time when this account password expires.  If `null`, the account password never expires.
	PasswordExpiration *time.Time `json:"PasswordExpiration,omitempty"`

	// The role for this account.
	RoleId string `json:"RoleId,omitempty"`

	SNMP ManagerAccountV162SnmpUserInfo `json:"SNMP,omitempty"`

	// The user name for the account.
	UserName string `json:"UserName,omitempty"`
}