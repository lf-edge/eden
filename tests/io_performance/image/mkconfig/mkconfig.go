package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func contains(hs []string, val string) bool {
	for _, v := range hs {
		if v == val {
			return true
		}
	}
	return false
}

func containsInt(hs []int, val int) bool {
	for _, v := range hs {
		if v == val {
			return true
		}
	}
	return false
}

type tType []string

func (t *tType) Set(v string) error {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	v = strings.ToLower(v)
	var val = strings.Split(v, ",")
	*t = []string{}
	var valid = []string{"read", "write", "randread", "randwrite", "readwrite"}
	for _, s := range val {
		if !contains(valid, s) {
			return errors.New("Invalid value for type " + s)
		}
		*t = append(*t, s)
	}

	return nil
}

func (t tType) String() string {
	return strings.Join(t, ",")
}

type blockSize []string

func (bs *blockSize) Set(v string) error {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	v = strings.ToLower(v)
	var val = strings.Split(v, ",")
	var valid = []string{"512", "1k", "2k", "4k", "8k", "16k", "32k", "64k", "128k", "256k", "512k", "1m"}
	*bs = []string{}
	for _, s := range val {
		if !contains(valid, s) {
			return errors.New("Invalid value for block size " + s)
		}
		*bs = append(*bs, s)
	}

	return nil
}

func (bs blockSize) String() string {
	return strings.Join(bs, ",")
}

type jobs []int

func (j *jobs) Set(v string) error {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	v = strings.ToLower(v)
	var val = strings.Split(v, ",")
	*j = []int{}
	var valid = []int{1, 4, 8, 16, 32}
	for _, s := range val {
		n, err := strconv.Atoi(s)
		if err != nil || !containsInt(valid, n) {
			return errors.New("Invalid value for jobs " + s)
		}
		*j = append(*j, n)
	}

	return nil
}

func (j jobs) String() string {
	var sVal []string
	for _, n := range j {
		sVal = append(sVal, strconv.Itoa(n))
	}
	return strings.Join(sVal, ",")
}

type depth []int

func (d *depth) Set(v string) error {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	v = strings.ToLower(v)
	var val = strings.Split(v, ",")
	*d = []int{}
	var valid = []int{1, 4, 8, 16, 32}
	for _, s := range val {
		n, err := strconv.Atoi(s)
		if err != nil || !containsInt(valid, n) {
			return errors.New("Invalid value for depth " + s)
		}
		*d = append(*d, n)
	}

	return nil
}

func (d depth) String() string {
	var sVal []string
	for _, n := range d {
		sVal = append(sVal, strconv.Itoa(n))
	}
	return strings.Join(sVal, ",")
}

var fType = tType{"read", "write"}
var fBS = blockSize{"4k", "64k", "1m"}
var fJobs = jobs{1, 8}
var fDepth = depth{1, 8, 32}
var fTime string
var outPath string

func init() {
	flag.Var(&fType, "type", "Use comma separated string with combinations of read, write, randread, randwrite ...")
	flag.Var(&fBS, "bs", "Use comma separated string with combinations of 4k,8k,16k,64k ...")
	flag.Var(&fJobs, "jobs", "Use comma separated string with combinations of int values")
	flag.Var(&fDepth, "depth", "Use comma separated string with combinations of int values")
	flag.StringVar(&fTime, "time", "60", "Use seconds to pass execution time")
	flag.StringVar(&outPath, "out", "./config.fio", "Change output file path")
	flag.Parse()
}

const globalTpl = `[global]
ioengine=libaio
size=1G
direct=1
runtime=%s
time_based=1
group_reporting
filename=%s
`

const globalTplcheckSumm = `[global]
ioengine=libaio
size=1G
direct=1
runtime=%s
verify=%s
verify_fatal=1
time_based=1
group_reporting
filename=%s
`

const sectionTpl = `
[%s]
rw=%s
bs=%s
iodepth=%d
numjobs=%d
stonewall
`

func main() {
	var countTests = len(fType) * len(fBS) * len(fDepth) * len(fJobs)
	var ftPath = "/data/fio.test.file"

	path, exists := os.LookupEnv("FOLDER_GIT")
	if exists {
		ftPath = fmt.Sprintf("/data/%s/fio.test.file", path)
	}

	fmt.Fprintln(os.Stderr, "type:", fType)
	fmt.Fprintln(os.Stderr, "bs:", fBS)
	fmt.Fprintln(os.Stderr, "jobs:", fJobs)
	fmt.Fprintln(os.Stderr, "depth:", fDepth)
	fmt.Fprintln(os.Stderr, "time:", fTime)
	fmt.Fprintln(os.Stderr, "Total tests:", countTests)
	fmt.Fprint(os.Stdout, countTests)

	fd, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	if fTime == "" {
		fTime = "60"
	}

	verify, exists := os.LookupEnv("FIO_CHECKSUMM")
	if exists {
		fmt.Fprintf(fd, globalTplcheckSumm, fTime, verify, ftPath)
	} else {
		fmt.Fprintf(fd, globalTpl, fTime, ftPath)
	}

	for _, rw := range fType {
		for _, bs := range fBS {
			var count = 0
			for _, depth := range fDepth {
				for _, job := range fJobs {
					var section = fmt.Sprintf("%s-%s-%d", rw, bs, count)
					fmt.Fprintf(fd, sectionTpl, section, rw, bs, depth, job)
					count++
				}
			}
		}
	}

}
