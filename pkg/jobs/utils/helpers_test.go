// Copyright Contributors to the Open Cluster Management project.
package utils

import (
	"context"
	"errors"
	"testing"

	clustercuratorv1 "github.com/open-cluster-management/cluster-curator-controller/pkg/api/v1beta1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckErrorNil(t *testing.T) {

	InitKlog(4)
	assert.NotPanics(t, func() { CheckError(nil) }, "No panic, when err is not present")
}

func TestCheckErrorNotNil(t *testing.T) {

	assert.Panics(t, func() { CheckError(errors.New("TeST")) }, "Panics when a err is received")
}

func TestLogErrorNil(t *testing.T) {

	assert.Nil(t, LogError(nil), "err nil, when no err message")
}

func TestLogErrorNotNil(t *testing.T) {

	assert.NotNil(t, LogError(errors.New("TeST")), "err nil, when no err message")
}

// TODO, replace all instances of klog.Warning that include an IF, this saves us 2x lines of code
func TestLogWarning(t *testing.T) {

	assert.NotPanics(t, func() { LogWarning(nil) }, "No panic, when logging warnings")
	assert.NotPanics(t, func() { LogWarning(errors.New("TeST")) }, "No panic, when logging warnings")
}

func TestPathSplitterFromEnv(t *testing.T) {

	_, _, err := PathSplitterFromEnv("")
	assert.NotNil(t, err, "err not nil, when empty path")

	_, _, err = PathSplitterFromEnv("value")
	assert.NotNil(t, err, "err not nil, when only one value")

	_, _, err = PathSplitterFromEnv("value/")
	assert.NotNil(t, err, "err not nil, when only one value with split present")

	_, _, err = PathSplitterFromEnv("/value")
	assert.NotNil(t, err, "err not nil, when only one value with split present")

	namespace, secretName, err := PathSplitterFromEnv("ns1/s1")

	assert.Nil(t, err, "err nil, when path is split successfully")
	assert.Equal(t, namespace, "ns1", "namespace should be ns1")
	assert.Equal(t, secretName, "s1", "secret name should be s1")

}

const ClusterName = "my-cluster"
const PREHOOK = "prehook"
const jobName = "my-jobname-12345"

func getClusterCurator() *clustercuratorv1.ClusterCurator {
	return &clustercuratorv1.ClusterCurator{
		ObjectMeta: v1.ObjectMeta{
			Name:      ClusterName,
			Namespace: ClusterName,
		},
		Spec: clustercuratorv1.ClusterCuratorSpec{},
	}
}

func TestRecordCurrentCuratorJob(t *testing.T) {

	cc := getClusterCurator()

	s := scheme.Scheme
	s.AddKnownTypes(CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})

	dynclient := dynfake.NewSimpleDynamicClient(s, cc)

	assert.NotEqual(t, jobName, cc.Spec.CuratingJob, "Nost equal because curating job name not written yet")
	assert.NotPanics(t, func() {
		patchDyn(dynclient, ClusterName, jobName, CurrentCuratorJob)
	}, "no panics, when update successful")

	ccMap, err := dynclient.Resource(CCGVR).Namespace(ClusterName).Get(context.TODO(), ClusterName, v1.GetOptions{})
	assert.Nil(t, err, "err is nil, when my-cluster clusterCurator resource is retrieved")

	assert.Equal(t,
		jobName, ccMap.Object["spec"].(map[string]interface{})[CurrentCuratorJob],
		"Equal when curator job recorded")
}

func TestRecordCurrentCuratorJobError(t *testing.T) {

	s := scheme.Scheme

	dynclient := dynfake.NewSimpleDynamicClient(s)

	assert.NotPanics(t, func() {

		err := patchDyn(dynclient, ClusterName, jobName, CurrentCuratorJob)
		assert.NotNil(t, err, "err is not nil, when patch fails")

	}, "no panics, when update successful")

}

func TestRecordCuratorJob(t *testing.T) {

	err := RecordCuratorJob(ClusterName, jobName)
	assert.NotNil(t, err, "err is not nil, when failure occurs")
	t.Logf("err:\n%v", err)
}

func TestGetDynset(t *testing.T) {
	_, err := GetDynset(nil)
	assert.Nil(t, err, "err is nil, when dynset is initialized")
}

func TestGetClient(t *testing.T) {
	_, err := GetClient()
	assert.Nil(t, err, "err is nil, when client is initialized")
}

func TestGetKubeset(t *testing.T) {
	_, err := GetKubeset()
	assert.Nil(t, err, "err is nil, when kubset is initialized")
}

func TestRecordCuratedStatusCondition(t *testing.T) {

	cc := getClusterCurator()

	s := scheme.Scheme
	s.AddKnownTypes(CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})

	client := clientfake.NewFakeClientWithScheme(s, cc)

	assert.Nil(t,
		recordCuratedStatusCondition(
			client,
			ClusterName,
			CurrentAnsibleJob,
			v1.ConditionTrue,
			JobHasFinished,
			"Almost finished"),
		"err is nil, when conditon successfully set")

	ccNew := &clustercuratorv1.ClusterCurator{}
	assert.Nil(t,
		client.Get(context.Background(), types.NamespacedName{Namespace: ClusterName, Name: ClusterName}, ccNew),
		"err is nil, when ClusterCurator resource is retreived")
	t.Log(ccNew)
	assert.Equal(t,
		ccNew.Status.Conditions[0].Type, CurrentAnsibleJob,
		"equal when status condition correctly recorded")

}

func TestRecordCurrentStatusConditionNoResource(t *testing.T) {

	s := scheme.Scheme
	s.AddKnownTypes(CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})

	client := clientfake.NewFakeClientWithScheme(s)

	err := RecordCurrentStatusCondition(
		client,
		ClusterName,
		CurrentAnsibleJob,
		v1.ConditionTrue,
		"Almost finished")
	assert.NotNil(t, err, "err is not nil, when conditon can not be written")
	t.Logf("err: %v", err)
}

func TestGetClusterCurator(t *testing.T) {

	cc := getClusterCurator()

	s := scheme.Scheme
	s.AddKnownTypes(CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})

	client := clientfake.NewFakeClientWithScheme(s, cc)

	_, err := GetClusterCurator(client, ClusterName)
	assert.Nil(t, err, "err is nil, when ClusterCurator resource is retrieved")
}

func TestGetClusterCuratorNoResource(t *testing.T) {

	s := scheme.Scheme
	s.AddKnownTypes(CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})

	client := clientfake.NewFakeClientWithScheme(s)

	cc, err := GetClusterCurator(client, ClusterName)
	assert.Nil(t, cc, "cc is nil, when ClusterCurator resource is not found")
	assert.NotNil(t, err, "err is not nil, when ClusterCurator is not found")
	t.Logf("err: %v", err)
}

func TestRecordCuratorJobName(t *testing.T) {

	cc := getClusterCurator()

	s := scheme.Scheme
	s.AddKnownTypes(CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})
	client := clientfake.NewFakeClientWithScheme(s, cc)

	err := RecordCuratorJobName(client, ClusterName, "my-job-ABCDE")

	assert.Nil(t, err, "err nil, when Job name written to ClusterCurator.Spec.curatorJob")

}

func TestRecordCuratorJobNameInvalidCurator(t *testing.T) {

	s := scheme.Scheme
	s.AddKnownTypes(CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})
	client := clientfake.NewFakeClientWithScheme(s)

	err := RecordCuratorJobName(client, ClusterName, "my-job-ABCDE")

	assert.NotNil(t, err, "err nil, when Job name written to ClusterCurator.Spec.curatorJob")
	t.Logf("Detected Errror: %v", err.Error())

}
