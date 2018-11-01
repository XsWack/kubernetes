/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package storage

import (
	"fmt"
	"math/rand"
	"time"

	"k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/storage/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// snapshot CRD api group
const snapshotGroup = "snapshot.storage.k8s.io"

// snapshot CRD api version
const snapshotAPIVersion = "snapshot.storage.k8s.io/v1alpha1"

var (
	snapshotGVR        = schema.GroupVersionResource{Group: snapshotGroup, Version: "v1alpha1", Resource: "volumesnapshots"}
	snapshotClassGVR   = schema.GroupVersionResource{Group: snapshotGroup, Version: "v1alpha1", Resource: "volumesnapshotclasses"}
	snapshotContentGVR = schema.GroupVersionResource{Group: snapshotGroup, Version: "v1alpha1", Resource: "volumesnapshotcontents"}
)

type SnapshotClassTest struct {
	Name                 string
	CloudProviders       []string
	Snapshotter          string
	Parameters           map[string]string
	NodeName             string
	SnapshotContentCheck func(snapshotContent *unstructured.Unstructured) error
}

var _ = utils.SIGDescribe("CSI Snapshots", func() {
	f := framework.NewDefaultFramework("csi-snapshot-plugin")

	var (
		cs            clientset.Interface
		dynamicClient dynamic.Interface
		ns            *v1.Namespace
		node          v1.Node
		config        framework.VolumeTestConfig
	)

	BeforeEach(func() {
		cs = f.ClientSet
		dynamicClient = f.DynamicClient
		ns = f.Namespace
		nodes := framework.GetReadySchedulableNodesOrDie(f.ClientSet)
		node = nodes.Items[rand.Intn(len(nodes.Items))]
		config = framework.VolumeTestConfig{
			Namespace:         ns.Name,
			Prefix:            "csi",
			ClientNodeName:    node.Name,
			ServerNodeName:    node.Name,
			WaitForCompletion: true,
		}
	})

	Context(fmt.Sprintf("CSI plugin snapshot test using HostPath CSI driver [Serial]"), func() {
		var (
			driver csiTestDriver
		)
		BeforeEach(func() {
			driver = initCSIHostpath(f, config)
			driver.createCSIDriver()
		})

		AfterEach(func() {
			driver.cleanupCSIDriver()
		})

		It("should create snapshot", func() {
			storageClassTest := driver.createStorageClassTest()
			claim := newClaim(storageClassTest, ns.GetName(), "", nil)
			scName := storageClassTest.StorageClassName
			claim.Spec.StorageClassName = &scName

			By("creating a claim")
			claim, err := cs.CoreV1().PersistentVolumeClaims(claim.Namespace).Create(claim)
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				framework.Logf("deleting claim %q/%q", claim.Namespace, claim.Name)
				// typically this claim has already been deleted
				err = cs.CoreV1().PersistentVolumeClaims(claim.Namespace).Delete(claim.Name, nil)
				if err != nil && !apierrs.IsNotFound(err) {
					framework.Failf("Error deleting claim %q. Error: %v", claim.Name, err)
				}
			}()
			err = framework.WaitForPersistentVolumeClaimPhase(v1.ClaimBound, cs, claim.Namespace, claim.Name, framework.Poll, framework.ClaimProvisionTimeout)
			Expect(err).NotTo(HaveOccurred())

			By("checking the claim")
			// Get new copy of the claim
			claim, err = cs.CoreV1().PersistentVolumeClaims(claim.Namespace).Get(claim.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Get the bound PV
			pv, err := cs.CoreV1().PersistentVolumes().Get(claim.Spec.VolumeName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("creating a SnapshotClass")
			snapshotClassTest := driver.createSnapshotClassTest()
			snapshotClass := newSnapshotClass(snapshotClassTest, ns.GetName(), "scc")
			snapshotClass, err = dynamicClient.Resource(snapshotClassGVR).Create(snapshotClass, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				framework.Logf("deleting SnapshotClass %s", snapshotClass.GetName())
				framework.ExpectNoError(dynamicClient.Resource(snapshotClassGVR).Delete(snapshotClass.GetName(), nil))
			}()

			By("creating a snapshot")
			snapshot := newSnapshot(claim, ns.GetName(), snapshotClass.GetName())
			snapshot, err = dynamicClient.Resource(snapshotGVR).Namespace(ns.GetName()).Create(snapshot, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				framework.Logf("deleting snapshot %q/%q", snapshot.GetNamespace(), snapshot.GetName())
				// typically this snapshot has already been deleted
				err = dynamicClient.Resource(snapshotGVR).Namespace(ns.GetName()).Delete(snapshot.GetName(), nil)
				if err != nil && !apierrs.IsNotFound(err) {
					framework.Failf("Error deleting snapshot %q. Error: %v", claim.Name, err)
				}
			}()
			err = WaitForSnapshotReady(dynamicClient, snapshot.GetNamespace(), snapshot.GetName(), framework.Poll, framework.SnapshotCreateTimeout)
			Expect(err).NotTo(HaveOccurred())

			By("checking the snapshot")
			// Get new copy of the snapshot
			snapshot, err = dynamicClient.Resource(snapshotGVR).Namespace(snapshot.GetNamespace()).Get(snapshot.GetName(), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Get the bound snapshotContent
			snapshotSpec := snapshot.Object["spec"].(map[string]interface{})
			snapshotContentName := snapshotSpec["snapshotContentName"].(string)
			snapshotContent, err := dynamicClient.Resource(snapshotContentGVR).Get(snapshotContentName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			snapshotContentSpec := snapshotContent.Object["spec"].(map[string]interface{})
			volumeSnapshotRef := snapshotContentSpec["volumeSnapshotRef"].(map[string]interface{})
			persistentVolumeRef := snapshotContentSpec["persistentVolumeRef"].(map[string]interface{})

			// Check SnapshotContent properties
			By("checking the SnapshotContent")
			Expect(snapshotContentSpec["snapshotClassName"]).To(Equal(snapshotClass.GetName()))
			Expect(volumeSnapshotRef["name"]).To(Equal(snapshot.GetName()))
			Expect(volumeSnapshotRef["namespace"]).To(Equal(snapshot.GetNamespace()))
			Expect(persistentVolumeRef["name"]).To(Equal(pv.Name))

			// Run the checker
			if snapshotClassTest.SnapshotContentCheck != nil {
				err = snapshotClassTest.SnapshotContentCheck(snapshotContent)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})

// WaitForSnapshotReady waits for a VolumeSnapshot to be ready to use or until timeout occurs, whichever comes first.
func WaitForSnapshotReady(c dynamic.Interface, ns string, snapshotName string, Poll, timeout time.Duration) error {
	framework.Logf("Waiting up to %v for VolumeSnapshot %s to become ready", timeout, snapshotName)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(Poll) {
		snapshot, err := c.Resource(snapshotGVR).Namespace(ns).Get(snapshotName, metav1.GetOptions{})
		if err != nil {
			framework.Logf("Failed to get claim %q, retrying in %v. Error: %v", snapshotName, Poll, err)
			continue
		} else {
			status := snapshot.Object["status"]
			if status == nil {
				framework.Logf("VolumeSnapshot %s found but is not ready.", snapshotName)
				continue
			}
			value := status.(map[string]interface{})
			if value["ready"] == true {
				framework.Logf("VolumeSnapshot %s found and is ready", snapshotName, time.Since(start))
				return nil
			} else {
				framework.Logf("VolumeSnapshot %s found but is not ready.", snapshotName)
			}
		}
	}
	return fmt.Errorf("VolumeSnapshot %s is not ready within %v", snapshotName, timeout)
}

func newSnapshotClass(test SnapshotClassTest, ns string, suffix string) *unstructured.Unstructured {
	snapshotClass := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "VolumeSnapshotClass",
			"apiVersion": snapshotAPIVersion,
			"metadata": map[string]interface{}{
				"name": ns + "-" + suffix,
			},
			"snapshotter": test.Snapshotter,
			"parameters":  test.Parameters,
		},
	}

	return snapshotClass
}

func newSnapshot(claim *v1.PersistentVolumeClaim, ns, snapshotClassName string) *unstructured.Unstructured {
	snapshot := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "VolumeSnapshot",
			"apiVersion": snapshotAPIVersion,
			"metadata": map[string]interface{}{
				"generateName": "snapshot-",
				"namespace":    ns,
			},
			"spec": map[string]interface{}{
				"snapshotClassName": snapshotClassName,
				"source": map[string]interface{}{
					"name": claim.Name,
					"kind": "PersistentVolumeClaim",
				},
			},
		},
	}

	return snapshot
}
