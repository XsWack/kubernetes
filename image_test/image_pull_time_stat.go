package main

import (
	"bufio"
	"fmt"
	"github.com/spf13/cobra"
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const PULLING = "Pulling"
const PULLED = "Pulled"

// SortableEvents implements sort.Interface for []api.Event based on the Timestamp field
type SortableEvents []v1.Event

func (list SortableEvents) Len() int {
	return len(list)
}

func (list SortableEvents) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list SortableEvents) Less(i, j int) bool {
	return list[i].LastTimestamp.Time.Before(list[j].LastTimestamp.Time)
}

type ImagePullTimeInfo struct {
	podStartTime time.Time
	podName      string
	nodeName     string
	imageName    string
	beginTime    time.Time
	endTime      time.Time
	totalTimeSec float64
}

var imagesMaps = map[string][]string{
	//"100M": {"100.125.0.198:20202/image-pull-test/myimage_20g:latest", "100.125.0.198:20202/image-pull-test/myimage_10g:latest"},
	"1G":   {"100.125.0.198:20202/image-pull-test/myimage_10_102g:latest", "100.125.0.198:20202/image-pull-test/myimage_51g:latest"},
	"5G":   {"100.125.0.198:20202/image-pull-test/myimage_10_512g:latest", "100.125.0.198:20202/image-pull-test/myimage_256g:latest"},
	"10G":  {"100.125.0.198:20202/image-pull-test/myimage_10_1024g:latest", "100.125.0.198:20202/image-pull-test/myimage_512g:latest"},
}

var imagePullSecret string

type ImagePullTestCase struct {
	// 测试场景名字
	Name string
	// 测试场景镜像规格，如1GB
	ImageSize string
	// 测试场景并发数，如100
	Concurrent int
	// 是否是单节点场景
	IsSingleNode bool
}

// key 测试场景名字， 如`单节点/100MB/10并发`
var testCases = map[string]ImagePullTestCase{
	//"single100m10test": {Name: "单节点/100M/10并发", ImageSize: "100M", Concurrent: 10, IsSingleNode: true,},
	//"single100m20test": {Name: "单节点/100M/20并发", ImageSize: "100M", Concurrent: 20, IsSingleNode: true,},
	//"single100m50test": {Name: "单节点/100M/50并发", ImageSize: "100M", Concurrent: 50, IsSingleNode: true,},
	//"single1g10test":   {Name: "单节点/1G/10并发", ImageSize: "1G", Concurrent: 10, IsSingleNode: true,},
	//"single1g20test":   {Name: "单节点/1G/20并发", ImageSize: "1G", Concurrent: 20, IsSingleNode: true,},
	//"single1g50test":   {Name: "单节点/1G/50并发", ImageSize: "1G", Concurrent: 50, IsSingleNode: true,},
	//"single5g10test":   {Name: "单节点/5G/10并发", ImageSize: "5G", Concurrent: 10, IsSingleNode: true,},
	//"single5g20test":   {Name: "单节点/5G/20并发", ImageSize: "5G", Concurrent: 20, IsSingleNode: true,},
	//"single5g50test":   {Name: "单节点/5G/50并发", ImageSize: "5G", Concurrent: 50, IsSingleNode: true,},
	//"single10g10test":  {Name: "单节点/10G/10并发", ImageSize: "5G", Concurrent: 10, IsSingleNode: true,},
	//"single10g20test":  {Name: "单节点/10G/20并发", ImageSize: "5G", Concurrent: 20, IsSingleNode: true,},
	//"single10g50test":  {Name: "单节点/10G/50并发", ImageSize: "5G", Concurrent: 50, IsSingleNode: true,},

	//"multi100m100test": {Name: "多节点/100M/100并发", ImageSize: "100M", Concurrent: 100, IsSingleNode: false,},
	//"multi100m200test": {Name: "多节点/100M/200并", ImageSize: "100M", Concurrent: 200, IsSingleNode: false,},
	//"multi100m500test": {Name: "多节点/100M/500并发", ImageSize: "100M", Concurrent: 500, IsSingleNode: false,},
	"multi1g100test":   {Name: "多节点/1G/100并发", ImageSize: "1G", Concurrent: 100, IsSingleNode: false,},
	"multi1g200test":   {Name: "多节点/1G/200并发", ImageSize: "1G", Concurrent: 200, IsSingleNode: false,},
	"multi1g500test":   {Name: "多节点/1G/500并发", ImageSize: "1G", Concurrent: 500, IsSingleNode: false,},
	"multi5g100test":   {Name: "多节点/5G/100并发", ImageSize: "5G", Concurrent: 100, IsSingleNode: false,},
	"multi5g200test":   {Name: "多节点/5G/200并发", ImageSize: "5G", Concurrent: 200, IsSingleNode: false,},
	"multi5g500test":   {Name: "多节点/5G/500并发", ImageSize: "5G", Concurrent: 500, IsSingleNode: false,},
	"multi10g100test":  {Name: "多节点/10G/100并发", ImageSize: "10G", Concurrent: 100, IsSingleNode: false,},
	"multi10g200test":  {Name: "多节点/10G/200并", ImageSize: "10G", Concurrent: 200, IsSingleNode: false,},
	"multi10g500test":  {Name: "多节点/10G/500并发", ImageSize: "10G", Concurrent: 500, IsSingleNode: false,},
}

func main() {
	var nodeName, namespace string

	var testCommand = &cobra.Command{
		Use:   "run",
		Short: "run.",
		Run: func(cmd *cobra.Command, args []string) {
			// print parameter
			fmt.Println(fmt.Sprintf("namespace: %v", namespace))

			if len(namespace) == 0 {
				fmt.Println("namespace is empty!")
				os.Exit(0)
			}

			// verify client is correct.
			_, err := conn.listPod(metav1.ListOptions{}, namespace)
			if err != nil {
				fmt.Println(fmt.Sprintf("list pod test err: %v", err))
				os.Exit(0)
			}

			TestImagePullCases(namespace, nodeName)
		},
	}

	testCommand.PersistentFlags().StringVar(&namespace, "ns", "", "namespace of pod")
	testCommand.PersistentFlags().StringVar(&nodeName, "node", "", "single node")
	testCommand.PersistentFlags().StringVar(&imagePullSecret, "image-pull-secret", "imagepull-secret", "image pull secret")

	testCommand.Execute()
}

func TestImagePullCases(ns, nodeName string) {
	for k, testCase := range testCases {
		labelForDp := map[string]string{"testpullimagecase": k}
		fmt.Println(fmt.Sprintf("开始测试 %s", testCase.Name))
		if testCase.IsSingleNode {
			TestPullImageFunc(k, ns, testCase.ImageSize, nodeName, testCase.Concurrent, nil, labelForDp)
		} else {
			TestPullImageFunc(k, ns, testCase.ImageSize, "", testCase.Concurrent, nil, labelForDp)
		}
		fmt.Println(fmt.Sprintf("结束测试 %s", testCase.Name))
	}
}

// namespace 测试使用命令空间
// imageSize 镜像规格，如100M，1G等
// nodeName 指定运行节点，当测试单节点时务必设置
// concurrent 此测试场景并发数量
// nodeSelector 节点selector，限制测试的节点范围
// labels 创建job和pod的label，用来筛选哪些是测试job和pod
func TestPullImageFunc(generateName, ns, imageSize, nodeName string, concurrent int, nodeSelector, podLabels map[string]string) {
	deploymentArray := []*apps.Deployment{}
	for k, imageArray := range imagesMaps {
		if k == imageSize {
			totalDp := concurrent / 50
			for i := 0; i < totalDp; i++ {
				labVal := podLabels["testpullimagecase"]
				podLabels[fmt.Sprintf("testpullimagecase%ddp", i)] = fmt.Sprintf("%s%ddp", labVal, i)
				fmt.Println(fmt.Sprintf("使用镜像: %s", imageArray[i%2]))
				dp := NewDeployment(generateName, 50, nodeSelector, podLabels, ns, nodeName, imageArray[i%2])
				apiDp, err := conn.createDeployment(dp)
				delete(podLabels, fmt.Sprintf("testpullimagecase%ddp", i))
				if err != nil {
					fmt.Println(fmt.Sprintf("create dp %s failed: %v", apiDp.Name, err))
					continue
				}
				deploymentArray = append(deploymentArray, apiDp)
			}
		}
	}

	// 等待所有的deployment ready
	for _, dp := range deploymentArray {
		err := waitDeploymentReady(dp)
		if err != nil {
			fmt.Println(fmt.Sprintf("waitDeploymentReady error: %v", err))
			continue
		}
	}

	fmt.Println(fmt.Sprintf("query pod labels for %v: %v", generateName, podLabels))
	selector := labels.Set(podLabels).AsSelector()
	options := metav1.ListOptions{LabelSelector: selector.String()}

	podList, err := conn.listPod(options, ns)
	if err != nil {
		fmt.Println(fmt.Errorf("get pod list err: %v", err))
	}

	fmt.Println(fmt.Sprintf("length of pods: %d", len(podList.Items)))

	podMap := map[string]*ImagePullTimeInfo{}

	for i, pod := range podList.Items {
		fmt.Println(fmt.Sprintf("begin process pod %d: %s/%s", i, pod.Namespace, pod.Name))
		imagePullInfo, err := statSinglePodImagePullDuration(ns, &pod)
		if err != nil || imagePullInfo == nil {
			fmt.Println(fmt.Errorf("stat pod image pull time err: %v", err))
			continue
		}

		key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		podMap[key] = imagePullInfo
	}

	// 将结果写入文件
	writeFile(fmt.Sprintf("%s.csv", generateName), podMap)

	for _, dp := range deploymentArray {
		err := DeleteDeployment(dp)
		if err != nil {
			fmt.Println(fmt.Sprintf("DeleteDeployment error: %v", err))
		}
		continue
	}
}

func statSinglePodImagePullDuration(ns string, pod *v1.Pod) (*ImagePullTimeInfo, error) {
	events, err := conn.SearchEvent(scheme.Scheme, pod, pod.Namespace)
	if err != nil {
		return nil, err
	}
	if events == nil {
		return nil, fmt.Errorf("event is nil for %s", pod.Name)
	}

	sort.Sort(SortableEvents(events.Items))
	var startPullingTime metav1.Time
	var endPulledTime metav1.Time

	for _, el := range events.Items {
		if el.Reason == PULLING {
			startPullingTime = el.FirstTimestamp
			fmt.Println(fmt.Sprintf("startPullingTime event %v", el.Name))
		}

		if el.Reason == PULLED {
			fmt.Println(fmt.Sprintf("endPulledTime event %v", el.Name))
			endPulledTime = el.FirstTimestamp
		}
	}

	info := ImagePullTimeInfo{
		podName:      fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
		nodeName:     pod.Spec.NodeName,
		imageName:    pod.Spec.Containers[0].Image,
		podStartTime: pod.Status.StartTime.Time,
		beginTime:    startPullingTime.Time,
		endTime:      endPulledTime.Time,
		totalTimeSec: endPulledTime.Sub(startPullingTime.Time).Seconds(),
	}

	return &info, nil
}

func writeFile(fileName string, podMap map[string]*ImagePullTimeInfo) {
	f, err := os.Create(fileName)
	if err != nil {
		fmt.Println("os Create error: ", err)
		return
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	bw.WriteString("podName" + "," + "nodeName" + "," + "imageName" + "," + "podStartTime" + "," + "pullingTime" + "," + "pulledTime" + "," + "pullImageTotalTime" + "\n")
	for _, v := range podMap {
		begin := v.beginTime.Format("2006-01-02 15:04:05")
		end := v.endTime.Format("2006-01-02 15:04:05")
		podStartTime := v.podStartTime.Format("2006-01-02 15:04:05")
		bw.WriteString(v.podName + "," + v.nodeName + "," + v.imageName + "," + podStartTime + "," + begin + "," + end + "," + fmt.Sprint(v.totalTimeSec) + "\n")
	}
	bw.Flush()
}

func DeleteDeployment(d *apps.Deployment) error {
	err := conn.deleteDeployment(d)
	if err != nil {
		return err
	}

	selector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return err
	}
	options := metav1.ListOptions{LabelSelector: selector.String()}

	// Ensuring deployment Pods were deleted
	var pods *v1.PodList
	if err := wait.PollImmediate(time.Second, 5*time.Minute, func() (bool, error) {
		pods, err = conn.listPod(options, d.Namespace)
		if err != nil {
			return false, err
		}

		if len(pods.Items) == 0 {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("Err : %s\n. Failed to remove deployment %s pods : %+v", err, d.Name, pods)
	}

	return nil
}

func waitDeploymentReady(d *apps.Deployment) error {
	var deployment *apps.Deployment
	err := wait.PollImmediate(10*time.Second, 40*time.Minute, func() (bool, error) {
		var err error
		deployment, err = conn.getDeployment(d)
		if err != nil {
			return false, err
		}

		// When the deployment status and its underlying resources reach the desired state, we're done
		if DeploymentComplete(d, &deployment.Status) {
			return true, nil
		}

		return false, nil
	})

	if err != nil {
		return fmt.Errorf("error waiting for deployment %q status to match expectation: %v", d.Name, err)
	}

	return nil
}

func DeploymentComplete(deployment *apps.Deployment, newStatus *apps.DeploymentStatus) bool {
	fmt.Println(fmt.Sprintf("Number of ready pod %v for dp %s", newStatus.ReadyReplicas, fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)))
	return newStatus.ReadyReplicas == *(deployment.Spec.Replicas)
}

func NewDeployment(deploymentName string, replicas int32, nodeSelector, podLabels map[string]string, ns, nodeName, image string) *apps.Deployment {
	zero := int64(0)
	return &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: deploymentName,
			Namespace:    ns,
			Labels:       podLabels,
		},
		Spec: apps.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: v1.PodSpec{
					NodeName:                      nodeName,
					NodeSelector:                  nodeSelector,
					TerminationGracePeriodSeconds: &zero,
					ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: imagePullSecret,
						},
					},
					Containers: []v1.Container{
						{
							Name:    deploymentName,
							Image:   image,
							Command: []string{"sh", "-c", "echo test && sleep 3600"},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("250m"),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("250m"),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
		},
	}
}
