package kubectlcmds

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestCoverage_ApproveCSR(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	csr := certificatesv1.CertificateSigningRequest{ObjectMeta: metav1.ObjectMeta{Name: "csr-1"}}

	approveUpdateApprovalFn = func(_ *kubernetes.Clientset, _ string, _ *certificatesv1.CertificateSigningRequest) (*certificatesv1.CertificateSigningRequest, error) {
		return nil, nil
	}
	if err := approveCSR(&kubernetes.Clientset{}, csr); err != nil {
		t.Fatalf("approveCSR success: %v", err)
	}

	approveUpdateApprovalFn = func(_ *kubernetes.Clientset, _ string, _ *certificatesv1.CertificateSigningRequest) (*certificatesv1.CertificateSigningRequest, error) {
		return nil, errors.New("update failed")
	}
	if err := approveCSR(&kubernetes.Clientset{}, csr); err == nil {
		t.Fatalf("approveCSR expected error")
	}
}

func TestCoverage_GetClientset(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	approveClientConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	approveNewForConfigFn = func(*rest.Config) (*kubernetes.Clientset, error) { return &kubernetes.Clientset{}, nil }
	if _, err := GetClientset(); err != nil {
		t.Fatalf("GetClientset direct config: %v", err)
	}

	approveClientConfigFn = func() (*rest.Config, error) { return nil, errors.New("no kubeconfig") }
	approveInClusterConfigFn = func() (*rest.Config, error) { return nil, errors.New("no incluster") }
	if _, err := GetClientset(); err == nil {
		t.Fatalf("GetClientset expected fallback error")
	}

	approveInClusterConfigFn = func() (*rest.Config, error) { return &rest.Config{}, nil }
	approveNewForConfigFn = func(*rest.Config) (*kubernetes.Clientset, error) { return nil, errors.New("new failed") }
	if _, err := GetClientset(); err == nil {
		t.Fatalf("GetClientset expected new client error")
	}
}

func TestCoverage_ApprovePendingCertificates(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	stop := make(chan struct{})
	close(stop)
	called := false
	approvePendingCSRsOnceFn = func(*kubernetes.Clientset) { called = true }
	ApprovePendingCertificates(&kubernetes.Clientset{}, stop)
	if called {
		t.Fatalf("expected no loop execution when stop is closed")
	}

	stop2 := make(chan struct{})
	count := 0
	approvePendingCSRsOnceFn = func(*kubernetes.Clientset) {
		count++
		close(stop2)
	}
	approvePendingSleepFn = func(time.Duration) {}
	ApprovePendingCertificates(&kubernetes.Clientset{}, stop2)
	if count != 1 {
		t.Fatalf("expected one loop execution, got %d", count)
	}
}

func TestCoverage_ApprovePendingCSRsOnce(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	approveListCSRsFn = func(*kubernetes.Clientset) (int, []interface{}, error) {
		return 0, nil, errors.New("list failed")
	}
	approvePendingCSRsOnce(&kubernetes.Clientset{})

	approveListCSRsFn = func(*kubernetes.Clientset) (int, []interface{}, error) {
		items := []interface{}{
			certificatesv1.CertificateSigningRequest{ObjectMeta: metav1.ObjectMeta{Name: "a"}},
			certificatesv1.CertificateSigningRequest{ObjectMeta: metav1.ObjectMeta{Name: "b"}},
			certificatesv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{Name: "c"},
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{{Type: certificatesv1.CertificateApproved}},
				},
			},
		}
		return len(items), items, nil
	}

	seen := map[string]int{}
	approveCSRLoopFn = func(_ *kubernetes.Clientset, csr interface{}) error {
		typed := csr.(certificatesv1.CertificateSigningRequest)
		seen[typed.Name]++
		if typed.Name == "b" {
			return errors.New("approve b failed")
		}
		return nil
	}
	approvePendingCSRsOnce(&kubernetes.Clientset{})

	if seen["a"] != 1 || seen["b"] != 1 || seen["c"] != 0 {
		t.Fatalf("unexpected approve loop calls: %+v", seen)
	}
}

func TestCoverage_CheckStatus(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	checkStatusNewClientsetFn = func() (*kubernetes.Clientset, error) {
		return nil, errors.New("new clientset failed")
	}
	if err := CheckStatus([]string{"kube-apiserver"}, nil, 1); err == nil {
		t.Fatalf("expected error when clientset creation fails")
	}

	base := time.Now()
	times := []time.Time{base, base, base.Add(2 * time.Minute)}
	i := 0
	checkStatusNowFn = func() time.Time {
		if i >= len(times) {
			return times[len(times)-1]
		}
		tm := times[i]
		i++
		return tm
	}
	checkStatusNewClientsetFn = func() (*kubernetes.Clientset, error) { return &kubernetes.Clientset{}, nil }
	checkStatusSleepFn = func(time.Duration) {}
	checkStatusListPodsFn = func(*kubernetes.Clientset) ([]corev1.Pod, error) { return nil, errors.New("list failed") }
	if err := CheckStatus([]string{"kube-apiserver"}, nil, 1); err == nil {
		t.Fatalf("expected timeout/list error")
	}

	checkStatusNowFn = func() time.Time { return base }
	checkStatusListPodsFn = func(*kubernetes.Clientset) ([]corev1.Pod, error) {
		return []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "kube-apiserver-a"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}}}, nil
	}
	checkStatusAllRunningFn = func(requiredPods []string, excludePods []string, pods []corev1.Pod) bool { return true }
	if err := CheckStatus([]string{"kube-apiserver"}, nil, 1); err != nil {
		t.Fatalf("expected success: %v", err)
	}

	times = []time.Time{base, base, base.Add(2 * time.Minute)}
	i = 0
	checkStatusNowFn = func() time.Time {
		if i >= len(times) {
			return times[len(times)-1]
		}
		tm := times[i]
		i++
		return tm
	}
	checkStatusListPodsFn = func(*kubernetes.Clientset) ([]corev1.Pod, error) {
		return []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "kube-apiserver-a"}, Status: corev1.PodStatus{Phase: corev1.PodPending}}}, nil
	}
	checkStatusAllRunningFn = func(requiredPods []string, excludePods []string, pods []corev1.Pod) bool { return false }
	if err := CheckStatus([]string{"kube-apiserver"}, nil, 1); err == nil {
		t.Fatalf("expected timeout when pods are not all running")
	}
}

func TestCoverage_AllRequiredPodsRunningAndExcluded(t *testing.T) {
	required := []string{"kube-apiserver", "kube-controller-manager"}
	exclude := []string{"kube-proxy"}
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "kube-apiserver-a"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		{ObjectMeta: metav1.ObjectMeta{Name: "kube-controller-manager-a"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
	}
	if !allRequiredPodsRunning(required, exclude, pods) {
		t.Fatalf("expected true when required pods are running")
	}

	pods[0].Status.Phase = corev1.PodPending
	if allRequiredPodsRunning(required, exclude, pods) {
		t.Fatalf("expected false when a required pod is not running")
	}

	excludedPods := []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "kube-proxy-a"}, Status: corev1.PodStatus{Phase: corev1.PodPending}}}
	if !allRequiredPodsRunning([]string{"kube-apiserver"}, exclude, excludedPods) {
		t.Fatalf("expected true due excluded pod shortcut")
	}

	if !isExcludedPod("kube-proxy-a", exclude) {
		t.Fatalf("expected kube-proxy to be excluded")
	}
	if isExcludedPod("kube-apiserver-a", exclude) {
		t.Fatalf("did not expect apiserver to be excluded")
	}
}

func TestCoverage_DefaultCheckStatusListPodsFn(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	clientset, err := kubernetes.NewForConfig(&rest.Config{Host: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("new clientset: %v", err)
	}

	if _, err := checkStatusListPodsFn(clientset); err == nil {
		t.Fatalf("expected list pods error from unreachable host")
	}
}

func TestCoverage_NewKubernetesClientset(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	checkstatusBuildConfigFn = func(string, string) (*rest.Config, error) { return nil, errors.New("build failed") }
	if _, err := newKubernetesClientset(); err == nil {
		t.Fatalf("expected build config error")
	}

	checkstatusBuildConfigFn = func(string, string) (*rest.Config, error) { return &rest.Config{}, nil }
	checkstatusNewForConfigFn = func(*rest.Config) (*kubernetes.Clientset, error) { return nil, errors.New("new failed") }
	if _, err := newKubernetesClientset(); err == nil {
		t.Fatalf("expected new for config error")
	}

	checkstatusNewForConfigFn = func(*rest.Config) (*kubernetes.Clientset, error) { return &kubernetes.Clientset{}, nil }
	if _, err := newKubernetesClientset(); err != nil {
		t.Fatalf("expected success: %v", err)
	}
}

func TestCoverage_GetKubeClient(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	clientHomeDirFn = func() string { return "/tmp" }
	clientBuildConfigFn = func(string, string) (*rest.Config, error) { return nil, errors.New("build failed") }
	if _, err := getKubeClient(); err == nil {
		t.Fatalf("expected build config error")
	}

	clientBuildConfigFn = func(string, string) (*rest.Config, error) { return &rest.Config{}, nil }
	clientNewRuntimeClientFn = func(*rest.Config, client.Options) (client.Client, error) { return nil, errors.New("new client failed") }
	if _, err := getKubeClient(); err == nil {
		t.Fatalf("expected runtime client error")
	}

	clientNewRuntimeClientFn = func(*rest.Config, client.Options) (client.Client, error) { return nil, nil }
	if _, err := getKubeClient(); err != nil {
		t.Fatalf("expected success: %v", err)
	}
}

func TestCoverage_BuildKustomizeYAML(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	tmp := t.TempDir()
	fileInfo, err := os.Stat(tmp)
	if err != nil {
		t.Fatalf("stat temp dir: %v", err)
	}

	kustomizeStatFn = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	if _, err := buildKustomizeYAML("missing"); err == nil {
		t.Fatalf("expected stat error")
	}

	kustomizeStatFn = func(string) (os.FileInfo, error) { return fileInfo, nil }
	kustomizeRunFn = func(string) (resmap.ResMap, error) { return nil, errors.New("run failed") }
	if _, err := buildKustomizeYAML("dir"); err == nil {
		t.Fatalf("expected run error")
	}

	kustomizeRunFn = func(string) (resmap.ResMap, error) { return nil, nil }
	kustomizeAsYAMLFn = func(resmap.ResMap) ([]byte, error) { return nil, errors.New("yaml failed") }
	if _, err := buildKustomizeYAML("dir"); err == nil {
		t.Fatalf("expected as yaml error")
	}

	kustomizeAsYAMLFn = func(resmap.ResMap) ([]byte, error) { return []byte("kind: List\n"), nil }
	if _, err := buildKustomizeYAML("dir"); err != nil {
		t.Fatalf("expected success: %v", err)
	}
}

func TestCoverage_RunKustomize(t *testing.T) {
	tmp := t.TempDir()
	if _, err := runKustomize(filepath.Join(tmp, "missing")); err == nil {
		t.Fatalf("expected error for missing dir")
	}

	dir := filepath.Join(tmp, "kustom")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- cm.yaml\n"), 0o644); err != nil {
		t.Fatalf("write kustomization: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cm.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cfg\n"), 0o644); err != nil {
		t.Fatalf("write cm: %v", err)
	}
	if _, err := runKustomize(dir); err != nil {
		t.Fatalf("expected runKustomize success: %v", err)
	}
}

func TestCoverage_KubectlApply(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	tmp := t.TempDir()
	f := filepath.Join(tmp, "r.yaml")
	if err := os.WriteFile(f, []byte("kind: List\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	applyStatFn = func(string) (os.FileInfo, error) { return nil, nil }
	applyReadFileFn = func(string) ([]byte, error) { return nil, errors.New("read failed") }
	if err := KubectlApply(context.Background(), f); err == nil {
		t.Fatalf("expected read error")
	}

	applyReadFileFn = os.ReadFile
	applyGetKubeClientFn = func() (client.Client, error) { return nil, errors.New("client failed") }
	if err := KubectlApply(context.Background(), f); err == nil {
		t.Fatalf("expected client error")
	}

	applyGetKubeClientFn = func() (client.Client, error) { return nil, nil }
	applyYAMLFn = func(context.Context, client.Client, []byte) error { return errors.New("apply failed") }
	if err := KubectlApply(context.Background(), f); err == nil {
		t.Fatalf("expected apply error")
	}

	applyYAMLFn = func(context.Context, client.Client, []byte) error { return nil }
	if err := KubectlApply(context.Background(), f); err != nil {
		t.Fatalf("expected success: %v", err)
	}
}

func TestCoverage_KubectlApplyKustomize(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	tmp := t.TempDir()
	applyStatFn = func(string) (os.FileInfo, error) { return nil, nil }

	applyBuildKustomizeYAMLFn = func(string) ([]byte, error) { return nil, errors.New("build failed") }
	if err := KubectlApplyKustomize(context.Background(), tmp); err == nil {
		t.Fatalf("expected build error")
	}

	applyBuildKustomizeYAMLFn = func(string) ([]byte, error) { return []byte("kind: List\n"), nil }
	applyGetKubeClientFn = func() (client.Client, error) { return nil, errors.New("client failed") }
	if err := KubectlApplyKustomize(context.Background(), tmp); err == nil {
		t.Fatalf("expected client error")
	}

	applyGetKubeClientFn = func() (client.Client, error) { return nil, nil }
	applyYAMLFn = func(context.Context, client.Client, []byte) error { return errors.New("apply failed") }
	if err := KubectlApplyKustomize(context.Background(), tmp); err == nil {
		t.Fatalf("expected apply error")
	}

	applyYAMLFn = func(context.Context, client.Client, []byte) error { return nil }
	if err := KubectlApplyKustomize(context.Background(), tmp); err != nil {
		t.Fatalf("expected success: %v", err)
	}
}

func TestCoverage_ApplyYAMLAndNode(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	yamlReadNodesFn = func([]byte) ([]*kyaml.RNode, error) { return nil, errors.New("parse failed") }
	if err := applyYAML(context.Background(), nil, []byte("x")); err == nil {
		t.Fatalf("expected parse error")
	}

	yamlReadNodesFn = func(data []byte) ([]*kyaml.RNode, error) {
		reader := kio.ByteReader{Reader: bytes.NewReader(data)}
		return reader.Read()
	}
	yamlNodeToUnstructuredFn = func(string) (*unstructured.Unstructured, error) { return nil, errors.New("convert failed") }
	if err := applyYAML(context.Background(), nil, []byte("kind: ConfigMap\nmetadata:\n  name: x\n")); err == nil {
		t.Fatalf("expected convert error")
	}

	yamlNodeToUnstructuredFn = nodeToUnstructured
	yamlApplyObjectWithRetryFn = func(context.Context, client.Client, *unstructured.Unstructured) error {
		return errors.New("retry failed")
	}
	if err := applyYAML(context.Background(), nil, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n")); err == nil {
		t.Fatalf("expected retry error")
	}

	yamlApplyObjectWithRetryFn = func(context.Context, client.Client, *unstructured.Unstructured) error { return nil }
	if err := applyYAML(context.Background(), nil, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n")); err != nil {
		t.Fatalf("expected success: %v", err)
	}

	node, err := kyaml.Parse("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cfg\n")
	if err != nil {
		t.Fatalf("parse node: %v", err)
	}
	if _, err := nodeToUnstructured(node.MustString()); err != nil {
		t.Fatalf("nodeToUnstructured success: %v", err)
	}

	if _, err := nodeToUnstructured("not: [valid"); err == nil {
		t.Fatalf("expected nodeToUnstructured error")
	}
}

func TestCoverage_ApplyObjectWithRetryAndNormalize(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	if normalizeContext(nil) != context.Background() {
		t.Fatalf("expected nil context to normalize to background")
	}
	reqCtx := context.Background()
	if normalizeContext(reqCtx) != reqCtx {
		t.Fatalf("expected context to remain unchanged")
	}

	obj := &unstructured.Unstructured{}
	obj.SetName("o")

	calls := 0
	yamlPatchObjectFn = func(context.Context, client.Client, *unstructured.Unstructured) error {
		calls++
		return nil
	}
	if err := applyObjectWithRetry(context.Background(), nil, obj); err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected single patch call, got %d", calls)
	}

	calls = 0
	yamlPatchObjectFn = func(context.Context, client.Client, *unstructured.Unstructured) error {
		calls++
		if calls == 1 {
			return errors.New("retry")
		}
		return nil
	}
	yamlApplySleepFn = func(time.Duration) {}
	if err := applyObjectWithRetry(context.Background(), nil, obj); err != nil {
		t.Fatalf("expected eventual success: %v", err)
	}

	yamlPatchObjectFn = func(context.Context, client.Client, *unstructured.Unstructured) error {
		return errors.New("always fail")
	}
	if err := applyObjectWithRetry(context.Background(), nil, obj); err == nil {
		t.Fatalf("expected terminal error")
	}
}

func TestCoverage_OriginalDefaultClosures(t *testing.T) {
	resetKubectlHooks(t)
	defer resetKubectlHooks(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/apis/certificates.k8s.io/v1/certificatesigningrequests", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apiVersion":"certificates.k8s.io/v1","kind":"CertificateSigningRequestList","items":[{"metadata":{"name":"csr1"}}]}`))
	})
	mux.HandleFunc("/apis/certificates.k8s.io/v1/certificatesigningrequests/csr1/approval", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apiVersion":"certificates.k8s.io/v1","kind":"CertificateSigningRequest","metadata":{"name":"csr1"}}`))
	})
	mux.HandleFunc("/api/v1/pods", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[]}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	clientset, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatalf("new clientset: %v", err)
	}
	unreachableClientset, err := kubernetes.NewForConfig(&rest.Config{Host: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("new unreachable clientset: %v", err)
	}

	approveUpdateApprovalFn = origApproveUpdateApprovalFn
	count, items, err := origApproveListCSRsFn(clientset)
	if err != nil {
		t.Fatalf("origApproveListCSRsFn: %v", err)
	}
	if count != 1 || len(items) != 1 {
		t.Fatalf("unexpected csr list result: count=%d len=%d", count, len(items))
	}

	if err := origApproveCSRLoopFn(clientset, certificatesv1.CertificateSigningRequest{ObjectMeta: metav1.ObjectMeta{Name: "csr1"}}); err != nil {
		t.Fatalf("origApproveCSRLoopFn: %v", err)
	}

	if _, err := origCheckStatusListPodsFn(clientset); err != nil {
		t.Fatalf("origCheckStatusListPodsFn: %v", err)
	}
	if _, _, err := origApproveListCSRsFn(unreachableClientset); err == nil {
		t.Fatalf("expected error from origApproveListCSRsFn with unreachable host")
	}
	if _, err := origCheckStatusListPodsFn(unreachableClientset); err == nil {
		t.Fatalf("expected error from origCheckStatusListPodsFn with unreachable host")
	}

	if _, err := origYAMLReadNodesFn([]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n")); err != nil {
		t.Fatalf("origYAMLReadNodesFn: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	fakeClient := crfake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cfg", Namespace: "default"}}).Build()
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("ConfigMap")
	u.SetName("cfg")
	u.SetNamespace("default")
	_ = origYAMLPatchObjectFn(context.Background(), fakeClient, u)

	dir := filepath.Join(t.TempDir(), "kustom")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- cm.yaml\n"), 0o644); err != nil {
		t.Fatalf("write kustomization: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cm.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cfg\n"), 0o644); err != nil {
		t.Fatalf("write cm: %v", err)
	}
	resMap, err := runKustomize(dir)
	if err != nil {
		t.Fatalf("runKustomize: %v", err)
	}
	if _, err := origKustomizeAsYAMLFn(resMap); err != nil {
		t.Fatalf("origKustomizeAsYAMLFn: %v", err)
	}
}
