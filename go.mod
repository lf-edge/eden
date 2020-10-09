module github.com/lf-edge/eden

go 1.12

require (
	github.com/Microsoft/hcsshim v0.8.9 // indirect
	github.com/Shopify/logrus-bugsnag v0.0.0-20171204204709-577dee27f20d // indirect
	github.com/amitbet/vncproxy v0.0.0-20200118084310-ea8f9b510913
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bshuster-repo/logrus-logstash-hook v1.0.0 // indirect
	github.com/bugsnag/bugsnag-go v1.5.3 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/containerd v1.3.2
	github.com/containerd/continuity v0.0.0-20200710164510-efbc4488d8fe // indirect
	github.com/deislabs/oras v0.8.1
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/garyburd/redigo v1.6.2 // indirect
	github.com/go-redis/redis/v7 v7.2.0
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/golang/protobuf v1.4.0-rc.4.0.20200313231945-b860323f09d0
	github.com/google/go-containerregistry v0.0.0-20200331213917-3d03ed9b1ca2
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/lf-edge/adam v0.0.0-20200910222414-5a4c90bf19c2
	github.com/lf-edge/eden/eserver v0.0.0-00010101000000-000000000000
	github.com/lf-edge/edge-containers v0.0.0-20200921124327-cb84b924624b
	github.com/lf-edge/eve/api/go v0.0.0-20201009050143-0ca6703afd1e
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40 // indirect
	github.com/mcuadros/go-lookup v0.0.0-20200513230334-5988786b5617
	github.com/nerd2/gexto v0.0.0-20190529073929-39468ec063f6
	github.com/rn/iso9660wrap v0.0.0-20180101235755-3a04f8ca150a
	github.com/rogpeppe/go-internal v1.6.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.7 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	google.golang.org/api v0.13.0
	gopkg.in/errgo.v2 v2.1.0
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
	rsc.io/letsencrypt v0.0.3 // indirect
)

replace github.com/lf-edge/eden/eserver => ./eserver

replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20190205005809-0d3efadf0154
