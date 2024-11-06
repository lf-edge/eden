package utils

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
)

// GetControllerMode parse url with controller
func GetControllerMode(controllerMode string) (modeType, modeURL string, err error) {

	params := GetParams(controllerMode, defaults.DefaultControllerModePattern)
	if len(params) == 0 {
		return "", "", fmt.Errorf("cannot parse mode (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	ok := false
	if modeType, ok = params["Type"]; !ok {
		return "", "", fmt.Errorf("cannot parse modeType (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	if modeURL, ok = params["URL"]; !ok {
		return "", "", fmt.Errorf("cannot parse modeURL (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	return
}

// GetParams parse line with regexp into map
func GetParams(line, regEx string) (paramsMap map[string]string) {

	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(strings.TrimSpace(line))

	paramsMap = make(map[string]string)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return
}

// GeneratePassword returns string with defined length and random characters
func GeneratePassword(length int) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rnd.Intn(len(chars))])
	}
	return b.String()
}
