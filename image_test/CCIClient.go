package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"

	"io/ioutil"
	"net/http"

	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/api/ref"
)

const CCIServerAddr = "https://cci.cn-north-1.myhuaweicloud.com"

var conn *CCIConn

func init() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	conn = &CCIConn{
		CCIServerAddr: CCIServerAddr,
		client:        &http.Client{Transport: tr},
	}
}

type CCIConn struct {
	CCIServerAddr string
	client        *http.Client
}

func (conn *CCIConn) createDeployment(deployment *apps.Deployment) (*apps.Deployment, error) {
	url := fmt.Sprintf("%s/apis/apps/v1/namespaces/%s/deployments", CCIServerAddr, deployment.Namespace)
	byteOfDp, err := json.Marshal(deployment)
	if err != nil {
		return nil, fmt.Errorf("marshal request body failed: %v", err)
	}
	request, err := CommonRequest(http.MethodPost, url, nil, bytes.NewReader(byteOfDp))
	if err != nil {
		return nil, fmt.Errorf("build request error: %v", err)
	}
	_, body, _, err := PerfromRequest(conn.client, request)
	if err != nil {
		return nil, fmt.Errorf("getPod PerfromRequest error: %v", err)
	}
	var newDp apps.Deployment
	err = json.Unmarshal(body, &newDp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal failed: %v", err)
	}
	return &newDp, nil
}

func (conn *CCIConn) deleteDeployment(deployment *apps.Deployment) error {
	url := fmt.Sprintf("%s/apis/apps/v1/namespaces/%s/deployments/%s", CCIServerAddr, deployment.Namespace, deployment.Name)
	request, err := CommonRequest(http.MethodDelete, url, nil, nil)
	if err != nil {
		return fmt.Errorf("build request error: %v", err)
	}

	_, _, _, err = PerfromRequest(conn.client, request)
	if err != nil {
		return fmt.Errorf("deleteDeployment PerfromRequest error: %v", err)
	}

	return nil
}

func (conn *CCIConn) getDeployment(deployment *apps.Deployment) (*apps.Deployment, error) {
	url := fmt.Sprintf("%s/apis/apps/v1/namespaces/%s/deployments/%s", CCIServerAddr, deployment.Namespace, deployment.Name)
	request, err := CommonRequest(http.MethodGet, url, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("build request error: %v", err)
	}

	_, body, _, err := PerfromRequest(conn.client, request)
	if err != nil {
		return nil, fmt.Errorf("getDeployment PerfromRequest error: %v", err)
	}
	var newDp apps.Deployment
	err = json.Unmarshal(body, &newDp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal failed: %v", err)
	}
	return &newDp, nil
}

func (conn *CCIConn) listPod(options metav1.ListOptions, namespace string) (*v1.PodList, error) {
	path := fmt.Sprintf("%s/api/v1/namespaces/%s/pods", CCIServerAddr, namespace)

	finalUrl := path
	if len(options.LabelSelector) > 0 {
		finalUrl = fmt.Sprintf("%s?labelSelector=%s", path, url.QueryEscape(options.LabelSelector))
	}

	request, err := CommonRequest(http.MethodGet, finalUrl, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("build request error: %v", err)
	}

	_, body, _, err := PerfromRequest(conn.client, request)
	if err != nil {
		return nil, fmt.Errorf("listPod PerfromRequest error: %v", err)
	}

	var podList v1.PodList
	err = json.Unmarshal(body, &podList)
	if err != nil {
		return nil, fmt.Errorf("listPod unmarshal failed: %v", err)
	}

	return &podList, nil
}

func (conn *CCIConn) SearchEvent(runTimeScheme *runtime.Scheme, objOrRef runtime.Object, namespace string) (*v1.EventList, error) {
	path := fmt.Sprintf("%s/api/v1/namespaces/%s/events", CCIServerAddr, namespace)
	selector, err := makeFieldSelector(runTimeScheme, objOrRef)
	if err != nil {
		return nil, err
	}
	finalUrl := fmt.Sprintf("%s?fieldSelector=%s", path, url.QueryEscape(selector.String()))
	request, err := CommonRequest(http.MethodGet, finalUrl, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("build request error: %v", err)
	}
	_, body, _, err := PerfromRequest(conn.client, request)
	if err != nil {
		return nil, fmt.Errorf("SearchEvent PerfromRequest error: %v", err)
	}

	var eventList v1.EventList
	err = json.Unmarshal(body, &eventList)
	if err != nil {
		return nil, fmt.Errorf("SearchEvent unmarshal failed: %v", err)
	}

	return &eventList, nil
}

func makeFieldSelector(scheme *runtime.Scheme, objOrRef runtime.Object) (fields.Selector, error) {
	ref, err := ref.GetReference(scheme, objOrRef)
	if err != nil {
		return nil, err
	}
	stringRefKind := string(ref.Kind)
	var refKind *string
	if stringRefKind != "" {
		refKind = &stringRefKind
	}
	stringRefUID := string(ref.UID)
	var refUID *string
	if stringRefUID != "" {
		refUID = &stringRefUID
	}

	return GetFieldSelector(&ref.Name, &ref.Namespace, refKind, refUID), nil
}

func GetFieldSelector(involvedObjectName, involvedObjectNamespace, involvedObjectKind, involvedObjectUID *string) fields.Selector {
	field := fields.Set{}
	if involvedObjectName != nil {
		field[GetInvolvedObjectNameFieldLabel()] = *involvedObjectName
	}
	if involvedObjectNamespace != nil {
		field["involvedObject.namespace"] = *involvedObjectNamespace
	}
	if involvedObjectKind != nil {
		field["involvedObject.kind"] = *involvedObjectKind
	}
	if involvedObjectUID != nil {
		field["involvedObject.uid"] = *involvedObjectUID
	}
	return field.AsSelector()
}

func GetInvolvedObjectNameFieldLabel() string {
	return "involvedObject.name"
}

func CommonRequest(
	method string,
	path string,
	headers map[string]string,
	body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("X-Auth-Token", getXAuthToken())

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

func PerfromRequest(client *http.Client, request *http.Request) (code int, buf []byte, header map[string][]string, err error) {
	var resp *http.Response

	resp, err = client.Do(request)
	if err != nil {
		return
	}

	header = resp.Header

	defer resp.Body.Close()

	code = resp.StatusCode
	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return
}

func getXAuthToken() string {
	token := os.Getenv("Token")
	if len(token) == 0 {
		panic("token is empty")
	}
	return token
}