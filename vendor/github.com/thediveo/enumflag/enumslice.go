package enumflag

import (
	"fmt"
	"reflect"
	"strings"
)

// EnumSliceValue wraps a slice of enum values for a user-defined enum type.
type EnumSliceValue struct {
	*EnumValue
	merge bool // replace complete slice or merge values?
}

// NewSlice warps a given enum slice variable so that it can ve used as a flag
// Value with pflag.Var and pflag.VarP. It takes the same parameters as New,
// with the exception of expecting a slice instead of a single enum var.
func NewSlice(flag interface{}, typename string, mapping interface{}, sensitivity EnumCaseSensitivity) *EnumSliceValue {
	flagtype := reflect.TypeOf(flag)
	if flagtype.Kind() != reflect.Ptr || flagtype.Elem().Kind() != reflect.Slice {
		panic(fmt.Sprintf(
			"NewSlice requires flag to be a pointer to an enum value slice in order to manage it"))
	}
	return &EnumSliceValue{
		EnumValue: newEnumValue(flag, typename, mapping, sensitivity),
	}
}

// Set either sets or merges the enum slice flag: the first call will set the
// flag value to the specified set of enum values. Later calls then merge enum
// values instead of replacing the current set. This mimics the behavior of
// pflag's slice flags.
func (e *EnumSliceValue) Set(val string) error {
	// First parse and convert the textual enum values into their
	// program-internal codes.
	vals := strings.Split(val, ",")
	enums := make([]Flag, len(vals))
	for idx, enumval := range vals {
		enumcode, err := e.code(enumval)
		if err != nil {
			return err
		}
		enums[idx] = enumcode
	}
	enumslice := reflect.ValueOf(e.value).Elem()
	if !e.merge {
		// Replace any existing default enum value set. For this, we need to
		// convert the parsed enum codes into the format used by the
		// user-defined enum slice.
		size := len(enums)
		v := reflect.MakeSlice(reflect.TypeOf(e.value).Elem(), size, size)
		for idx, enumcode := range enums {
			v.Index(idx).Set(reflect.ValueOf(enumcode).Convert(e.flagtype))
		}
		enumslice.Set(v)
		e.merge = true // ...and next time: merge.
	} else {
		// Merge in the existing enum values.
	next:
		for _, enumcode := range enums {
			// Is this enum value (code) already part of the slice? Then skip
			// it, otherwise append it.
			size := enumslice.Len()
			for idx := 0; idx < size; idx++ {
				if enumslice.Index(idx).Convert(enumFlagType).Interface().(Flag) == enumcode {
					continue next
				}
			}
			enumslice.Set(reflect.Append(
				enumslice,
				reflect.ValueOf(enumcode).Convert(e.flagtype)))
		}
	}
	return nil
}

// String returns the textual representation of an enumeration (flag) slice,
// which can contain multiple enumeration values from the same enumeration
// simultaneously. In case multiple textual representations (=identifiers) exist
// for the same enumeration value, then only the first textual representation is
// returned, which is considered to be the canonical one.
func (e *EnumSliceValue) String() string {
	flagvals := reflect.ValueOf(e.value).Elem()
	idsl := []string{}
	size := flagvals.Len()
	for idx := 0; idx < size; idx++ {
		if ids, ok := e.names[flagvals.Index(idx).Convert(enumFlagType).Interface().(Flag)]; ok {
			if len(ids) > 0 {
				idsl = append(idsl, ids[0])
			}
		}
	}
	return "[" + strings.Join(idsl, ",") + "]"
}
