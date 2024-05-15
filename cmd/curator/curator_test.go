// Copyright Contributors to the Open Cluster Management project.

package main

import (
	"os"
	"strings"
	"testing"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	clustercuratorv1 "github.com/stolostron/cluster-curator-controller/pkg/api/v1beta1"
	"github.com/stolostron/cluster-curator-controller/pkg/jobs/utils"
	"github.com/stolostron/library-go/pkg/config"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const ClusterName = "my-cluster"
const ClusterNamespace = "clusters"
const NodepoolName = "my-cluster-us-east-2"

var s = scheme.Scheme

func getClusterCurator() *clustercuratorv1.ClusterCurator {
	return &clustercuratorv1.ClusterCurator{
		ObjectMeta: v1.ObjectMeta{
			Name:      ClusterName,
			Namespace: ClusterName,
		},
		Spec: clustercuratorv1.ClusterCuratorSpec{
			DesiredCuration: "install",
		},
	}
}

func getClusterCuratorWithInstallOperation() *clustercuratorv1.ClusterCurator {
	return &clustercuratorv1.ClusterCurator{
		ObjectMeta: v1.ObjectMeta{
			Name:      ClusterName,
			Namespace: ClusterName,
		},
		Operation: &clustercuratorv1.Operation{
			RetryPosthook: "installPosthook",
		},
		Spec: clustercuratorv1.ClusterCuratorSpec{
			DesiredCuration: "install",
		},
	}
}

func getClusterCuratorWithUpgradeOperation() *clustercuratorv1.ClusterCurator {
	return &clustercuratorv1.ClusterCurator{
		ObjectMeta: v1.ObjectMeta{
			Name:      ClusterName,
			Namespace: ClusterName,
		},
		Operation: &clustercuratorv1.Operation{
			RetryPosthook: "upgradePosthook",
		},
		Spec: clustercuratorv1.ClusterCuratorSpec{
			DesiredCuration: "upgrade",
		},
	}
}

func getHypershiftClusterCurator() *clustercuratorv1.ClusterCurator {
	return &clustercuratorv1.ClusterCurator{
		ObjectMeta: v1.ObjectMeta{
			Name:      ClusterName,
			Namespace: ClusterNamespace,
		},
		Spec: clustercuratorv1.ClusterCuratorSpec{
			DesiredCuration: "install",
		},
	}
}

func getHostedCluster(hcType string, hcConditions []interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "hypershift.openshift.io/v1beta1",
			"kind":       "HostedCluster",
			"metadata": map[string]interface{}{
				"name":      ClusterName,
				"namespace": ClusterNamespace,
				"labels": map[string]interface{}{
					"hypershift.openshift.io/auto-created-for-infra": ClusterName + "-xyz",
				},
			},
			"spec": map[string]interface{}{
				"pausedUntil": "true",
				"platform": map[string]interface{}{
					"type": hcType,
				},
				"release": map[string]interface{}{
					"image": "quay.io/openshift-release-dev/ocp-release:4.13.6-multi",
				},
			},
			"status": map[string]interface{}{
				"conditions": hcConditions,
			},
		},
	}
}

func getNodepool(npName string, npNamespace string, npClusterName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "hypershift.openshift.io/v1beta1",
			"kind":       "NodePool",
			"metadata": map[string]interface{}{
				"name":      npName,
				"namespace": npNamespace,
			},
			"spec": map[string]interface{}{
				"pausedUntil": "true",
				"clusterName": npClusterName,
				"release": map[string]interface{}{
					"image": "quay.io/openshift-release-dev/ocp-release:4.13.6-multi",
				},
			},
		},
	}
}

func TestCuratorRunNoParam(t *testing.T) {

	defer func() {
		r := recover()
		t.Log(r.(error).Error())

		if !strings.Contains(r.(error).Error(), "Command: ./curator [") &&
			!strings.Contains(r.(error).Error(), "Invalid Parameter: \"\"") {
			t.Fatal(r)
		}
		t.Log("Detected missing paramter")
	}()

	os.Args[1] = ""

	curatorRun(nil, nil, ClusterName, ClusterName)
}

func TestCuratorRunWrongParam(t *testing.T) {

	defer func() {
		r := recover()
		t.Log(r.(error).Error())

		if !strings.Contains(r.(error).Error(), "Command: ./curator [") &&
			!strings.Contains(r.(error).Error(), "something-wrong") {
			t.Fatal(r)
		}
		t.Log("Detected wrong paramter")
	}()

	os.Args[1] = "something-wrong"

	curatorRun(nil, nil, ClusterName, ClusterName)
}

func TestCuratorRunNoClusterCurator(t *testing.T) {

	defer func() {
		r := recover()
		t.Log(r.(error).Error())

		if !strings.Contains(r.(error).Error(), "clustercurators.cluster.open-cluster-management.io \"my-cluster\"") {
			t.Fatal(r)
		}
		t.Log("Detected missing ClusterCurator resource")
	}()

	s := scheme.Scheme
	s.AddKnownTypes(utils.CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})

	client := clientfake.NewFakeClientWithScheme(s)

	os.Args[1] = "SKIP_ALL_TESTING"

	curatorRun(nil, client, ClusterName, ClusterName)
}

func TestCuratorRunClusterCurator(t *testing.T) {

	s := scheme.Scheme
	s.AddKnownTypes(utils.CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})

	client := clientfake.NewFakeClientWithScheme(s, getClusterCurator())

	os.Args[1] = "SKIP_ALL_TESTING"

	assert.NotPanics(t, func() { curatorRun(nil, client, ClusterName, ClusterName) }, "no panic when ClusterCurator found and skip test")
}

func TestCuratorRunClusterCuratorInstallUpgradeOperation(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(utils.CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})
	client := clientfake.NewFakeClientWithScheme(s, getClusterCuratorWithInstallOperation())

	os.Args[1] = "SKIP_ALL_TESTING"

	assert.NotPanics(t, func() { curatorRun(nil, client, ClusterName, ClusterName) }, "no panic when ClusterCurator found and skip test")

	client = clientfake.NewFakeClientWithScheme(s, getClusterCuratorWithUpgradeOperation())

	assert.NotPanics(t, func() { curatorRun(nil, client, ClusterName, ClusterName) }, "no panic when ClusterCurator found and skip test")
}

func TestCuratorRunNoProviderCredentialPath(t *testing.T) {

	defer func() {
		r := recover()
		t.Log(r.(error).Error())

		if !strings.Contains(r.(error).Error(), "Missing spec.providerCredentialPath") {
			t.Fatal(r)
		}
		t.Log("Detected missing provierCredentialPath")
	}()

	s := scheme.Scheme
	hivev1.AddToScheme(s)
	s.AddKnownTypes(utils.CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})

	client := clientfake.NewClientBuilder().WithRuntimeObjects(getClusterCurator()).WithScheme(s).Build()

	os.Args[1] = "applycloudprovider-ansible"

	curatorRun(nil, client, ClusterName, ClusterName)
}

func TestCuratorRunProviderCredentialPathEnv(t *testing.T) {

	defer func() {
		r := recover()
		t.Log(r.(error).Error())

		if !strings.Contains(r.(error).Error(), "secrets \"secretname\"") {
			t.Fatal(r)
		}
		t.Log("Detected missing namespace/secretName")
	}()

	os.Setenv("PROVIDER_CREDENTIAL_PATH", "namespace/secretname")
	client := clientfake.NewClientBuilder().Build()

	os.Args[1] = "applycloudprovider-ansible"

	curatorRun(nil, client, ClusterName, ClusterName)
}

func TestInvokeMonitor(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected recover, but failed")
		}
	}()

	os.Setenv("PROVIDER_CREDENTIAL_PATH", "namespace/secretname")
	os.Args[1] = "monitor"

	curatorRun(nil, clientfake.NewClientBuilder().Build(), ClusterName, ClusterName)
}

func TestInvokeMonitorImport(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected recover, but failed")
		}
	}()

	os.Setenv("PROVIDER_CREDENTIAL_PATH", "namespace/secretname")
	os.Args[1] = "monitor-import"

	curatorRun(nil, clientfake.NewClientBuilder().Build(), ClusterName, ClusterName)
}

func TestInvokeMonitorDestroy(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected recover, but failed")
		}
	}()

	os.Setenv("PROVIDER_CREDENTIAL_PATH", "namespace/secretname")
	os.Args[1] = "monitor-destroy"

	curatorRun(nil, clientfake.NewClientBuilder().Build(), ClusterName, ClusterName)
}

func TestUpgradFailed(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected recover, but failed")
		}
	}()

	os.Args[1] = "upgrade-cluster"

	s := scheme.Scheme
	s.AddKnownTypes(utils.CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})

	client := clientfake.NewFakeClientWithScheme(s, &clustercuratorv1.ClusterCurator{
		ObjectMeta: v1.ObjectMeta{
			Name:      ClusterName,
			Namespace: ClusterName,
		},
		Spec: clustercuratorv1.ClusterCuratorSpec{
			DesiredCuration: "upgrade",
			Upgrade: clustercuratorv1.UpgradeHooks{
				DesiredUpdate: "4.11.4",
			},
		},
	})

	curatorRun(nil, client, ClusterName, ClusterName)
}

func TestUpgradDone(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected error %v", r)
		}
	}()

	os.Args[1] = "done"

	s := scheme.Scheme
	s.AddKnownTypes(utils.CCGVR.GroupVersion(), &clustercuratorv1.ClusterCurator{})

	client := clientfake.NewFakeClientWithScheme(s, &clustercuratorv1.ClusterCurator{
		ObjectMeta: v1.ObjectMeta{
			Name:      ClusterName,
			Namespace: ClusterName,
		},
		Spec: clustercuratorv1.ClusterCuratorSpec{
			DesiredCuration: "upgrade",
			Upgrade: clustercuratorv1.UpgradeHooks{
				DesiredUpdate: "4.11.4",
			},
		},
	})

	curatorRun(nil, client, ClusterName, ClusterName)
}

func TestHypershiftActivate(t *testing.T) {
	// Test will fail because we can't pass in a fake dynamic client
	// But that's ok, we just need to test the curator code
	defer func() { // recover from not having a hub config object
		if r := recover(); r == nil {
			t.Fatal("expected recover, but failed")
		}
	}()

	os.Args[1] = "activate-and-monitor"
	os.Args[2] = ClusterName

	s.AddKnownTypes(clustercuratorv1.SchemeBuilder.GroupVersion, &clustercuratorv1.ClusterCurator{})
	client := clientfake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(
		getHypershiftClusterCurator(),
		getHostedCluster("AWS", []interface{}{}),
		getNodepool(NodepoolName, ClusterNamespace, ClusterName),
	).Build()

	config, _ := config.LoadConfig("", "", "")

	curatorRun(config, client, ClusterNamespace, ClusterName)
}

func TestHypershiftMonitor(t *testing.T) {
	// Test will fail because we can't pass in a fake dynamic client
	// But that's ok, we just need to test the curator code
	defer func() { // recover from not having a hub config object
		if r := recover(); r == nil {
			t.Fatal("expected recover, but failed")
		}
	}()

	os.Args[1] = "monitor"
	os.Args[2] = ClusterName

	s.AddKnownTypes(clustercuratorv1.SchemeBuilder.GroupVersion, &clustercuratorv1.ClusterCurator{})
	client := clientfake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(
		getHypershiftClusterCurator(),
		getHostedCluster("AWS", []interface{}{}),
		getNodepool(NodepoolName, ClusterNamespace, ClusterName),
	).Build()

	config, _ := config.LoadConfig("", "", "")

	curatorRun(config, client, ClusterNamespace, ClusterName)
}

func TestHypershiftDestroyCluster(t *testing.T) {
	// Test will fail because we can't pass in a fake dynamic client
	// But that's ok, we just need to test the curator code
	defer func() { // recover from not having a hub config object
		if r := recover(); r == nil {
			t.Fatal("expected recover, but failed")
		}
	}()

	os.Args[1] = "destroy-cluster"
	os.Args[2] = ClusterName

	s.AddKnownTypes(clustercuratorv1.SchemeBuilder.GroupVersion, &clustercuratorv1.ClusterCurator{})
	client := clientfake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(
		getHypershiftClusterCurator(),
		getHostedCluster("KubeVirt", []interface{}{}),
		getNodepool(NodepoolName, ClusterNamespace, ClusterName),
	).Build()

	config, _ := config.LoadConfig("", "", "")

	curatorRun(config, client, ClusterNamespace, ClusterName)
}

func TestHypershiftMonitorDestroy(t *testing.T) {
	// Test will fail because we can't pass in a fake dynamic client
	// But that's ok, we just need to test the curator code
	defer func() { // recover from not having a hub config object
		if r := recover(); r == nil {
			t.Fatal("expected recover, but failed")
		}
	}()

	os.Args[1] = "monitor-destroy"
	os.Args[2] = ClusterName

	s.AddKnownTypes(clustercuratorv1.SchemeBuilder.GroupVersion, &clustercuratorv1.ClusterCurator{})
	client := clientfake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(
		getHypershiftClusterCurator(),
		getHostedCluster("KubeVirt", []interface{}{}),
		getNodepool(NodepoolName, ClusterNamespace, ClusterName),
	).Build()

	config, _ := config.LoadConfig("", "", "")

	curatorRun(config, client, ClusterNamespace, ClusterName)
}

func TestHypershiftUpgradeCluster(t *testing.T) {
	// Test will fail because we can't pass in a fake dynamic client
	// But that's ok, we just need to test the curator code
	defer func() { // recover from not having a hub config object
		if r := recover(); r == nil {
			t.Fatal("expected recover, but failed")
		}
	}()

	os.Args[1] = "upgrade-cluster"
	os.Args[2] = ClusterName

	s.AddKnownTypes(clustercuratorv1.SchemeBuilder.GroupVersion, &clustercuratorv1.ClusterCurator{})
	client := clientfake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(
		getHypershiftClusterCurator(),
		getHostedCluster("AWS", []interface{}{}),
		getNodepool(NodepoolName, ClusterNamespace, ClusterName),
	).Build()

	config, _ := config.LoadConfig("", "", "")

	curatorRun(config, client, ClusterNamespace, ClusterName)
}

func TestHypershiftMonitorUpgrade(t *testing.T) {
	// Test will fail because we can't pass in a fake dynamic client
	// But that's ok, we just need to test the curator code
	defer func() { // recover from not having a hub config object
		if r := recover(); r == nil {
			t.Fatal("expected recover, but failed")
		}
	}()

	os.Args[1] = "monitor-upgrade"
	os.Args[2] = ClusterName

	s.AddKnownTypes(clustercuratorv1.SchemeBuilder.GroupVersion, &clustercuratorv1.ClusterCurator{})
	client := clientfake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(
		getHypershiftClusterCurator(),
		getHostedCluster("AWS", []interface{}{}),
		getNodepool(NodepoolName, ClusterNamespace, ClusterName),
	).Build()

	config, _ := config.LoadConfig("", "", "")

	curatorRun(config, client, ClusterNamespace, ClusterName)
}
