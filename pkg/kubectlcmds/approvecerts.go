package kubectlcmds

import (
	"context"

	"github.com/rs/zerolog/log"
	certificatesv1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// getClientset creates a Kubernetes clientset from the in-cluster config or kubeconfig file
func GetClientset() (*kubernetes.Clientset, error) {
	log.Trace().Msg("Attempting to create Kubernetes clientset")

	// Load config from the current kubeconfig context.
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load kubeconfig, attempting in-cluster config")
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Error().Err(err).Msg("Failed to create in-cluster config")
			return nil, err
		}
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error().Err(err).Msg("Error creating Kubernetes clientset")
		return nil, err
	}

	log.Info().Msg("Kubernetes clientset created successfully")
	return clientset, nil
}

func approveCSR(clientset *kubernetes.Clientset, csr certificatesv1.CertificateSigningRequest) error {
	csrCopy := csr.DeepCopy()
	csrCopy.Status.Conditions = []certificatesv1.CertificateSigningRequestCondition{
		{
			Type:           certificatesv1.CertificateApproved,
			Reason:         "AutoApproved",
			Message:        "This CSR was approved automatically by controller.",
			LastUpdateTime: metav1.Now(),
			Status:         "True",
		},
	}

	_, err := clientset.CertificatesV1().CertificateSigningRequests().UpdateApproval(context.TODO(), csr.Name, csrCopy, metav1.UpdateOptions{})
	return err
}
