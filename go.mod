module github.com/lf-edge/eden

go 1.12

require (
	github.com/amitbet/vncproxy v0.0.0-20200118084310-ea8f9b510913
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20190924003213-a8608b5b67c7
	github.com/docker/go-connections v0.4.0
	github.com/dustin/go-humanize v1.0.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-redis/redis/v7 v7.2.0
	github.com/golang/protobuf v1.4.0-rc.4.0.20200313231945-b860323f09d0
	github.com/google/go-containerregistry v0.0.0-20200331213917-3d03ed9b1ca2
	github.com/lf-edge/adam v0.0.0-20200502191324-bedd2e5e0d61
	github.com/lf-edge/eden/eserver v0.0.0-00010101000000-000000000000
	github.com/lf-edge/eve/api/go v0.0.0-20200805060635-507771c7f963
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40 // indirect
	github.com/mcuadros/go-lookup v0.0.0-20200513230334-5988786b5617
	github.com/nerd2/gexto v0.0.0-20190529073929-39468ec063f6
	github.com/rogpeppe/go-internal v1.6.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	gopkg.in/errgo.v2 v2.1.0
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
)

replace github.com/lf-edge/eden/eserver => ./eserver
