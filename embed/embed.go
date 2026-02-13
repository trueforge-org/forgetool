package embed

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog/log"

	"github.com/leaanthony/debme"
	"github.com/trueforge-org/forgetool/pkg/clustertemplate"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

type readDirFS interface {
	fs.FS
	fs.ReadFileFS
}

var removeAll = os.RemoveAll
var mkdirAll = os.MkdirAll
var writeFile = os.WriteFile
var walkDir = fs.WalkDir
var fromEmbeddedFS = func(embeddedFS embed.FS, sub string) (readDirFS, error) {
	return debme.FS(embeddedFS, sub)
}
var clusterTemplateToCache = clustertemplate.ToCache
var fatalErr = func(err error) {
	log.Fatal().Err(err)
}

func AllToCache() {
	err := removeAll(helper.CacheDir)
	if err != nil {
		fatalErr(err)
		return
	}
	GOOSARCH := runtime.GOOS + "_" + runtime.GOARCH
	filesToCache(StaticFiles, GOOSARCH)
	if err := clusterTemplateToCache(); err != nil {
		log.Warn().Err(err).Msg("Failed to cache cluster-template release; verify FORGETOOL_CLUSTER_TEMPLATE_VERSION is a valid release tag or check network connectivity to GitHub")
	}
}

func filesToCache(embededfs embed.FS, sub string) {

	// Ensure the base cache directory exists
	if err := mkdirAll(helper.CacheDir, os.ModePerm); err != nil {
		log.Info().Msgf("Error creating base cache directory: %v", err)
		return
	}

	root, _ := fromEmbeddedFS(embededfs, sub)
	walkDir(root, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() != sub {
			if d.IsDir() {
				// If it's a directory, create the corresponding directory in the cache
				writePath := filepath.Join(helper.CacheDir, path)
				if err := os.MkdirAll(writePath, os.ModePerm); err != nil {
					log.Info().Msgf("Error creating directory in cache: %v", err)
					return err
				}
			} else {

				// If it's a file, read and write it to the cache
				data, err := root.ReadFile(path)
				if err != nil {
					log.Info().Msgf("Error reading file: %v", err)
					return err
				}
				writePath := filepath.Join(helper.CacheDir, path)
				if err := writeFile(writePath, data, 0755); err != nil {
					log.Info().Msgf("Error writing file to cache: %v", err)
					return err
				}
			}
		}
		return nil
	})
}
