package openevec

import (
	"flag"
	"fmt"
	"reflect"
	"strings"
)

type PodConfig struct {
	PodName           string   `cobrafield:"name"`
	NoHyper           bool     `cobrafield:"no-hyper"`
	AppLink           string   `cobrafield:""`
	PodMetadata       string   `cobrafield:"metadata"`
	VncDisplay        uint32   `cobrafield:"vnc-display"`
	VncPassword       string   `cobrafield:"vnc-password"`
	PodNetworks       []string `cobrafield:"networks"`
	PortPublish       []string `cobrafield:"publish"`
	DiskSize          uint64   `cobrafield:"disk-size"`
	VolumeSize        uint64   `cobrafield:"volume-size"`
	AppMemory         uint64   `cobrafield:"memory"`
	VolumeType        string   `cobrafield:"volume-type"`
	AppCpus           uint32   `cobrafield:"cpus"`
	ImageFormat       string   `cobrafield:"format"`
	Acl               []string `cobrafield:"acl"`
	Vlans             []string `cobrafield:"vlan"`
	SftpLoad          bool     `cobrafield:"sftp"`
	DirectLoad        bool     `cobrafield:"direct"`
	Mount             []string `cobrafield:"mount"`
	Disks             []string `cobrafield:"disks"`
	Registry          string   `cobrafield:"registry"`
	OpenStackMetadata bool     `cobrafield:"openstack-metadata"`
	Profiles          []string `cobrafield:"profile"`
	DatastoreOverride string   `cobrafield:"datastoreOverride"`
	AppAdapters       []string `cobrafield:"adapters"`
	AclOnlyHost       bool     `cobrafield:"only-host"`
}

func (s *PodConfig) FromCobra(fs *flag.FlagSet) {
	v := reflect.ValueOf(s).Elem()
	if !v.CanAddr() {
		panic("cannot assign to the item passed, item must be a pointer in order to assign")
	}
	findCobraName := func(t reflect.StructTag) (string, error) {
		if jt, ok := t.Lookup("cobrafield"); ok {
			return strings.Split(jt, ",")[0], nil
		}
		return "", fmt.Errorf("tag provided does not define a json tag")
	}
	fieldNames := map[string]int{}
	for i := 0; i < v.NumField(); i++ {
		typeField := v.Type().Field(i)
		tag := typeField.Tag
		jname, _ := findCobraName(tag)
		fieldNames[jname] = i
	}
	fs.Visit(func(f *flag.Flag) {
		fieldNum, ok := fieldNames[f.Name]
		if !ok {
			return
		}
		fieldVal := v.Field(fieldNum)
		fieldVal.Set(reflect.ValueOf(f.Value))
	})
}
