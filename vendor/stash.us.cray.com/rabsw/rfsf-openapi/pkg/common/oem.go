/*
 * Redfish / Swordfish OEM Methods
 *
 * File containing methods supporting OEM fields in the various model_* files
 *
 * Author: Nate Roiger
 *
 * Copyright 2020 Hewlett Packard Enterprise Development LP
 */

package openapi

import (
	"reflect"
)

// MarshalOem method converts the provided structure into the OEM map[string]interface{} format for use in the various model_* files
func MarshalOem(st interface{}) map[string]interface{} {
	ret := make(map[string]interface{})

	val := reflect.ValueOf(st)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {

		fieldVal := val.Field(i)
		fieldTyp := typ.Field(i)

		fieldName := fieldTyp.Name

		switch fieldTyp.Type.Kind() {
		case reflect.Bool:
			ret[fieldName] = fieldVal.Bool()
		case reflect.Int:
			ret[fieldName] = fieldVal.Int()
		case reflect.String:
			ret[fieldName] = fieldVal.String()
		case reflect.Struct:
			ret[fieldName] = MarshalOem(fieldVal.Interface())
		default:
			//return &InvalidMarshalOemError{fieldTyp}
			panic("oem: Marshal(" + fieldTyp.Type.Kind().String() + ")")
		}
	}

	return ret
}

// UnmarshalOem method converts an OEM map[string]interface{} into the provided structure pointer
func UnmarshalOem(mp map[string]interface{}, st interface{}) error {

	val := reflect.ValueOf(st)

	if val.Kind() != reflect.Ptr || val.IsNil() {
		return &InvalidUnmarshalOemError{reflect.TypeOf(st)}
	}

	val = val.Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldTyp := typ.Field(i)

		fieldName := fieldTyp.Name

		if _, ok := mp[fieldName]; !ok {
			continue
		}

		switch fieldTyp.Type.Kind() {
		case reflect.Bool:
			fieldVal.SetBool(mp[fieldName].(bool))
		case reflect.Int:
			fieldVal.SetInt(int64(mp[fieldName].(float64)))
		case reflect.String:
			fieldVal.SetString(mp[fieldName].(string))
		case reflect.Struct:
			if err := UnmarshalOem(mp[fieldName].(map[string]interface{}), fieldVal.Addr().Interface()); err != nil {
				return err
			}
		default:
			return &InvalidUnmarshalOemError{fieldTyp.Type}
		}
	}

	return nil
}

// An InvalidMarshalOemError describes a OEM value that was not
// appropriate value or Go type.
type InvalidMarshalOemError struct {
	Type reflect.StructField
}

func (e *InvalidMarshalOemError) Error() string {
	if e.Type.Type.Kind() != reflect.Ptr {
		return "oem: Marshal(non-pointer " + e.Type.Type.String() + ")"
	}

	return "oem: Marshal(nil " + e.Type.Type.String() + ")"
}

// InvalidUnmarshalOemError describes an invalid argument passed to UnmarshalOem
type InvalidUnmarshalOemError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalOemError) Error() string {
	if e.Type == nil {
		return "oem: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "oem: Unmarshal(non-pointer " + e.Type.String() + ")"
	}

	return "oem: Unmarshal(nil " + e.Type.String() + ")"
}
