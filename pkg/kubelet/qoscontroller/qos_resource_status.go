package qoscontroller

import (
	"k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
	"k8s.io/kubernetes/pkg/types"
	"time"
)

// QosMonitor is passed to all the controllers sequentially and finally to the executer
type QosResourceStatus struct {
	// the pods that will be killed, frozen or unfrozen in executor
	ActionList []*Action
	// cAdvisor interface from the kubelet
	cadvisor cadvisor.Interface
	// the current active pods
	activePods []*api.Pod
	// the pods that will be unfreezed in executor
	UnfreezePodList []*api.Pod
	// the current frozen pods
	FrozenPodList []*api.Pod
	// the current primary pods
	primaryPodList []*api.Pod
	// the current secondary pods
	secondaryPodList []*api.Pod
	// the current best effort pods
	bestEffortPodList []*api.Pod
	// node level resource summary extracted from cAdvisor
	nodeResourceSummary ResourceSummary
	// maps each pod to the ResourceSummary associated with it extracted from cAdvisor
	podResourceSummary map[types.UID]*ResourceSummary
	// policy settings, like thresholds
	QosConfig *QosConfig
	// flag whether to process next controller
	processNextController bool
	// kubelet client
	kubeClient clientset.Interface
}

// Initialize method initializes the pods and cAdvisor interface
func NewQosResourceStatus(qosConfig *QosConfig, cadvisor cadvisor.Interface) *QosResourceStatus {
	return &QosResourceStatus{
		cadvisor:           cadvisor,
		QosConfig:          qosConfig,
		podResourceSummary: make(map[types.UID]*ResourceSummary),
	}
}

func (qosResourceStatus *QosResourceStatus) InititalizeQosResourceStatus(pods []*api.Pod,
	kubeClient clientset.Interface) *QosResourceStatus {

	qosResourceStatus.activePods = pods
	qosResourceStatus.kubeClient = kubeClient
	return qosResourceStatus
}

// Action contains the target pod and the corrective function associated with that pod.
// There is a list of actions named actionList which will be passed to all the controllers sequentially and then to the executor
type Action struct {
	Target     *api.Pod //the action target
	ActionType ActionType
}

// ActionType is an enum for the corrective function like kill, freeze and unfreeze
type ActionType int8

const (
	KillPod ActionType = iota
	FreezePod
	UnfreezePod
)

// FrozenPod contains the details of the frozen pod and the last resource used by that pod
type FrozenPod struct {
	pod               api.Pod
	lastResourceUsage *api.ResourceList
}

// QosConfig stores the extracted details from the config file using readConfig() method
type QosConfig struct {
	StartQosMonitor    bool          `json:"StartQosMonitor"`
	MonitoringInterval time.Duration `json:"MonitoringInterval"`
	MemoryConfig       struct {
		NodeLowMemoryRequestThresholdRate  float64 `json:"NodeLowMemoryRequestThresholdRate"`
		NodeHighMemoryRequestThresholdRate float64 `json:"NodeHighMemoryRequestThresholdRate"`
		PodMemoryThresholdRate             float64 `json:"PodMemoryThresholdRate"`
		NodeNumMemorySamples               int     `json:"NodeNumMemorySamples"`
		PodNumMemorySamples                int     `json:"PodNumMemorySamples"`
		ProcessMultiPod                    bool    `json:"ProcessMultiPod"`
	} `json:"memoryConfig"`
	NetworkIOConfig struct {
		NodeNetworkInterfaceName   string  `json:"NodeNetworkInterfaceName"`
		NodeNetworkIOCapacity      uint64  `json:"NodeNetworkIOCapacity"`
		NodeNetworkIOThresholdRate float64 `json:"NodeNetworkIOThresholdRate"`
		PodNetworkIOThresholdRate  float64 `json:"PodNetworkIOThresholdRate"`
		PodNumNetworkIOSamples     float64 `json:"PodNumNetworkIOSamples"`
		ProcessMultiPod            bool    `json:"ProcessMultiPod"`
	} `json:"networkIOConfig"`
	DiskIOConfig struct {
		NodeDiskIOThresholdRate float64 `json:"NodeDiskIOThresholdRate"`
		PodDiskIOThresholdRate  float64 `json:"PodDiskIOThresholdRate"`
		PodNumDiskIOSamples     float64 `json:"PodNumDiskIOSamples"`
		ProcessMultiPod         bool    `json:"ProcessMultiPod"`
	} `json:"diskIOConfig"`
	SlaConfig struct {
		NodeLatencyThresholdRate float64 `json:"NodeLatencyThresholdRate"`
		PodLatencyThresholdRate  float64 `json:"PodLatencyThresholdRate"`
		ProcessMultiPod          bool    `json:"ProcessMultiPod"`
		OpsAgentApiPath          string  `json:"OpsAgentApiPath"`
	} `json:"slaConfig"`
}
