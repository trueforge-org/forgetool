package kubectlcmds

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/yaml"
)

func applyYAML(ctx context.Context, k8sClient client.Client, yamlData []byte) error {
	log.Trace().Msg("Applying YAML data to the Kubernetes cluster")

	reader := kio.ByteReader{Reader: bytes.NewReader(yamlData)}
	nodes, err := reader.Read()
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse YAML")
		return fmt.Errorf("failed to parse YAML: %v", err)
	}

	for _, node := range nodes {
		obj, err := nodeToUnstructured(node.MustString())
		if err != nil {
			return err
		}

		if err := applyObjectWithRetry(ctx, k8sClient, obj); err != nil {
			return err
		}

		log.Info().Msgf("Successfully applied object: %s of kind: %s in namespace: %s", obj.GetName(), obj.GetKind(), obj.GetNamespace())
	}

	log.Debug().Msg("YAML application completed")
	return nil
}

func nodeToUnstructured(nodeYAML string) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(nodeYAML), obj); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal node")
		return nil, fmt.Errorf("failed to unmarshal node: %v", err)
	}

	return obj, nil
}

func applyObjectWithRetry(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) error {
	requestCtx := normalizeContext(ctx)

	if err := k8sClient.Patch(requestCtx, obj, client.Apply, client.FieldOwner("kustomize-controller")); err != nil {
		log.Warn().Err(err).Msg("Failed to apply yaml object... trying again in 15 seconds...")
		time.Sleep(15 * time.Second)
		if err := k8sClient.Patch(requestCtx, obj, client.Apply, client.FieldOwner("kustomize-controller")); err != nil {
			log.Error().Err(err).Msg("Failed to apply object")
			return fmt.Errorf("failed to apply object: %v", err)
		}
	}

	return nil
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}
