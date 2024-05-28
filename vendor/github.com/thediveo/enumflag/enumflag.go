// Copyright 2020 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy
// of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package enumflag

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Flag represents a CLI (enumeration) flag which can take on only a single
// enumeration value out of a fixed set of enumeration values. Applications
// using the enumflag package might want to derive their enumeration flags from
// Flag, such as "type MyFoo enumflag.Flag", but they don't need to. The only
// requirement for user-defined enumeration flags is that they must be
// compatible with the Flag type.
type Flag uint

// EnumCaseSensitivity specifies whether the textual representations of enum
// values are considered to be case sensitive, or not.
type EnumCaseSensitivity bool

// Controls whether the textual representations for enum values are case
// sensitive, or not.
const (
	EnumCaseInsensitive EnumCaseSensitivity = false
	EnumCaseSensitive   EnumCaseSensitivity = true
)

// EnumValue wraps a user-defined enum type value and implements the pflag.Value
// interface, so the user's enum type value can directly be used with the fine
// pflag drop-in package for Golang CLI flags.
type EnumValue struct {
	value       interface{}         // enum value of a user-defined enum type.
	enumtype    string              // name of the user-defined enum type.
	names       enumValueNames      // enum value names.
	sensitivity EnumCaseSensitivity // case sensitive or insensitive?
	flagtype    reflect.Type        // cached enum reflection type.
}

// New wraps a given enum variable so that it can be used as a flag Value with
// pflag.Var and pflag.VarP. The specified flag must be a pointer to a
// user-defined enum value, as otherwise the flag value cannot be managed
// (changed) later on, when a CLI user tries to set it via its corresponding CLI
// flag.
func New(flag interface{}, typename string, mapping interface{}, sensitivity EnumCaseSensitivity) *EnumValue {
	// Ensure that the specified enumeration variable is of a compatible type
	// and that it actually is a pointer to the enum value.
	flagtype := reflect.TypeOf(flag)
	if flagtype.Kind() != reflect.Ptr {
		panic(fmt.Sprintf(
			"New requires flag to be a pointer to an enum value in order to manage it"))
	}
	return newEnumValue(flag, typename, mapping, sensitivity)
}

// newEnumValue returns a correctly set up new EnumValue. It expects the flag
// var to be either a pointer or a pointer to the slice, and the caller to have
// checked this before calling.
func newEnumValue(flag interface{}, typename string, mapping interface{}, sensitivity EnumCaseSensitivity) *EnumValue {
	flagtype := reflect.TypeOf(flag).Elem()
	if flagtype.Kind() == reflect.Slice {
		flagtype = flagtype.Elem()
	}
	flagtypename := flagtype.Name()
	if !flagtype.ConvertibleTo(uintType) {
		panic(fmt.Sprintf("incompatible enum value type %s", flagtypename))
	}
	// Next, ensure that the enumeration values (in form of textual
	// representations) actually are stored in a map.
	mappingrval := reflect.ValueOf(mapping)
	if mappingrval.Kind() != reflect.Map || !mappingrval.Type().Key().ConvertibleTo(uintType) {
		panic(fmt.Sprintf("incompatible enum values map type %s",
			mappingrval.Type().Name()))
	}
	// Oh, the magic of Golang reflexions makes us put on our beer googles, erm,
	// goggles: we now convert the specified enum values into our "canonical"
	// mapping. While we can (mostly) keep the map values (which are string
	// slices), we have to convert the map keys into our canonical EnumFlag key
	// type (=enum values type).
	enummap := enumValueNames{}
	for _, key := range mappingrval.MapKeys() {
		names := mappingrval.MapIndex(key).Interface().([]string)
		if sensitivity == EnumCaseInsensitive {
			names = append(names[:0:0], names...) // https://github.com/golang/go/wiki/SliceTricks
			for idx, name := range names {
				names[idx] = strings.ToLower(name)
			}
		}
		enummap[key.Convert(enumFlagType).Interface().(Flag)] = names
	}
	// Finally return the Value-compatible wrapper, which now has the necessary
	// information about the mapping and case sensitivity, as well as the other
	// things.
	return &EnumValue{
		value:       flag,
		enumtype:    typename,
		names:       enummap,
		sensitivity: sensitivity,
		flagtype:    flagtype,
	}
}

// Get returns the managed enum value as a convenience.
func (e *EnumValue) Get() interface{} {
	return e.value
}

// Set sets the enum flag to the specified enum value. If the specified value
// isn't a valid enum value, then the enum flag will be unchanged and an error
// returned instead.
func (e *EnumValue) Set(val string) error {
	enumcode, err := e.code(val)
	if err == nil {
		// When creating our enum flag wrapper we made sure it has a value
		// reference, so we don't need to double-check here again, but now access it
		// always indirectly.
		reflect.ValueOf(e.value).Elem().Set(
			reflect.ValueOf(enumcode).Convert(e.flagtype))
	}
	return err
}

// code parses the textual representation of an enumeration value, returning the
// corresponding enumeration value, or an error.
func (e *EnumValue) code(val string) (Flag, error) {
	if e.sensitivity == EnumCaseInsensitive {
		val = strings.ToLower(val)
	}
	// Try to find a matching enum value textual representation, and then take
	// its enumation value ("code").
	for enumval, ids := range e.names {
		for _, id := range ids {
			if val == id {
				return enumval, nil
			}
		}
	}
	// Oh no! An invalid textual enum value was specified, so let's generate
	// some useful error explaining which textual representations are valid.
	// We're ordering values by their canonical names in order to achieve a
	// stable error message.
	allids := []string{}
	for _, ids := range e.names {
		s := []string{}
		for _, id := range ids {
			s = append(s, "'"+id+"'")
		}
		allids = append(allids, strings.Join(s, "/"))
	}
	sort.Strings(allids)
	return 0, fmt.Errorf("must be %s", strings.Join(allids, ", "))
}

// String returns the textual representation of an enumeration (flag) value. In
// case multiple textual representations (=identifiers) exist for the same
// enumeration value, then only the first textual representation is returned,
// which is considered to be the canonical one.
func (e *EnumValue) String() string {
	flagval := reflect.ValueOf(e.value).Elem()
	if ids, ok := e.names[flagval.Convert(enumFlagType).Interface().(Flag)]; ok {
		if len(ids) > 0 {
			return ids[0]
		}
	}
	return "<unknown>"
}

// Type returns the name of the flag value type. The type name is used in error
// message.
func (e *EnumValue) Type() string {
	return e.enumtype
}

// enumValueNames maps enumeration values to their corresponding textual
// representations. This mapping is a one-to-many mapping in that the same
// enumeration value may have more than only one associated textual
// representation.
type enumValueNames map[Flag][]string

// Reflection types used in this package.
var enumFlagType = reflect.TypeOf(Flag(0))
var uintType = reflect.TypeOf(uint(0))
