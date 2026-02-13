package kubectlcmds

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

var (
	kustomizeStatFn   = os.Stat
	kustomizeRunFn    = runKustomize
	kustomizeAsYAMLFn = func(r resmap.ResMap) ([]byte, error) { return r.AsYaml() }
)

func buildKustomizeYAML(filePath string) ([]byte, error) {
	fileInfo, err := kustomizeStatFn(filePath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to stat path")
		return nil, fmt.Errorf("failed to stat path: %v", err)
	}

	kustomizePath := resolveKustomizePath(filePath, fileInfo)
	resMap, err := kustomizeRunFn(kustomizePath)
	if err != nil {
		return nil, err
	}

	yamlData, err := kustomizeAsYAMLFn(resMap)
	if err != nil {
		log.Error().Err(err).Msg("Failed to convert kustomize output to YAML")
		return nil, fmt.Errorf("failed to convert kustomize output to YAML: %v", err)
	}

	return yamlData, nil
}

func runKustomize(kustomizePath string) (resmap.ResMap, error) {
	fSys := filesys.MakeFsOnDisk()
	kustomizer := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resMap, err := kustomizer.Run(fSys, kustomizePath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to run kustomize")
		return nil, fmt.Errorf("failed to run kustomize: %v", err)
	}

	return resMap, nil
}

func resolveKustomizePath(filePath string, fileInfo os.FileInfo) string {
	if fileInfo.IsDir() {
		log.Debug().Msgf("Using directory as kustomize path: %s", filePath)
		return filePath
	}

	kustomizePath := filepath.Dir(filePath)
	log.Debug().Msgf("Using file's directory as kustomize path: %s", kustomizePath)
	return kustomizePath
}
