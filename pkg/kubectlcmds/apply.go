package kubectlcmds

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

var (
	applyStatFn               = os.Stat
	applyReadFileFn           = os.ReadFile
	applyGetKubeClientFn      = getKubeClient
	applyYAMLFn               = applyYAML
	applyBuildKustomizeYAMLFn = buildKustomizeYAML
)

// KubectlApply applies a YAML file to the Kubernetes cluster and filters the logs
func KubectlApply(ctx context.Context, filePath string) error {
	log.Trace().Msgf("Applying YAML file at path: %s", filePath)

	// Check if the file exists
	if _, err := applyStatFn(filePath); os.IsNotExist(err) {
		log.Error().Err(err).Msgf("File does not exist: %s", filePath)
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Read the YAML file
	yamlData, err := applyReadFileFn(filePath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read YAML file")
		return fmt.Errorf("failed to read YAML file: %v", err)
	}

	// Initialize Kubernetes client
	k8sClient, err := applyGetKubeClientFn()
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize Kubernetes client")
		return err
	}

	// Apply the YAML to the cluster
	if err := applyYAMLFn(ctx, k8sClient, yamlData); err != nil {
		log.Error().Err(err).Msg("Failed to apply YAML")
		return fmt.Errorf("failed to apply YAML: %v", err)
	}

	// Filter the logs
	log.Info().Msg("KubectlApply operation completed")

	return nil
}

// KubectlApplyKustomize applies a kustomize directory or file to the Kubernetes cluster and filters the logs
func KubectlApplyKustomize(ctx context.Context, filePath string) error {
	log.Trace().Msgf("Applying Kustomize directory or file at path: %s", filePath)

	// Check if the path exists
	if _, err := applyStatFn(filePath); os.IsNotExist(err) {
		log.Error().Err(err).Msgf("Path does not exist: %s", filePath)
		return fmt.Errorf("path does not exist: %s", filePath)
	}

	yamlData, err := applyBuildKustomizeYAMLFn(filePath)
	if err != nil {
		return err
	}

	// Initialize Kubernetes client
	k8sClient, err := applyGetKubeClientFn()
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize Kubernetes client")
		return err
	}

	// Apply the YAML to the cluster
	if err := applyYAMLFn(ctx, k8sClient, yamlData); err != nil {
		log.Error().Err(err).Msg("Failed to apply YAML from kustomize")
		return fmt.Errorf("failed to apply YAML from kustomize: %v", err)
	}

	log.Info().Msg("KubectlApplyKustomize operation completed")

	return nil
}
