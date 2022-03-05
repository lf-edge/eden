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
	"https://cloud-images.ubuntu.com/releases/impish/release-20220201/ubuntu-21.10-server-cloudimg-amd64.img": {
		Size:   585105408,
		Sha256: "73fe1785c60edeb506f191affff0440abcc2de02420bb70865d51d0ff9b28223",
	},
	"https://cloud-images.ubuntu.com/releases/impish/release-20220201/ubuntu-21.10-server-cloudimg-arm64.img": {
		Size:   562692096,
		Sha256: "1b5b3fe616e1eea4176049d434a360344a7d471f799e151190f21b0a27f0b424",
	},
	"http://download.cirros-cloud.net/0.5.2/cirros-0.5.2-x86_64-disk.img": {
		Size:   16300544,
		Sha256: "932fcae93574e242dc3d772d5235061747dfe537668443a1f0567d893614b464",
	},
	"http://download.cirros-cloud.net/0.5.2/cirros-0.5.2-aarch64-disk.img": {
		Size:   16872448,
		Sha256: "889c1117647b3b16cfc47957931c6573bf8e755fc9098fdcad13727b6c9f2629",
	},
}
