package projects

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ProcInfoFunc func(info *info.ZInfoMsg) error
type ProcLogFunc func(log *elog.LogItem) error
type ProcMetricFunc func(metric *metrics.ZMetricMsg) error
type ProcTimerFunc func() error

type absFunc struct {
	disabled bool
	states   bool
	proc     interface{}
}

type processingBus struct {
	tc   *TestContext
	wg   *sync.WaitGroup
	proc map[*device.Ctx][]*absFunc
}

func initBus(tc *TestContext) *processingBus {
	return &processingBus{tc: tc, proc: map[*device.Ctx][]*absFunc{}, wg: &sync.WaitGroup{}}
}

func (lb *processingBus) clean() {
	for _, funcs := range lb.proc {
		for _, el := range funcs {
			if el.states || el.disabled {
				continue
			}
			el.disabled = true
		}
	}
	lb.wg = &sync.WaitGroup{}
}

func (lb *processingBus) processReturn(edgeNode *device.Ctx, procFunc *absFunc, result error) {
	if result != nil {
		procFunc.disabled = true
		lb.wg.Done()
		toRet := fmt.Sprintf("%T done with return: \"%s\"", procFunc.proc, result)
		if t, ok := lb.tc.tests[edgeNode]; ok {
			t.Log(toRet)
		} else {
			log.Println(toRet)
		}
	}
}

func (lb *processingBus) process(edgeNode *device.Ctx, inp interface{}) bool {
	for node, functions := range lb.proc {
		if node == edgeNode {
			for _, procFunc := range functions {
				if procFunc.disabled == false {
					switch pf := procFunc.proc.(type) {
					case ProcInfoFunc:
						el, match := inp.(*info.ZInfoMsg)
						if match {
							lb.processReturn(edgeNode, procFunc, pf(el))
						}
					case ProcLogFunc:
						el, match := inp.(*elog.LogItem)
						if match {
							lb.processReturn(edgeNode, procFunc, pf(el))
						}
					case ProcMetricFunc:
						el, match := inp.(*metrics.ZMetricMsg)
						if match {
							lb.processReturn(edgeNode, procFunc, pf(el))
						}
					}
				}
			}
		}
	}
	return false
}

func (lb *processingBus) processTimers(edgeNode *device.Ctx) bool {
	for node, functions := range lb.proc {
		if node == edgeNode {
			for _, procFunc := range functions {
				if procFunc.disabled == false {
					switch pf := procFunc.proc.(type) {
					case ProcTimerFunc:
						lb.processReturn(edgeNode, procFunc, pf())
					}
				}
			}
		}
	}
	return false
}

func (lb *processingBus) getMainProcessorLog(dev *device.Ctx) elog.HandlerFunc {
	return func(im *elog.LogItem) bool {
		return lb.process(dev, im)
	}
}

func (lb *processingBus) getMainProcessorInfo(dev *device.Ctx) einfo.HandlerFunc {
	return func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface, infoType einfo.ZInfoType) bool {
		return lb.process(dev, im)
	}
}

func (lb *processingBus) getMainProcessorMetric(dev *device.Ctx) emetric.HandlerFunc {
	return func(msg *metrics.ZMetricMsg) bool {
		return lb.process(dev, msg)
	}
}

func (lb *processingBus) initCheckers(dev *device.Ctx) {
	go func() {
		err := lb.tc.GetController().LogChecker(dev.GetID(), map[string]string{}, lb.getMainProcessorLog(dev), elog.LogNew, 0)
		if err != nil {
			log.Errorf("LogChecker for dev %s error %s", dev.GetID(), err)
		}
	}()
	go func() {
		err := lb.tc.GetController().InfoChecker(dev.GetID(), map[string]string{}, einfo.ZAll, lb.getMainProcessorInfo(dev), einfo.InfoNew, 0)
		if err != nil {
			log.Errorf("InfoChecker for dev %s error %s", dev.GetID(), err)
		}
	}()
	go func() {
		err := lb.tc.GetController().MetricChecker(dev.GetID(), map[string]string{}, lb.getMainProcessorMetric(dev), emetric.MetricNew, 0)
		if err != nil {
			log.Errorf("MetricChecker for dev %s error %s", dev.GetID(), err)
		}
	}()
	go func() {
		for {
			timer := time.NewTimer(defaults.DefaultRepeatTimeout)
			<-timer.C
			lb.processTimers(dev)
		}
	}()
}

func (lb *processingBus) addProc(dev *device.Ctx, procFunc interface{}) {
	if _, exists := lb.proc[dev]; !exists {
		lb.initCheckers(dev)
	}
	lb.proc[dev] = append(lb.proc[dev], &absFunc{proc: procFunc, disabled: false})
	lb.wg.Add(1)
}
