package templates

import (
	"reflect"
	"testing"

	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/info"
)

// These tests verify the functionality of EDEN utils.LookupWithCallback
// by passing different paths for lookup we expect different objects passed into callback

var (
	devID       = "test"
	macAddress1 = "00:00:00:01"
	macAddress2 = "00:00:00:02"
	ipAddress1  = "192.168.0.1/24"
	ipAddress2  = "192.168.0.2/24"
	ipAddress3  = "192.168.0.3/24"
	ipAddress4  = "192.168.0.4/24"
	nii         = &info.ZInfoMsg_Niinfo{Niinfo: &info.ZInfoNetworkInstance{
		IpAssignments: []*info.ZmetIPAssignmentEntry{{
			MacAddress: macAddress1,
			IpAddress:  []string{ipAddress1, ipAddress2},
		}, {
			MacAddress: macAddress2,
			IpAddress:  []string{ipAddress3, ipAddress4},
		}},
	}}
	infoTest = &info.ZInfoMsg{
		Ztype:       info.ZInfoTypes_ZiNetworkInstance,
		DevId:       devID,
		InfoContent: nii,
		AtTimeStamp: nil,
	}
)

func checkInSlice(el string, sl []string) bool {
	for _, val := range sl {
		if el == val {
			return true
		}
	}
	return false
}

// TestLookupEmpty try to use empty lookup string
func TestLookupEmpty(t *testing.T) {
	q := ""
	var callback = func(inp reflect.Value) {
		t.Error("not expected callback")
	}
	utils.LookupWithCallback(infoTest, q, callback)
}

// TestLookupWrong try to use lookup with wrong string
//
//	expected not to fire callback
func TestLookupWrong(t *testing.T) {
	q := "wrong"
	var callback = func(inp reflect.Value) {
		t.Error("not expected callback")
	}
	utils.LookupWithCallback(infoTest, q, callback)
}

// TestLookupBase try to use lookup with first element
//
//	expected to receive value of DevId
func TestLookupBase(t *testing.T) {
	q := "DevId"
	var callback = func(inp reflect.Value) {
		t.Log(inp)
		if inp.String() != devID {
			t.Errorf("expected: %s, received: %s", devID, inp.String())
		}
	}
	utils.LookupWithCallback(infoTest, q, callback)
}

// TestLookupIndex try to use lookup with one indexed element [0]
//
//	expected to fire callback one time with MacAddress1
func TestLookupIndex(t *testing.T) {
	q := "InfoContent.Niinfo.IpAssignments[0].MacAddress"
	var received []string
	var callback = func(inp reflect.Value) {
		t.Log(inp)
		if inp.String() != macAddress1 {
			t.Errorf("expected: %s, received: %s", macAddress1, inp.String())
		}
		received = append(received, inp.String())
	}
	utils.LookupWithCallback(infoTest, q, callback)
	if len(received) != 1 {
		t.Errorf("expected %d values, received %d", 1, len(received))
	}
	if !checkInSlice(macAddress1, received) {
		t.Errorf("not %s in %s", macAddress1, received)
	}
}

// TestLookupIndexes try to use lookup with two indexed elements
//
//	expected to fire callback one time with IpAddress4
func TestLookupIndexes(t *testing.T) {
	q := "InfoContent.Niinfo.IpAssignments[1].IpAddress[1]"
	var received []string
	var callback = func(inp reflect.Value) {
		t.Log(inp)
		switch inp.String() {
		case ipAddress4:
			received = append(received, inp.String())
		default:
			t.Errorf("not expected value %s", inp.String())
		}
	}
	utils.LookupWithCallback(infoTest, q, callback)
	if len(received) != 1 {
		t.Errorf("expected %d values, received %d", 2, len(received))
	}
	if !checkInSlice(ipAddress4, received) {
		t.Errorf("not %s in %s", ipAddress4, received)
	}
}

// TestLookupBracket try to use lookup with empty brackets to iterate through elements
//
//	expected to fire callback two times with MacAddress1 and MacAddress2
func TestLookupBracket(t *testing.T) {
	q := "InfoContent.Niinfo.IpAssignments[].MacAddress"
	var received []string
	var callback = func(inp reflect.Value) {
		t.Log(inp)
		switch inp.String() {
		case macAddress1:
			received = append(received, inp.String())
		case macAddress2:
			received = append(received, inp.String())
		default:
			t.Errorf("not expected value %s", inp.String())
		}
	}
	utils.LookupWithCallback(infoTest, q, callback)
	if len(received) != 2 {
		t.Errorf("expected %d values, received %d", 2, len(received))
	}
	if !checkInSlice(macAddress1, received) {
		t.Errorf("not %s in %s", macAddress1, received)
	}
	if !checkInSlice(macAddress2, received) {
		t.Errorf("not %s in %s", macAddress2, received)
	}
}

// TestLookupBrackets try to use lookup with empty brackets to iterate through elements and with defined index
//
//	expected to fire callback two times with IpAddress1 and IpAddress3
func TestLookupBrackets(t *testing.T) {
	q := "InfoContent.Niinfo.IpAssignments[].IpAddress[0]"
	var received []string
	var callback = func(inp reflect.Value) {
		t.Log(inp)
		received = append(received, inp.String())
	}
	utils.LookupWithCallback(infoTest, q, callback)
	if len(received) != 2 {
		t.Errorf("expected %d values, received %d", 2, len(received))
	}
	if !checkInSlice(ipAddress1, received) {
		t.Errorf("not %s in %s", ipAddress1, received)
	}
	if !checkInSlice(ipAddress3, received) {
		t.Errorf("not %s in %s", ipAddress3, received)
	}
}

// TestLookupBracketsEnds try to use lookup with multiple empty brackets to iterate through elements
//
//	expected to fire callback four times with all IpAddresses
func TestLookupBracketsEnds(t *testing.T) {
	q := "InfoContent.Niinfo.IpAssignments[].IpAddress[]"
	var received []string
	var callback = func(inp reflect.Value) {
		t.Log(inp)
		received = append(received, inp.String())
	}
	utils.LookupWithCallback(infoTest, q, callback)
	if len(received) != 4 {
		t.Errorf("expected %d values, received %d", 4, len(received))
	}
	if !checkInSlice(ipAddress1, received) {
		t.Errorf("not %s in %s", ipAddress1, received)
	}
	if !checkInSlice(ipAddress2, received) {
		t.Errorf("not %s in %s", ipAddress2, received)
	}
	if !checkInSlice(ipAddress3, received) {
		t.Errorf("not %s in %s", ipAddress3, received)
	}
	if !checkInSlice(ipAddress4, received) {
		t.Errorf("not %s in %s", ipAddress4, received)
	}
}
