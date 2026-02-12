package initfiles

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/trueforge-org/forgetool/pkg/fluxhandler"
	"github.com/trueforge-org/forgetool/pkg/helper"
	"github.com/trueforge-org/forgetool/pkg/talassist"
)

func InitFiles() error {
	removeRunAgainFile()
	ageGen()
	genRootFiles()
	genBaseFiles()
	UpdateRootFiles()
	UpdateBaseFiles()
	talassist.GenSchema()
	GenPatches()
	genKubernetes()
	GenTalEnvConfigMap()
	UpdateGitRepo()
	fluxhandler.CreateGitSecret(helper.TalEnv["GITHUB_REPOSITORY"])
	GenSopsSecret()
	processKustomizations(path.Join(helper.ClusterPath, "kubernetes"))

	helper.CreateEncrPreCommitHook()
	log.Info().Msg("Init: Completed Successfully!")
	return nil
}

func processKustomizations(kubePath string) {
	if err := fluxhandler.ProcessDirectory(kubePath); err != nil {
		log.Error().Msgf("Error: %v", err)
	}

	if err := fluxhandler.ProcessDirectory(kubePath); err != nil {
		log.Error().Msgf("Error: %v", err)
		return
	}

	log.Info().Msg("Kustomizations processed successfully.")
}

func genKubernetes() error {

	err := helper.CopyDir(helper.KubeCache, helper.ClusterPath+"/kubernetes", false)
	if err != nil {
		log.Info().Msgf("Error: %v", err)
	} else {
		log.Info().Msgf("Kubernetes files copied successfully.")
	}

	helper.ReplaceInFile(path.Join(helper.ClusterPath, "/kubernetes/flux-entry.yaml"), "REPLACEWITHCLUSTERNAME", helper.ClusterName)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error: %s", err)
	}

	return nil
}

func GenTalEnvConfigMap() error {

	log.Info().Msg("Creating TalEnv configmap reference 'clustersettings'.")
	// Read the content of the talenv.yaml file
	talenvContent, err := os.ReadFile(helper.ClusterEnvFile)
	if err != nil {
		return err
	}

	// Convert the file content to a string and split it into lines
	talenvLines := strings.Split(string(talenvContent), "\n")

	// Add indentation to each line
	for i, line := range talenvLines {
		talenvLines[i] = "  " + line
	}
	indentClusterName := "  CLUSTERNAME: " + helper.ClusterName
	talenvLines = append(talenvLines, indentClusterName)

	// Join the indented lines back into a single string
	indentedTalenvContent := strings.Join(talenvLines, "\n")

	clusterSettings := filepath.Join("flux-system", "flux", "clustersettings.secret.yaml")
	clusterSettingsDest := filepath.Join(helper.ClusterPath+"/kubernetes", clusterSettings)
	clusterSettingsSrc := filepath.Join(helper.KubeCache, clusterSettings)
	os.MkdirAll(filepath.Join(helper.ClusterPath, "/kubernetes", "flux-system", "flux"), os.ModePerm)
	err = helper.CopyFile(clusterSettingsSrc, clusterSettingsDest, true)
	log.Debug().Msgf("clusterSettingsDest %v", clusterSettingsDest)
	helper.ReplaceInFile(clusterSettingsDest, "REPLACEWITHENV", indentedTalenvContent)
	if err != nil {
		log.Fatal().Err(err).Msg("Error: %s")
	}
	log.Info().Msg("Configmap reference Created.")
	return nil
}

func UpdateGitRepo() {
	if helper.TalEnv["GITHUB_REPOSITORY"] != "" {
		repoPath := filepath.Join("repositories", "git", "this-repo.yaml")
		gitrepo := FormatGitURL(helper.TalEnv["GITHUB_REPOSITORY"])
		helper.ReplaceInFile(repoPath, "ssh://REPLACEWITHGITREPO", gitrepo)
	}
}

// FormatGitURL formats the input Git URL according to the specified rules.
func FormatGitURL(input string) string {
	// Remove "https://" prefix if present
	input = strings.TrimPrefix(input, "https://")

	if !strings.HasPrefix(input, "ssh://") {
		input = "ssh://" + input
	}

	// Ensure input starts with "ssh://git@"
	if !strings.HasPrefix(input, "ssh://git@") {
		// Prepend "ssh://git@" if neither "ssh://" nor "git@" is present
		input = strings.Replace(input, "ssh://", "ssh://git@", 1)
	}

	if strings.Contains(input, "git@git@") {
		input = strings.Replace(input, "git@git@", "git@", 1)
	}

	// Compile a regex to match and replace the URL pattern
	re := regexp.MustCompile(`^ssh://git@([^:/]+)([:/])([\w-]+)/([\w-]+)\.git$`)
	matches := re.FindStringSubmatch(input)

	if len(matches) == 5 {
		// Determine the user and repo based on the separator used
		user := matches[3] // Always captured as part of the matched group
		repo := matches[4] // Always captured as part of the matched group
		return fmt.Sprintf("ssh://git@%s/%s/%s.git", matches[1], user, repo)
	}

	return input // Return the input as is if it doesn't match
}

func genBaseFiles() error {
	clusterEnvPresent := false

	if _, err := os.Stat(helper.ClusterEnvFile); err == nil {
		clusterEnvPresent = true
		log.Debug().Msg("Detected existing cluster, continuing")
	} else if os.IsNotExist(err) {
		createRunAgainFile()
		log.Warn().Msg("New cluster detected, creating clusterenv.yaml\n Please fill out ClusterEnv.yaml and run init again, after setting-up clusterenv.yaml!")
	} else {
		log.Fatal().Err(err).Msgf("Error checking clusterenv file: %s", err)
		return err
	}

	err := helper.CopyDir(helper.BaseCache, helper.ClusterPath+"", false)
	if err != nil {
		log.Error().Msgf("Error: %v", err)
	} else {
		log.Info().Msg("Base files copied successfully.")
	}

	if !clusterEnvPresent {
		os.Exit(0)
	}

	log.Info().Msg("basefiles successfully altered.")
	return nil
}

func UpdateBaseFiles() error {
	log.Info().Msg("Updating Base files for cluster: helper.ClusterPath")
	// Read filenames in source directory
	sourceFiles, err := readFilenamesInDir(helper.BaseCache)
	if err != nil {
		log.Info().Msgf("Error reading source directory: %v\n", err)
		return err
	}

	// Process each file in the target directory
	for _, filename := range sourceFiles {
		sourceFilePath := filepath.Join(helper.BaseCache, filename)
		targetFilePath := filepath.Join(helper.ClusterPath+"", helper.ReplaceDotInFilename(filename))
		helper.ReplaceContentBetweenLines(targetFilePath, sourceFilePath, "## Do not edit between this and DO NOT REMOVE", "## DO NOT REMOVE: Personal setting go under this line")
	}
	log.Info().Msg("basefiles successfully updated.")

	CheckEnvVariables()

	return nil

}

func genRootFiles() error {

	err := helper.CopyDir(helper.RootCache, "./", false)
	if err != nil {
		log.Info().Msgf("Error: %v", err)
	} else {
		log.Info().Msg("Root files copied successfully.")
	}

	agePubKey, err := GetPubKey()
	if err != nil {
		log.Fatal().Err(err).Msg("error: %v")
	}
	log.Info().Msgf("Public Key: %v", agePubKey)
	helper.ReplaceInFile(".sops.yaml", "REPLACEME", agePubKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Error: %s")
	}

	log.Info().Msg("basefiles successfully altered.")
	return nil
}

func UpdateRootFiles() error {
	// Read filenames in source directory
	sourceFiles, err := readFilenamesInDir(helper.RootCache)
	if err != nil {
		log.Info().Msgf("Error reading source directory: %v\n", err)
		return err
	}

	// Process each file in the target directory
	for _, filename := range sourceFiles {
		sourceFilePath := filepath.Join(helper.BaseCache, filename)
		targetFilePath := filepath.Join("./", helper.ReplaceDotInFilename(filename))
		helper.ReplaceContentBetweenLines(targetFilePath, sourceFilePath, "## Do not edit between this and DO NOT REMOVE", "## DO NOT REMOVE: Personal setting go under this line")
	}
	log.Info().Msg("rootfiles successfully updated.")

	agePubKey, err := GetPubKey()
	if err != nil {
		log.Fatal().Err(err).Msg("error: %v")
	}

	helper.ReplaceInFile(".sops.yaml", "REPLACEME", agePubKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Error: %s")
	}

	CheckEnvVariables()

	return nil

}

// Function to read all filenames in a directory
func readFilenamesInDir(dir string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var filenames []string
	for _, file := range files {
		if !file.IsDir() {
			filenames = append(filenames, file.Name())
		}
	}
	return filenames, nil
}

func ResetBootstrapValues() error {
	LoadTalEnv(false)
	err := helper.CopyDirFiltered(helper.KubeCache, helper.ClusterPath+"/kubernetes", true, `^bootstrap-values\.yaml.ct$`)
	if err != nil {
		log.Info().Msg("Error:")
	}

	err2 := helper.EnvSubstRecursive(helper.ClusterPath+"/kubernetes", `^bootstrap-values\.yaml.ct$`, helper.TalEnv)
	if err2 != nil {
		log.Info().Msg("Error:")
	}

	log.Info().Msg("Bootstrap-Values.yaml Files reset successfully.")
	return nil
}

func GenPatches() error {

	err := helper.CopyDir(helper.PatchCache, path.Join(helper.ClusterPath, "/talos/patches"), true)
	if err != nil {
		log.Info().Msg("Error:")
	} else {
		log.Info().Msg("Patch files copied successfully.")
	}

	ageSecKey, err := GetSecKey()
	helper.ReplaceInFile(filepath.Join(helper.ClusterPath+"/talos/patches", "sopssecret.yaml"), "REPLACEWITHSOPS", ageSecKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Error: %s")
	}

	setDocker()

	return nil
}

func setDocker() {
	// Assuming this is part of your function
	patchFilePath := filepath.Join(helper.ClusterPath+"/talos/patches", "all.yaml")
	if helper.TalEnv["DOCKERHUB_USER"] != "" && helper.TalEnv["DOCKERHUB_PASSWORD"] != "" {
		// Prepare the content to append
		configContent := fmt.Sprintf(`
    # Add Dockerhub Login
    config:
      registry-1.docker.io:
        auth:
          username: "%s"
          password: "%s"
      docker.io:
        auth:
          username: "%s"
          password: "%s"`, helper.TalEnv["DOCKERHUB_USER"], helper.TalEnv["DOCKERHUB_PASSWORD"], helper.TalEnv["DOCKERHUB_USER"], helper.TalEnv["DOCKERHUB_PASSWORD"])
		appendContentToPatchFile(patchFilePath, configContent)
	} else {
		// Optional: Append a note if the environment variables are not set
		emptyContent := `# No DockerHub credentials provided
    `
		appendContentToPatchFile(patchFilePath, emptyContent)
	}
}

func appendContentToPatchFile(filePath, content string) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal().Err(err).Msg("Error opening file: %s")
	}
	defer file.Close()

	if _, err := file.Write([]byte(content)); err != nil {
		log.Fatal().Err(err).Msg("Error writing to file: %s")
	}
}
