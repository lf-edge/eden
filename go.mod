module github.com/lf-edge/eden

go 1.16

require (
	github.com/Insei/rolgo v0.0.2
	github.com/amitbet/vncproxy v0.0.0-20200118084310-ea8f9b510913
	github.com/bugsnag/bugsnag-go v1.5.3 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/cgroups v1.0.4 // indirect
	github.com/containerd/containerd v1.6.6
	github.com/containerd/continuity v0.3.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.12.0 // indirect
	github.com/docker/distribution v2.8.1+incompatible
	github.com/docker/docker v20.10.17+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.13.0
	github.com/fsnotify/fsnotify v1.5.4
	github.com/go-redis/redis/v9 v9.0.0-beta.1
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-containerregistry v0.10.0
	github.com/google/uuid v1.3.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/lf-edge/eden/eserver v0.0.0-20220711180217-6e2bfa9c3f67
	github.com/lf-edge/eden/sdn v0.0.0-00010101000000-000000000000
	github.com/lf-edge/edge-containers v0.0.0-20220320131500-9d9f95d81e2c
	github.com/lf-edge/eve/api/go v0.0.0-20220922050101-e6c69cc97282
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40 // indirect
	github.com/mcuadros/go-lookup v0.0.0-20200831155250-80f87a4fa5ee
	github.com/moby/sys/mount v0.3.3 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6
	github.com/nerd2/gexto v0.0.0-20190529073929-39468ec063f6
	github.com/opencontainers/runc v1.1.3 // indirect
	github.com/packethost/packngo v0.25.0
	github.com/prometheus/client_golang v1.12.2 // indirect
	github.com/prometheus/common v0.36.0 // indirect
	github.com/rogpeppe/go-internal v1.6.2
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.5.0
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.7.2
	github.com/thediveo/enumflag v0.10.1
	github.com/tmc/scp v0.0.0-20170824174625-f7b48647feef
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.7 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d
	golang.org/x/net v0.0.0-20220708220712-1185a9018129
	golang.org/x/oauth2 v0.0.0-20220630143837-2104d58473e0
	golang.org/x/sys v0.0.0-20220712014510-0a85c31ab51e // indirect
	golang.org/x/term v0.0.0-20220526004731-065cf7ba2467
	google.golang.org/api v0.86.0
	google.golang.org/genproto v0.0.0-20220712132514-bdd2acd4974d // indirect
	google.golang.org/protobuf v1.28.0
	gopkg.in/errgo.v2 v2.1.0
	gopkg.in/yaml.v2 v2.4.0
	oras.land/oras-go v1.2.0
)

replace github.com/lf-edge/eden/sdn => ./sdn

replace github.com/lf-edge/eve/libs/depgraph => github.com/lf-edge/eve/libs/depgraph v0.0.0-20220711144346-0659e3b03496

replace github.com/lf-edge/eve/api/go => github.com/milan-zededa/eve/api/go v0.0.0-20220823085700-482fb4b20aaa
