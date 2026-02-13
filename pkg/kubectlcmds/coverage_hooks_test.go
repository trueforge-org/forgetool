package kubectlcmds

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	origApproveUpdateApprovalFn = approveUpdateApprovalFn
	origApproveListCSRsFn       = approveListCSRsFn
	origApproveCSRLoopFn        = approveCSRLoopFn
	origCheckStatusListPodsFn   = checkStatusListPodsFn
	origKustomizeAsYAMLFn       = kustomizeAsYAMLFn
	origYAMLReadNodesFn         = yamlReadNodesFn
	origYAMLPatchObjectFn       = yamlPatchObjectFn
)

func resetKubectlHooks(t *testing.T) {
	t.Helper()

	applyStatFn = os.Stat
	applyReadFileFn = os.ReadFile
	applyGetKubeClientFn = getKubeClient
	applyYAMLFn = applyYAML
	applyBuildKustomizeYAMLFn = buildKustomizeYAML

	approveClientConfigFn = func() (*rest.Config, error) {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	}
	approveInClusterConfigFn = rest.InClusterConfig
	approveNewForConfigFn = kubernetes.NewForConfig
	approveUpdateApprovalFn = func(clientset *kubernetes.Clientset, name string, csrCopy *certificatesv1.CertificateSigningRequest) (*certificatesv1.CertificateSigningRequest, error) {
		return clientset.CertificatesV1().CertificateSigningRequests().UpdateApproval(context.TODO(), name, csrCopy, metav1.UpdateOptions{})
	}

	approvePendingCSRsOnceFn = approvePendingCSRsOnce
	approvePendingSleepFn = time.Sleep
	approveListCSRsFn = func(clientset *kubernetes.Clientset) (int, []interface{}, error) {
		csrList, err := clientset.CertificatesV1().CertificateSigningRequests().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return 0, nil, err
		}
		items := make([]interface{}, 0, len(csrList.Items))
		for i := range csrList.Items {
			items = append(items, csrList.Items[i])
		}
		return len(csrList.Items), items, nil
	}
	approveCSRLoopFn = func(clientset *kubernetes.Clientset, csr interface{}) error {
		typed := csr.(certificatesv1.CertificateSigningRequest)
		return approveCSR(clientset, typed)
	}

	checkStatusNewClientsetFn = newKubernetesClientset
	checkStatusNowFn = time.Now
	checkStatusSleepFn = time.Sleep
	checkStatusListPodsFn = func(clientset *kubernetes.Clientset) ([]corev1.Pod, error) {
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		return pods.Items, nil
	}
	checkStatusAllRunningFn = allRequiredPodsRunning
	checkstatusBuildConfigFn = clientcmd.BuildConfigFromFlags
	checkstatusNewForConfigFn = kubernetes.NewForConfig

	clientHomeDirFn = homedir.HomeDir
	clientBuildConfigFn = clientcmd.BuildConfigFromFlags
	clientNewRuntimeClientFn = client.New

	kustomizeStatFn = os.Stat
	kustomizeRunFn = runKustomize
	kustomizeAsYAMLFn = func(r resmap.ResMap) ([]byte, error) { return r.AsYaml() }

	yamlReadNodesFn = func(yamlData []byte) ([]*kyaml.RNode, error) {
		reader := kio.ByteReader{Reader: bytes.NewReader(yamlData)}
		return reader.Read()
	}
	yamlNodeToUnstructuredFn = nodeToUnstructured
	yamlApplyObjectWithRetryFn = applyObjectWithRetry
	yamlPatchObjectFn = func(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) error {
		return k8sClient.Patch(ctx, obj, client.Apply, client.FieldOwner("kustomize-controller"))
	}
	yamlApplySleepFn = time.Sleep
	yamlNormalizeContextFn = normalizeContext
}
