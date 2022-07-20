package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"

	"os"
)

type fioJSON struct {
	FioVersion   string `json:"fio version"`
	GlobalConfig struct {
		IoEngine string `json:"ioengine"`
		Direct   string `json:"direct"`
	} `json:"global options"`
	Jobs []struct {
		TestName   string `json:"jobname"`
		GroupID    int    `json:"groupid"`
		TestOption struct {
			RW      string `json:"rw"`
			BS      string `json:"bs"`
			IODepth string `json:"iodepth"`
			NumJobs string `json:"numjobs"`
		} `json:"job options"`
		Read struct {
			BW       int     `json:"bw"`
			Iops     float64 `json:"iops"`
			IoKbs    int     `json:"io_kbytes"`
			Runtime  int     `json:"runtime"`
			TotalIos int     `json:"total_ios"`
			IopsMin  int     `json:"iops_min"`
			IopsMax  int     `json:"iops_max"`
			IopsMean float64 `json:"iops_mean"`
			BWMin    int     `json:"bw_min"`
			BWMax    int     `json:"bw_max"`
			BWMean   float64 `json:"bw_mean"`
		} `json:"read"`
		Write struct {
			BW       int     `json:"bw"`
			Iops     float64 `json:"iops"`
			IoKbs    int     `json:"io_kbytes"`
			Runtime  int     `json:"runtime"`
			TotalIos int     `json:"total_ios"`
			IopsMin  int     `json:"iops_min"`
			IopsMax  int     `json:"iops_max"`
			IopsMean float64 `json:"iops_mean"`
			BWMin    int     `json:"bw_min"`
			BWMax    int     `json:"bw_max"`
			BWMean   float64 `json:"bw_mean"`
		} `json:"write"`
		JobRuntime        int     `json:"job_runtime"`
		UsrCPU            float64 `json:"usr_cpu"`
		SysCPU            float64 `json:"sys_cpu"`
		Ctx               int     `json:"ctx"`
		LatencyDepth      int     `json:"latency_depth"`
		LatencyTarget     int     `json:"latency_target"`
		LatencyPercentile float64 `json:"latency_percentile"`
	} `json:"jobs"`
	DiskUtil []struct {
		DiskName    string  `json:"name"`
		ReadIos     int     `json:"read_ios"`
		WriteIos    int     `json:"write_ios"`
		ReadMerges  int     `json:"read_merges"`
		WriteMerges int     `json:"write_merges"`
		ReadTicks   int     `json:"read_ticks"`
		WriteTicks  int     `json:"write_ticks"`
		InQueue     int     `json:"in_queue"`
		Util        float64 `json:"util"`
	} `json:"disk_util"`
}

func parseJSON(in []byte) (fioJSON, error) {
	var data fioJSON
	if err := json.Unmarshal(in, &data); err != nil {
		return fioJSON{}, fmt.Errorf("invalid JSON input: %w", err)
	}
	return data, nil
}

func toFixed(x float64, n int) float64 {
	var l = math.Pow(10, float64(n))
	return math.Round(x*l) / l
}
func mbps(x int) float64 {
	return toFixed(float64(x)/1024, 2)
}

func formatCSV(in fioJSON, to io.Writer) error {
	var header = []string{
		"Group ID", "Pattern", "Block Size", "IO Depth", "Jobs", "Mb/s",
	}

	var w = csv.NewWriter(to)
	if err := w.Write(header); err != nil {
		return err
	}

	for _, v := range in.Jobs {
		var bw = v.Write.BW
		if v.TestOption.RW == "read" {
			bw = v.Read.BW
		}
		var row = []string{
			fmt.Sprintf("%v", v.GroupID),
			v.TestOption.RW,
			v.TestOption.BS,
			v.TestOption.IODepth,
			v.TestOption.NumJobs,
			fmt.Sprintf("%v", mbps(bw)),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	w.Flush()

	return nil
}

func cleanJSON(in []byte) ([]byte, error) {
	var begin = bytes.IndexAny(in, "{")
	var end = bytes.LastIndexAny(in, "}") + 1
	if begin >= end {
		return nil, errors.New("incorrect input format")
	}
	return in[begin:end], nil
}

func main() {
	if len(os.Args) < 2 {
		panic("not enough arguments")
	}
	var inputFile = os.Args[1]
	var outputFile = os.Args[2]
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		panic(err)
	}

	text, err := cleanJSON(data)
	if err != nil {
		panic(err)
	}
	obj, err := parseJSON(text)
	if err != nil {
		panic(err)
	}

	fd, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	if err := formatCSV(obj, fd); err != nil {
		panic(err)
	}
}
