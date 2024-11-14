package testcontext

import (
	"fmt"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"

	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/controller/eflowlog"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/flowlog"
	"github.com/lf-edge/eve-api/go/info"
	"github.com/lf-edge/eve-api/go/logs"
	"github.com/lf-edge/eve-api/go/metrics"
)

// ProcInfoFunc provides callback to process info
type ProcInfoFunc func(info *info.ZInfoMsg) error

// ProcLogFunc provides callback to process log
type ProcLogFunc func(log *elog.FullLogEntry) error

// ProcLogFlowFunc provides callback to process flowLog
type ProcLogFlowFunc func(log *flowlog.FlowMessage) error

// ProcMetricFunc provides callback to process metric
type ProcMetricFunc func(metric *metrics.ZMetricMsg) error

// ProcAppLogFunc provides callback to process app log
type ProcAppLogFunc func(log *logs.LogEntry) error

// Callback provides callback to process
type Callback func()

// ProcTimerFunc provides callback to process on timer event
type ProcTimerFunc func() error

type absFunc struct {
	disabled bool
	states   bool
	proc     interface{}
	appUUID  uuid.UUID
}

type processingBus struct {
	tc   *TestContext
	wg   *sync.WaitGroup
	proc map[*device.Ctx][]*absFunc
}

func InitBus(tc *TestContext) *processingBus {
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
		if lb.tc.addTime != 0 {
			log.Infof("Expand timewait by %s", lb.tc.addTime)
			lb.tc.stopTime.Add(lb.tc.addTime)
		}
		procFunc.disabled = true
		toRet := utils.AddTimestamp(fmt.Sprintf("%T done with return: %s", procFunc.proc, result.Error()))
		if t, ok := lb.tc.Tests[edgeNode]; ok {
			t.Log(toRet)
		}
		log.Info(toRet)
		lb.wg.Done()
	}
}

func (lb *processingBus) process(edgeNode *device.Ctx, inp interface{}) bool {
	for node, functions := range lb.proc {
		if node == edgeNode {
			for _, procFunc := range functions {
				if !procFunc.disabled {
					switch pf := procFunc.proc.(type) {
					case ProcInfoFunc:
						el, match := inp.(*info.ZInfoMsg)
						if match {
							lb.processReturn(edgeNode, procFunc, pf(el))
						}
					case ProcLogFunc:
						el, match := inp.(*elog.FullLogEntry)
						if match {
							lb.processReturn(edgeNode, procFunc, pf(el))
						}
					case ProcMetricFunc:
						el, match := inp.(*metrics.ZMetricMsg)
						if match {
							lb.processReturn(edgeNode, procFunc, pf(el))
						}
					case ProcLogFlowFunc:
						el, match := inp.(*flowlog.FlowMessage)
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

func (lb *processingBus) processApp(edgeNode *device.Ctx, appUUID uuid.UUID, inp interface{}) bool {
	for node, functions := range lb.proc {
		if node == edgeNode {
			for _, procFunc := range functions {
				if procFunc.appUUID != appUUID {
					continue
				}
				if !procFunc.disabled {
					switch pf := procFunc.proc.(type) {
					case ProcAppLogFunc:
						el, match := inp.(*logs.LogEntry)
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
		if node != edgeNode {
			continue
		}
		for _, procFunc := range functions {
			if procFunc.disabled {
				continue
			}
			switch pf := procFunc.proc.(type) {
			case ProcTimerFunc:
				lb.processReturn(edgeNode, procFunc, pf())
			}
		}
	}
	return false
}

func (lb *processingBus) getMainProcessorLog(dev *device.Ctx) elog.HandlerFunc {
	return func(im *elog.FullLogEntry) bool {
		return lb.process(dev, im)
	}
}

func (lb *processingBus) getMainProcessorFlowLog(dev *device.Ctx) eflowlog.HandlerFunc {
	return func(msg *flowlog.FlowMessage) bool {
		return lb.process(dev, msg)
	}
}

func (lb *processingBus) getMainProcessorInfo(dev *device.Ctx) einfo.HandlerFunc {
	return func(im *info.ZInfoMsg) bool {
		return lb.process(dev, im)
	}
}

func (lb *processingBus) getMainProcessorMetric(dev *device.Ctx) emetric.HandlerFunc {
	return func(msg *metrics.ZMetricMsg) bool {
		return lb.process(dev, msg)
	}
}

func (lb *processingBus) getMainProcessorAppLog(dev *device.Ctx, appUUID uuid.UUID) eapps.HandlerFunc {
	return func(im *logs.LogEntry) bool {
		return lb.processApp(dev, appUUID, im)
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
		err := lb.tc.GetController().FlowLogChecker(dev.GetID(), map[string]string{}, lb.getMainProcessorFlowLog(dev), eflowlog.FlowLogNew, 0)
		if err != nil {
			log.Errorf("FlowLogChecker for dev %s error %s", dev.GetID(), err)
		}
	}()
	go func() {
		err := lb.tc.GetController().InfoChecker(dev.GetID(), map[string]string{}, lb.getMainProcessorInfo(dev), einfo.InfoNew, 0)
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
			timer := time.NewTimer(defaults.DefaultRepeatTimeout * 2)
			<-timer.C
			lb.processTimers(dev)
		}
	}()
}

func (lb *processingBus) initAppChecker(dev *device.Ctx, appUUID uuid.UUID) {
	for _, el := range lb.proc[dev] {
		if el.appUUID == appUUID {
			return
		}
	}
	go func() {
		err := lb.tc.GetController().LogAppsChecker(dev.GetID(), appUUID, map[string]string{}, lb.getMainProcessorAppLog(dev, appUUID), eapps.LogNew, 0)
		if err != nil {
			log.Errorf("AppLogChecker for dev %s error %s", dev.GetID(), err)
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

func (lb *processingBus) addAppProc(dev *device.Ctx, appUUID uuid.UUID, procFunc interface{}) {
	if _, exists := lb.proc[dev]; !exists {
		lb.initCheckers(dev)
	}
	lb.initAppChecker(dev, appUUID)
	lb.proc[dev] = append(lb.proc[dev], &absFunc{proc: procFunc, disabled: false, appUUID: appUUID})
	lb.wg.Add(1)
}
