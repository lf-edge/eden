module github.com/lf-edge/eden

go 1.15

require (
	cloud.google.com/go v0.74.0 // indirect
	github.com/Insei/rolgo v0.0.1
	github.com/Shopify/logrus-bugsnag v0.0.0-20171204204709-577dee27f20d // indirect
	github.com/amitbet/vncproxy v0.0.0-20200118084310-ea8f9b510913
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bshuster-repo/logrus-logstash-hook v1.0.0 // indirect
	github.com/bugsnag/bugsnag-go v1.5.3 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/containerd v1.5.7
	github.com/deislabs/oras v0.8.2-0.20201110191325-f1caa175232f
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.0+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.10.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/garyburd/redigo v1.6.2 // indirect
	github.com/go-redis/redis/v7 v7.4.0
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/go-containerregistry v0.2.1
	github.com/google/uuid v1.2.0
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/lf-edge/eden/eserver v0.0.0-20201210161141-8551a3b0751b
	github.com/lf-edge/edge-containers v0.0.0-20210630151415-7dbb4f290dab
	github.com/lf-edge/eve/api/go v0.0.0-20211019025616-e596c9ebf245
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mcuadros/go-lookup v0.0.0-20200831155250-80f87a4fa5ee
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/moby/term v0.0.0-20201110203204-bea5bbe245bf
	github.com/nerd2/gexto v0.0.0-20190529073929-39468ec063f6
	github.com/packethost/packngo v0.20.0
	github.com/pelletier/go-toml v1.9.0 // indirect
	github.com/prometheus/client_golang v1.8.0 // indirect
	github.com/prometheus/common v0.15.0 // indirect
	github.com/rogpeppe/go-internal v1.6.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/tmc/scp v0.0.0-20170824174625-f7b48647feef
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.7 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/net v0.0.0-20211029224645-99673261e6eb
	golang.org/x/oauth2 v0.0.0-20201208152858-08078c50e5b5
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf
	google.golang.org/api v0.36.0
	google.golang.org/genproto v0.0.0-20201211151036-40ec1c210f7a // indirect
	google.golang.org/protobuf v1.26.0
	gopkg.in/errgo.v2 v2.1.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/lf-edge/eden/eserver => ./eserver

replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20190205005809-0d3efadf0154
