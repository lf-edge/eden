package utils

import (
	"regexp"
	"strings"
)

//GetParams parse line with regexp into map
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
