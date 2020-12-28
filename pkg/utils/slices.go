package utils

import "reflect"

// DelEleInSlice delete an element from slice by index
//  - arr: the reference of slice
//  - index: the index of element will be deleted
func DelEleInSlice(arr interface{}, index int) {
	vField := reflect.ValueOf(arr)
	value := vField.Elem()
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		result := reflect.AppendSlice(value.Slice(0, index), value.Slice(index+1, value.Len()))
		value.Set(result)
	}
}

// DelEleInSliceByFunction delete an element from slice by function
//  - arr: the reference of slice
//  - f: delete if it returns true on element of slice
func DelEleInSliceByFunction(arr interface{}, f func(interface{}) bool) {
	vField := reflect.ValueOf(arr)
	value := vField.Elem()
	result := reflect.Zero(value.Type())
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		for i := 0; i < reflect.Indirect(vField).Len(); i++ {
			if !f(reflect.Indirect(vField).Index(i).Interface()) {
				result = reflect.Append(result, reflect.Indirect(vField).Index(i))
			}
		}
		value.Set(result)
	}
}

// FindEleInSlice takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func FindEleInSlice(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}
