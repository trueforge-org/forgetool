package talassist

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	talhelperCfg "github.com/budimanjojo/talhelper/v3/pkg/config"
	"github.com/budimanjojo/talhelper/v3/pkg/generate"
	talhelperTalos "github.com/budimanjojo/talhelper/v3/pkg/talos"
	"github.com/invopop/jsonschema"
	"github.com/rs/zerolog/log"
	sideroConfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

var (
	TalConfig          *talhelperCfg.TalhelperConfig
	LatestTalosVersion string

	loadAndValidateFromFileFn    = talhelperCfg.LoadAndValidateFromFile
	parseContractFromVersionFn   = sideroConfig.ParseContractFromVersion
	newSecretBundleFn            = talhelperTalos.NewSecretBundle
	generateConfigFn             = generate.GenerateConfig
	talConfigGenerateGitignoreFn = defaultTalConfigGenerateGitignore
	mkdirAllFn                   = os.MkdirAll
	writeFileFn                  = os.WriteFile
	talassistFatalFn             = defaultTalassistFatal
	talassistExitFn              = os.Exit
)

func defaultTalConfigGenerateGitignore(cfg *talhelperCfg.TalhelperConfig, outPath string) error {
	if cfg == nil {
		return fmt.Errorf("nil talconfig")
	}
	return cfg.GenerateGitignore(outPath)
}

func defaultTalassistFatal(err error, msg string) {
	log.Error().Err(err).Msg(msg)
}

func LoadTalConfig() {
	cfg, err := loadAndValidateFromFileFn(helper.TalConfigFile, []string{helper.ClusterEnvFile}, false)
	if err != nil {
		talassistFatalFn(err, "failed to parse talconfig or talenv file")
		talassistExitFn(1)
		return
	}
	TalConfig = cfg
	LatestTalosVersion = talhelperCfg.LatestTalosVersion
	return
}

func GenSchema() error {
	cfg := talhelperCfg.TalhelperConfig{}
	r := new(jsonschema.Reflector)
	r.FieldNameTag = "yaml"
	r.RequiredFromJSONSchemaTags = true
	mkdirAllFn(helper.ClusterPath+"/talos", os.ModePerm)
	var genschemaFile = path.Join(helper.ClusterPath, "/talos/talconfig.json")

	schema := r.Reflect(&cfg)
	data, _ := json.MarshalIndent(schema, "", "  ")
	if err := writeFileFn(genschemaFile, data, os.FileMode(0o644)); err != nil {
		talassistFatalFn(err, "failed to write talconfig schema")
		talassistExitFn(1)
		return err
	}
	return nil
}

func NewSecretBundle() *secrets.Bundle {
	version, _ := parseContractFromVersionFn(LatestTalosVersion)
	s, err := newSecretBundleFn(secrets.NewClock(), *version)
	if err != nil {
		log.Error().Msgf("Error loading secret bundle %s", err)
	}
	return s
}

func TalhelperGenConfig() error {
	genconfigTalosMode := "metal"
	genconfigNoGitignore := false
	genconfigDryRun := false
	genconfigOfflineMode := false
	genconfigCrtTTL := constants.TalosAPIDefaultCertificateValidityDuration

	err := generateConfigFn(TalConfig, genconfigDryRun, helper.TalosGenerated, helper.TalSecretFile, genconfigTalosMode, genconfigOfflineMode, false, genconfigCrtTTL)
	if err != nil {
		talassistFatalFn(err, "failed to generate talos config")
		talassistExitFn(1)
		return err
	}

	if !genconfigNoGitignore && !genconfigDryRun {
		err = talConfigGenerateGitignoreFn(TalConfig, helper.TalosGenerated)
		if err != nil {
			talassistFatalFn(err, "failed to generate gitignore file")
			talassistExitFn(1)
			return err
		}
	}
	return nil
}
