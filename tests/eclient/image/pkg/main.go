package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/lf-edge/eve/api/go/profile"
	"google.golang.org/protobuf/proto"
)

const (
	contentType = "Content-Type"
	mimeProto   = "application/x-proto-binary"
)

var (
	profileFile = flag.String("file", "/mnt/profile", "File with current profile")
	token       = flag.String("token", "", "Token of profile server")
)

func main() {
	flag.Parse()
	http.HandleFunc("/api/v1/local_profile", localProfile)
	fmt.Println(http.ListenAndServe(":8888", nil))
}

func localProfile(w http.ResponseWriter, _ *http.Request) {
	profileFromFile, err := ioutil.ReadFile(*profileFile)
	if err != nil {
		errStr := fmt.Sprintf("ReadFile: %s", err)
		fmt.Println(errStr)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	localProfileObject := &profile.LocalProfile{
		LocalProfile: strings.TrimSpace(string(profileFromFile)),
		ServerToken:  *token,
	}
	data, err := proto.Marshal(localProfileObject)
	if err != nil {
		errStr := fmt.Sprintf("Marshal: %s", err)
		fmt.Println(errStr)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	w.Header().Set(contentType, mimeProto)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		fmt.Printf("Failed to write: %s\n", err)
	}
}
