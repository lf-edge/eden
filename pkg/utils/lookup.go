package utils

import (
	"errors"
	"fmt"
	"github.com/mcuadros/go-lookup"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"strings"
)

type checkPart = func(reflect.Value)

func parseIndex(s string) (string, int, error) {
	start := strings.Index(s, lookup.IndexOpenChar)
	end := strings.Index(s, lookup.IndexCloseChar)

	if start == -1 && end == -1 {
		return s, -1, nil
	}

	if (start != -1 && end == -1) || (start == -1 && end != -1) {
		return "", -1, lookup.ErrMalformedIndex
	}

	index, err := strconv.Atoi(s[start+1 : end])
	if err != nil {
		return "", -1, lookup.ErrMalformedIndex
	}

	return s[:start], index, nil
}

//LookupWithCallback travels through inpValue by inpPath and apply callback
// you can pass [] without index for iterate over loops
func LookupWithCallback(inpValue interface{}, inpPath string, callback checkPart) {
	defer func() {
		var err error
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New(fmt.Sprint(r))
			}
		}
		if err != nil {
			log.Fatal(err)
		}
	}()
	if inpPath == "" {
		return
	}
	inpPath = strings.TrimLeft(inpPath, ".")
	newStrings := strings.Split(inpPath, "[]")
	if len(newStrings) > 1 {
		firstPart := newStrings[0]
		secondPart := strings.TrimPrefix(inpPath, fmt.Sprint(firstPart, "[]"))
		if secondPart == "" {
			secondPart = "."
		}
		var firstPartValue reflect.Value
		firstPartValue, err := LookUp(inpValue, firstPart)
		if err != nil {
			log.Debug(err)
			return
		}
		switch firstPartValue.Kind() {
		case reflect.Slice:
			for i := 0; i < firstPartValue.Len(); i++ {
				if firstPartValue.Index(i).CanInterface() {
					LookupWithCallback(firstPartValue.Index(i).Interface(), secondPart, callback)
				} else {
					log.Debug("cannot interface ", firstPartValue.Index(i))
				}
			}
		case reflect.Struct:
			for i := 0; i < firstPartValue.NumField(); i++ {
				if firstPartValue.Field(i).CanInterface() {
					LookupWithCallback(firstPartValue.Field(i).Interface(), secondPart, callback)
				} else {
					log.Debug("cannot interface ", firstPartValue.Field(i))
				}
			}
		default:
			log.Debug("default: ", reflect.TypeOf(firstPartValue).Kind())
		}
	} else {
		if inpPath == "" {
			callback(reflect.ValueOf(inpValue))
		} else {
			if strings.HasSuffix(inpPath, lookup.IndexCloseChar) {
				lastOpen := strings.LastIndex(inpPath, lookup.IndexOpenChar)
				firstPart := inpPath[:lastOpen]
				_, ind, err := parseIndex(inpPath[lastOpen-1:])
				if err != nil {
					log.Debug(err)
					return
				}
				value, err := LookUp(inpValue, firstPart)
				if err != nil {
					log.Debug(err)
					return
				}
				callback(value.Index(ind))
			} else {
				value, err := LookUp(inpValue, inpPath)
				if err == nil && value.IsValid() {
					callback(value)
				}
			}
		}
		return
	}
	return
}

//LookUp try to resolve values from interface by path
func LookUp(i interface{}, path string) (value reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New(fmt.Sprint(r))
			}
		}
	}()
	value, err = lookup.LookupString(i, path)
	if err != nil {
		return
	}
	return value, nil
}
