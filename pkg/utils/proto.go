package utils

import (
	"bytes"
	"fmt"

	"google.golang.org/protobuf/proto"
)

//CompareProtoMessages returns true if messages are equal
func CompareProtoMessages(m1, m2 proto.Message) (bool, error) {
	m1Data, err := proto.Marshal(m1)
	if err != nil {
		return false, fmt.Errorf("cannot marshal interface: %v", err)
	}
	m2Data, err := proto.Marshal(m2)
	if err != nil {
		return false, fmt.Errorf("cannot marshal interface: %v", err)
	}
	return bytes.Equal(m1Data, m2Data), nil
}
