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
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	syaml "sigs.k8s.io/yaml"
)

var (
	yamlReadNodesFn = func(yamlData []byte) ([]*kyaml.RNode, error) {
		reader := kio.ByteReader{Reader: bytes.NewReader(yamlData)}
		return reader.Read()
	}
	yamlNodeToUnstructuredFn   = nodeToUnstructured
	yamlApplyObjectWithRetryFn = applyObjectWithRetry
	yamlPatchObjectFn          = func(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) error {
		return k8sClient.Patch(ctx, obj, client.Apply, client.FieldOwner("kustomize-controller"))
	}
	yamlApplySleepFn       = time.Sleep
	yamlNormalizeContextFn = normalizeContext
)

func applyYAML(ctx context.Context, k8sClient client.Client, yamlData []byte) error {
	log.Trace().Msg("Applying YAML data to the Kubernetes cluster")

	nodes, err := yamlReadNodesFn(yamlData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse YAML")
		return fmt.Errorf("failed to parse YAML: %v", err)
	}

	for _, node := range nodes {
		obj, err := yamlNodeToUnstructuredFn(node.MustString())
		if err != nil {
			return err
		}

		if err := yamlApplyObjectWithRetryFn(ctx, k8sClient, obj); err != nil {
			return err
		}

		log.Info().Msgf("Successfully applied object: %s of kind: %s in namespace: %s", obj.GetName(), obj.GetKind(), obj.GetNamespace())
	}

	log.Debug().Msg("YAML application completed")
	return nil
}

func nodeToUnstructured(nodeYAML string) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	if err := syaml.Unmarshal([]byte(nodeYAML), obj); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal node")
		return nil, fmt.Errorf("failed to unmarshal node: %v", err)
	}

	return obj, nil
}

func applyObjectWithRetry(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) error {
	requestCtx := yamlNormalizeContextFn(ctx)

	if err := yamlPatchObjectFn(requestCtx, k8sClient, obj); err != nil {
		log.Warn().Err(err).Msg("Failed to apply yaml object... trying again in 15 seconds...")
		yamlApplySleepFn(15 * time.Second)
		if err := yamlPatchObjectFn(requestCtx, k8sClient, obj); err != nil {
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
