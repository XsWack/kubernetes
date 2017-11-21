package qoscontroller

import (
	"testing"
	"encoding/json"
	"bytes"
	"net/http"
	"io/ioutil"
)

type NodeMetricPoint struct {
	Value     interface{}       `json:"value"`
	Tags      map[string]string `json:"tags"`
	Timestamp int64             `json:"timestamp"`
}

type NodeMetricResp struct {
	MetricName  string                       `json:"metric_name"`
	MetricValue map[string][]*NodeMetricPoint `json:"metric_value"`
}

func TestDecodeNodeRealLatencyFromString(t *testing.T) {
	var nodeMetricResp *NodeMetricResp
	var nodeMetricValue []*NodeMetricPoint
	s := "{\"metric_name\":\"latency\",\"metric_value\":{\"129.188.37.75\":[{\"value\":0,\"tags\":{\"ip\":\"129.188.37.75\",\"transaction_type\":\"/\"},\"timestamp\":1473250120}]}}"
	b := bytes.NewBufferString(s)
	err := json.NewDecoder(b).Decode(&nodeMetricResp)

	if err != nil {
		t.Log("Failed decode node real latency from string: ", err)
	} else {
		t.Log("nodeMetricResp: ", *nodeMetricResp)
	}

	nodeMetricValue = nodeMetricResp.MetricValue["129.188.37.75"]
	t.Log("nodeMetricValue: ", nodeMetricValue)
	podRealLatency := nodeMetricValue[0].Value.(float64)
	t.Log("podRealLatency: ", podRealLatency)
}

func TestGetNodeRealLatency(t *testing.T) {
	var nodeMetricResp *NodeMetricResp
	hostIP := "10.162.215.149"
	opsAgentEndpoint := "http://" + hostIP + ":11808/api/v1/pod-metrics/latency"
	req, err := http.NewRequest("GET", opsAgentEndpoint, nil)

	if err != nil {
		t.Log("Failed http NewRequest to ops agent for node real latency: ", err)
		return
	}

	t.Log("Http request to ops agent for node real latency: ", req)
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)

	if err != nil {
		t.Log("Failed httpClient.Do for node real latency: ", err)
		return
	}

	t.Log("Http response from ops agent for node real latency: ", resp)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errInfo, _ := ioutil.ReadAll(resp.Body)
		t.Logf("Failed get node real latency with code: %d and info: %s", resp.StatusCode, string(errInfo))
		return
	}

	t.Log("Http response body from ops agent for node real latency: ", resp.Body)
	err = json.NewDecoder(resp.Body).Decode(&nodeMetricResp)

	if err != nil {
		t.Log("Failed decode node real latency from ops agent: ", err)
		return
	}

	podMetricValue := nodeMetricResp.MetricValue["192.168.0.9"]

	if podMetricValue == nil {
		t.Log("No pod metric value from pod ")
	} else {
		t.Log("Get pod real latency from ops agent: ", podMetricValue[len(podMetricValue)-1].Value.(float64))
	}
}

func TestUpdateActionList(t *testing.T) {

	var secondaryPods []string
	var actionList []string
	pod1 := "pod1"
	secondaryPods = append(secondaryPods, pod1)
	pod2 := "pod2"
	secondaryPods = append(secondaryPods, pod2)
	i := 0
	processMultiPod := false

	//Check the secondary pods to be killed
	for _, pod := range secondaryPods {

		//Populate the action list with the secondary pod to be killed
		actionList = append(actionList, "KillPod")
		i++
		t.Log("Secondary Pod %v added to action list", pod)

		//Check if the option of killing multiple secondary pods is enabled
		if processMultiPod == false {
			t.Log("processMultiPod is: ", processMultiPod)
			t.Log("Return secondary pods: ", secondaryPods[i:])
		}
	}

	t.Log("Return secondary pods: ", secondaryPods[i:])
}
