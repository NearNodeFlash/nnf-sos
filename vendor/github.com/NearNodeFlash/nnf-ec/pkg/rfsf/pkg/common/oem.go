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
func MarshalOem(in interface{}) map[string]interface{} {
	oem := make(map[string]interface{})

	val := reflect.ValueOf(in)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {

		fieldVal := val.Field(i)
		fieldTyp := typ.Field(i)

		fieldName := fieldTyp.Name

		switch fieldTyp.Type.Kind() {
		case reflect.Bool:
			oem[fieldName] = fieldVal.Bool()
		case reflect.Int:
			oem[fieldName] = fieldVal.Int()
		case reflect.String:
			oem[fieldName] = fieldVal.String()
		case reflect.Struct:
			oem[fieldName] = MarshalOem(fieldVal.Interface())
		case reflect.Slice:

			// Note: This encodes an abritrary value; but the decoder assumes these are strings.
			sliceField := reflect.MakeSlice(fieldVal.Type(), fieldVal.Len(), fieldVal.Cap())
			for j := 0; j < sliceField.Len(); j++ {
				sliceField.Index(j).Set(fieldVal.Index(j))
			}

			oem[fieldName] = sliceField.Interface()

		default:
			//return &InvalidMarshalOemError{fieldTyp}
			panic("oem: Marshal(" + fieldTyp.Type.Kind().String() + ")")
		}
	}

	return oem
}

// UnmarshalOem method converts an OEM map[string]interface{} into the provided structure pointer
func UnmarshalOem(oem map[string]interface{}, out interface{}) error {

	val := reflect.ValueOf(out)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return &InvalidUnmarshalOemError{reflect.TypeOf(out)}
	}

	val = val.Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldTyp := typ.Field(i)

		fieldName := fieldTyp.Name

		if _, ok := oem[fieldName]; !ok {
			continue
		}

		switch fieldTyp.Type.Kind() {
		case reflect.Bool:
			fieldVal.SetBool(oem[fieldName].(bool))
		case reflect.Int:
			if _, ok := oem[fieldName].(float64); ok {
				fieldVal.SetInt(int64(oem[fieldName].(float64)))
			} else {
				fieldVal.SetInt(oem[fieldName].(int64))
			}
		case reflect.String:
			fieldVal.SetString(oem[fieldName].(string))
		case reflect.Struct:
			if err := UnmarshalOem(oem[fieldName].(map[string]interface{}), fieldVal.Addr().Interface()); err != nil {
				return err
			}
		case reflect.Slice:

			arr := reflect.ValueOf(oem[fieldName])

			// Resize the fieldVal to take the provided slice/array length
			fieldVal.Set(reflect.MakeSlice(fieldVal.Type(), arr.Len(), arr.Len()))

			// This assumes an array of strings. We could probably nest this same as reflect.Struct
			// to make it an array of arbitrary types if we wanted to.
			for j := 0; j < arr.Len(); j++ {
				fieldVal.Index(j).SetString(arr.Index(j).Interface().(string))
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
