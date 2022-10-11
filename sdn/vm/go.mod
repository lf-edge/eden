module github.com/lf-edge/eden/sdn/vm

go 1.16

require (
	github.com/elazarl/goproxy v0.0.0-20220529153421-8ea89ba92021
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2
	github.com/gorilla/mux v1.8.0
	github.com/inconshreveable/go-vhost v1.0.0
	github.com/lf-edge/eve/libs/depgraph v0.0.0-20220711144346-0659e3b03496
	github.com/lf-edge/eve/libs/reconciler v0.0.0-20220711144346-0659e3b03496
	github.com/sirupsen/logrus v1.8.1
	github.com/vishvananda/netlink v1.1.1-0.20210924202909-187053b97868
	github.com/vishvananda/netns v0.0.0-20200728191858-db3c7e526aae
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f
)

replace github.com/lf-edge/eve/libs/depgraph => github.com/lf-edge/eve/libs/depgraph v0.0.0-20220711144346-0659e3b03496
