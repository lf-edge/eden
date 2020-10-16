package api

//URLArg is packet to send into eserver for downloading of external file
type URLArg struct {
	//URL contains link to file
	URL string `json:"url,omitempty"`
}

//FileInfo contains information about downloading or downloaded file
type FileInfo struct {
	//Sha256 of file
	Sha256 string `json:"sha256,omitempty"`
	//Size of file in bytes
	Size int64 `json:"size,omitempty"`
	//FileName is link for access file
	FileName string `json:"filename,omitempty"`
	//ISReady indicates status of image
	ISReady bool `json:"ready"`
	//Error contains errors
	Error string `json:"error,omitempty"`
}
