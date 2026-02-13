package kubectlcmds

import (
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	clientHomeDirFn          = homedir.HomeDir
	clientBuildConfigFn      = clientcmd.BuildConfigFromFlags
	clientNewRuntimeClientFn = client.New
)

// getKubeClient initializes and returns a controller-runtime client.Client
func getKubeClient() (client.Client, error) {
	log.Trace().Msg("Initializing Kubernetes client")

	kubeconfig := filepath.Join(clientHomeDirFn(), ".kube", "config")
	config, err := clientBuildConfigFn("", kubeconfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load kubeconfig")
		return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	kubeClient, err := clientNewRuntimeClientFn(config, client.Options{})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Kubernetes client")
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	log.Debug().Msg("Successfully initialized Kubernetes client")
	return kubeClient, nil
}
