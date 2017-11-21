package qoscontroller

type QosController interface {
	process(qosResourceStatus *QosResourceStatus) error
	// Initialization function for the controllers
	initialize(qosResourceStatus *QosResourceStatus) error
}

type QosMonitor struct {
}

func NewQosMonitor() *QosMonitor {
	return &QosMonitor{}
}

func (qosMonitor QosMonitor) StartQosMonitor(qosResourceStatus *QosResourceStatus) error {
	run(qosResourceStatus)
	return nil
}

// Instance of Overall Controller
var oc OverallController

// Instance of Resource Summary
var rs ResourceSummary

// Instance of Memory Controller
var mc MemController

// Instance of Network IO Controller
var nc NetworkIOController

// Instance of Disk IO Controller
var dc DiskIOController

// Instance of SLA Controller
var sc SlaController

// function to run all the controllers sequentially and then to run the executor
func run(qosResourceStatus *QosResourceStatus) error {
	//Call Metrics Aquisition
	rs.qosAcquireMetrics(qosResourceStatus)
	// Call Overall Controller
	oc.initialize(qosResourceStatus)
	oc.process(qosResourceStatus)
	if qosResourceStatus.processNextController == false {
		return nil
	}
	// Call Memory Controller
	mc.process(qosResourceStatus)
	// Call Network IO Controller
	nc.process(qosResourceStatus)
	// Call Disk IO Controller
	dc.process(qosResourceStatus)
	// Call SLA Controller
	sc.process(qosResourceStatus)
	return nil
}
