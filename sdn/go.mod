module github.com/lf-edge/eden/sdn

go 1.16

require (
	github.com/elazarl/goproxy v0.0.0-20220529153421-8ea89ba92021
	github.com/gorilla/mux v1.8.0
	github.com/lf-edge/eve/libs/depgraph v0.0.0-20220711144346-0659e3b03496
	github.com/lf-edge/eve/libs/reconciler v0.0.0-20220711144346-0659e3b03496
	github.com/sirupsen/logrus v1.8.1
	github.com/vishvananda/netlink v1.1.1-0.20210924202909-187053b97868
)

replace github.com/lf-edge/eve/libs/depgraph => github.com/lf-edge/eve/libs/depgraph v0.0.0-20220711144346-0659e3b03496
