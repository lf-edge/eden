package defaults

//ImageOptions indicates parameters of predefined VMs
type ImageOptions struct {
	Size   int64
	Sha256 string
}

//ImageStore contains image options for images used inside tests
var ImageStore = map[string]*ImageOptions{
	"https://cloud-images.ubuntu.com/releases/groovy/release-20201022.1/ubuntu-20.10-server-cloudimg-amd64.img": {
		Size:   558760448,
		Sha256: "ef3ed6aaf9c8fe1d063d556ace6c4dfbb51920d12ba8312e09a1baf3b3eedf3d",
	},
	"https://cloud-images.ubuntu.com/releases/groovy/release-20201022.1/ubuntu-20.10-server-cloudimg-arm64.img": {
		Size:   525336576,
		Sha256: "c64a5e20dd61cc112de2a47d8b0a3ec30a553fe5fe54ca0a5f83c840778aa300",
	},
	"https://cloud-images.ubuntu.com/releases/groovy/release-20210108/ubuntu-20.10-server-cloudimg-amd64.img": {
		Size:   562233344,
		Sha256: "655aac7749c7465137bfb0d21d5e9af779b56b168d47ab497dfb4a5c152c308f",
	},
	"https://cloud-images.ubuntu.com/releases/groovy/release-20210108/ubuntu-20.10-server-cloudimg-arm64.img": {
		Size:   528285696,
		Sha256: "076f86f027daddb1d48c92eba3fcb81f7e8f1512a86e051a5fdd9906671a92ca",
	},
	"https://cloud-images.ubuntu.com/releases/21.10/release-20211103/ubuntu-21.10-server-cloudimg-amd64.img": {
		Size:   568262656,
		Sha256: "4090e8317d9eecfddd0ca8c6fd11792209eaf13347d954e6e6119b0d41c69c41",
	},
	"https://cloud-images.ubuntu.com/releases/21.10/release-20211103/ubuntu-21.10-server-cloudimg-arm64.img": {
		Size:   549191680,
		Sha256: "063fa690f56b11f6bb12eecabd9e140a82faa05e26aed89cc1b7efa4c86e3bf2",
	},
}
