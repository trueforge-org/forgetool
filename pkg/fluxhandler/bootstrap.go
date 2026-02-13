package fluxhandler

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/kubectlcmds"
)

var (
	fluxGetYesOrNoFn               = helper.GetYesOrNo
	fluxBootstrapFluxCDFn          = bootstrapFluxCD
	fluxExitFn                     = os.Exit
	fluxLogFatalErrMsgFn           = func(err error, msg string) { log.Error().Err(err).Msg(msg); fluxExitFn(1) }
	fluxFatalErrMsgFn              = func(err error, msg string) { fluxLogFatalErrMsgFn(err, msg) }
	fluxCheckGitRepoFn             = checkGitRepo
	fluxSetupFluxCDFn              = setupFluxCD
	fluxSetupRepositoriesFn        = setupRepositories
	fluxKubectlApplyFn             = kubectlcmds.KubectlApply
	fluxIsCurrentDirGitRepoFn      = helper.IsCurrentDirGitRepo
	fluxKubectlApplyKustomizeFn    = kubectlcmds.KubectlApplyKustomize
	fluxRenameFluxBootstrapFilesFn = renameFluxBootstrapFiles
	fluxRevertFluxBootstrapFilesFn = revertFluxBootstrapFiles
	fluxOSRenameFn                 = os.Rename
)

func init() {
	// Configure zerolog to output to stdout with a timestamp and log level
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
}

// FluxBootstrap initializes the FluxCD bootstrapping process if GITHUB_REPOSITORY is set in TalEnv.
func FluxBootstrap(ctx context.Context) {

	if helper.TalEnv["GITHUB_REPOSITORY"] != "" {
		log.Info().Msg("GITHUB_Repository for Flux configured.")
		if fluxGetYesOrNoFn("Do you want to (re)bootstrap FluxCD as well? (yes/no) [y/n]: ", true) {
			if err := fluxBootstrapFluxCDFn(ctx); err != nil {
				fluxFatalErrMsgFn(err, "Error during FluxCD bootstrap")
				if fluxGetYesOrNoFn("Do you want to retry? (yes/no) [y/n]: ", true) {
					if err2 := fluxBootstrapFluxCDFn(ctx); err2 != nil {
						fluxFatalErrMsgFn(err2, "Error during FluxCD bootstrap")
					}
				}
			}
			log.Info().Msg("FluxCD Bootstrapped successfully")
		}
	}
}

// bootstrapFluxCD handles the entire FluxCD bootstrapping process.
func bootstrapFluxCD(ctx context.Context) error {
	if err := fluxCheckGitRepoFn(); err != nil {
		return err
	}

	fluxPath := filepath.Join(helper.ClusterPath, "kubernetes", "flux-system", "flux")
	if err := fluxSetupFluxCDFn(ctx, fluxPath); err != nil {
		return err
	}

	reposFilePath := "repositories"
	if err := fluxSetupRepositoriesFn(ctx, reposFilePath); err != nil {
		return err
	}

	clusterEntryFile := filepath.Join(helper.ClusterPath, "kubernetes", "flux-entry.yaml")
	if err := fluxKubectlApplyFn(ctx, clusterEntryFile); err != nil {
		log.Error().Err(err).Str("path", clusterEntryFile).Msg("Error applying Kubernetes flux-entry manifest")
		return err
	}

	return nil
}

// checkGitRepo verifies if the current directory is a valid Git repository.
func checkGitRepo() error {
	isRepo, err := fluxIsCurrentDirGitRepoFn()
	if err != nil {
		log.Error().Err(err).Msg("Error checking Git repository")
		return err
	}
	if !isRepo {
		errMsg := "Bootstrap: ERROR The current directory is not a Git repository. Cannot bootstrap fluxcd"
		log.Error().Msg(errMsg)
		return errors.New(errMsg)
	}
	log.Info().Msg("Bootstrap: The current directory is a valid GIT repository, continuing...")
	return nil
}

// setupFluxCD handles the setup of FluxCD manifests.
func setupFluxCD(ctx context.Context, fluxPath string) error {
	bootstrapFile := "bootstrap.yaml.ct"
	kustomFile := "kustomization.yaml"
	tmpFile := "placeholder"

	log.Info().Msg("Bootstrap: Loading fluxcd onto the cluster...")

	// Rename files for kustomize application
	if err := fluxRenameFluxBootstrapFilesFn(fluxPath, bootstrapFile, kustomFile, tmpFile); err != nil {
		return err
	}

	if err := fluxKubectlApplyKustomizeFn(ctx, fluxPath); err != nil {
		log.Error().Err(err).Str("path", fluxPath).Msg("Error applying FluxCD manifest")
		log.Debug().Msg("Reverting renamed files for fluxbootstrap")
		if revertErr := fluxRevertFluxBootstrapFilesFn(fluxPath, bootstrapFile, kustomFile, tmpFile); revertErr != nil {
			log.Error().Err(revertErr).Msg("Error reverting Flux bootstrap files")
			return revertErr
		}
		return err
	}

	if err := fluxRevertFluxBootstrapFilesFn(fluxPath, bootstrapFile, kustomFile, tmpFile); err != nil {
		log.Error().Err(err).Msg("Error reverting Flux bootstrap files")
		return err
	}

	return nil
}

func renameFluxBootstrapFiles(fluxPath, bootstrapFile, kustomFile, tmpFile string) error {
	if err := fluxOSRenameFn(filepath.Join(fluxPath, kustomFile), filepath.Join(fluxPath, tmpFile)); err != nil {
		log.Error().Err(err).Msg("Error renaming kustomization file")
		return err
	}
	if err := fluxOSRenameFn(filepath.Join(fluxPath, bootstrapFile), filepath.Join(fluxPath, kustomFile)); err != nil {
		log.Error().Err(err).Msg("Error renaming bootstrap file")
		return err
	}
	return nil
}

func revertFluxBootstrapFiles(fluxPath, bootstrapFile, kustomFile, tmpFile string) error {
	if err := fluxOSRenameFn(filepath.Join(fluxPath, kustomFile), filepath.Join(fluxPath, bootstrapFile)); err != nil {
		log.Error().Err(err).Msg("Error renaming kustomization file back")
		return err
	}
	if err := fluxOSRenameFn(filepath.Join(fluxPath, tmpFile), filepath.Join(fluxPath, kustomFile)); err != nil {
		log.Error().Err(err).Msg("Error renaming placeholder file back")
		return err
	}
	return nil
}

// setupRepositories handles the setup of repository manifests.
func setupRepositories(ctx context.Context, reposFilePath string) error {
	log.Info().Msg("Bootstrap: Loading git-repo manifests onto the cluster...")

	gitRepoFile := filepath.Join(reposFilePath, "git", "this-repo.yaml")
	if err := fluxKubectlApplyFn(ctx, gitRepoFile); err != nil {
		log.Error().Err(err).Str("path", reposFilePath).Msg("Error applying repositories manifest")
		return err
	}

	log.Info().Msg("Bootstrap: Loading repositories flux-entry onto the cluster...")
	reposEntryFile := filepath.Join(reposFilePath, "flux-entry.yaml")
	if err := fluxKubectlApplyFn(ctx, reposEntryFile); err != nil {
		log.Error().Err(err).Str("path", reposEntryFile).Msg("Error applying repositories flux-entry manifest")
		return err
	}

	return nil
}
