package kubectlcmds

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func CheckStatus(requiredPods []string, excludePod []string, timeout time.Duration) error {
	log.Trace().Msg("Starting CheckStatus function")
	clientset, err := newKubernetesClientset()
	if err != nil {
		return err
	}

	// Maximum duration to wait (timeout in minutes)
	maxDuration := timeout * time.Minute
	endTime := time.Now().Add(maxDuration)

	log.Info().Msg("Checking status of required pods")
	log.Debug().Msgf("required pods: %v, excluding pods: %v", requiredPods, excludePod)

	for time.Now().Before(endTime) {
		log.Debug().Msg("Retrieving list of pods")

		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Debug().Err(err).Msg("Error listing pods")
			log.Warn().Msg("Cannot recieve pods (yet), waiting before checking again")
			time.Sleep(5 * time.Second)
			continue
		}

		if allRequiredPodsRunning(requiredPods, excludePod, pods.Items) {
			log.Info().Msg("All required pods are running")
			return nil
		}

		log.Warn().Msg("Not all required pods are running, waiting before checking again")
		// Wait for 5 seconds before checking again
		time.Sleep(5 * time.Second)
	}

	log.Error().Msg("Timeout: Not all required pods are running after 15 minutes")
	return fmt.Errorf("timeout: not all required pods are running after 15 minutes")
}

func newKubernetesClientset() (*kubernetes.Clientset, error) {
	kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Error().Err(err).Msg("Error loading kubeconfig")
		return nil, fmt.Errorf("error loading kubeconfig: %w", err)
	}
	log.Debug().Msg("Kubeconfig loaded successfully")

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error().Err(err).Msg("Error creating Kubernetes clientset")
		return nil, fmt.Errorf("error creating clientset: %w", err)
	}
	log.Debug().Msg("Kubernetes clientset created successfully")

	return clientset, nil
}

func allRequiredPodsRunning(requiredPods []string, excludePods []string, pods []corev1.Pod) bool {
	requiredPodsMap := make(map[string]bool)
	for _, pod := range requiredPods {
		requiredPodsMap[pod] = false
	}

	log.Debug().Msg("Checking pod statuses")
	for _, pod := range pods {
		for _, requiredPod := range requiredPods {
			if isExcludedPod(pod.Name, excludePods) {
				requiredPodsMap[requiredPod] = true
				continue
			}
			if strings.Contains(pod.Name, requiredPod) && pod.Status.Phase == "Running" {
				requiredPodsMap[requiredPod] = true
				log.Debug().Str("podName", pod.Name).Msgf("Required pod %s is running", requiredPod)
			}
		}
	}

	for _, isRunning := range requiredPodsMap {
		if !isRunning {
			return false
		}
	}

	return true
}

func isExcludedPod(podName string, excludePods []string) bool {
	for _, excludePod := range excludePods {
		if strings.Contains(podName, excludePod) {
			log.Debug().Str("excludedPod", excludePod).Msg("Excluding pod from check")
			return true
		}
	}
	return false
}
