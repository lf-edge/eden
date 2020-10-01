module github.com/lf-edge/eden

go 1.12

require (
	github.com/Microsoft/hcsshim v0.8.9 // indirect
	github.com/amitbet/vncproxy v0.0.0-20200118084310-ea8f9b510913
	github.com/containerd/continuity v0.0.0-20200710164510-efbc4488d8fe // indirect
	github.com/docker/cli v0.0.0-20200130152716-5d0cf8839492 // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-redis/redis/v7 v7.2.0
	github.com/golang/protobuf v1.4.0-rc.4.0.20200313231945-b860323f09d0
	github.com/google/go-containerregistry v0.0.0-20200331213917-3d03ed9b1ca2
	github.com/lf-edge/adam v0.0.0-20200910222414-5a4c90bf19c2
	github.com/lf-edge/eden/eserver v0.0.0-00010101000000-000000000000
	github.com/lf-edge/edge-containers v0.0.0-20200921124327-cb84b924624b
	github.com/lf-edge/eve/api/go v0.0.0-20200929132246-e32375a3e2ab
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40 // indirect
	github.com/mcuadros/go-lookup v0.0.0-20200513230334-5988786b5617
	github.com/nerd2/gexto v0.0.0-20190529073929-39468ec063f6
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rn/iso9660wrap v0.0.0-20180101235755-3a04f8ca150a
	github.com/rogpeppe/go-internal v1.6.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sys v0.0.0-20200120151820-655fe14d7479 // indirect
	google.golang.org/api v0.13.0
	google.golang.org/genproto v0.0.0-20200117163144-32f20d992d24 // indirect
	google.golang.org/grpc v1.27.0 // indirect
	gopkg.in/errgo.v2 v2.1.0
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
)

replace github.com/lf-edge/eden/eserver => ./eserver

replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20190205005809-0d3efadf0154
