package image

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog/log"
)

// Images represents the structure of values.yaml.
type Images struct {
	ImagesMap map[string]ImageDetails
	K         *koanf.Koanf
}

// ImageDetails represents details for each image.
type ImageDetails struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
	Version    string
	Link       string
	// Add other fields as needed
}

var imageRegex = regexp.MustCompile(`^image|[a-zA-Z0-9]+Image$`)

func (i *Images) LoadValuesFile(filename string) error {
	// Initialize koanf instance
	i.K = koanf.New(".")

	// Load YAML file using koanf
	if err := i.K.Load(file.Provider(filename), yaml.Parser()); err != nil {
		return err
	}

	// List only root-level keys that match the criteria
	keys := getFilteredRootLevelKeys(i.K)
	i.ImagesMap = make(map[string]ImageDetails)
	for _, key := range keys {
		// Extract relevant fields from the loaded configuration
		var img ImageDetails
		if err := i.K.Unmarshal(key, &img); err != nil {
			return err
		}

		// Set the Link field based on the repository
		img.Link = constructLink(img.Repository)

		// Set the Version field based on the tag
		version, err := CleanTag(img.Tag)
		if err != nil {
			log.Error().Err(err).Msg("❌ Failed to clean tag")
		}

		img.Version = version

		// Save the extracted values to the struct
		i.ImagesMap[key] = img
	}

	return nil
}

func getFilteredRootLevelKeys(k *koanf.Koanf) []string {
	filteredKeys := []string{}

	// k.Raw() returns a map[string]interface{} with all the keys and their values
	// This means the keys will only be the root-level keys, we can drill into the
	// values later if we want the nested keys.
	for key := range k.Raw() {
		if key == "imageSelector" {
			log.Error().Msg("❌ Found [imageSelector] in top level keys, this is not supported.")
			continue
		}
		// Filter keys that match the regex
		if imageRegex.MatchString(key) {
			filteredKeys = append(filteredKeys, key)
		}
	}

	return filteredKeys
}

// constructLink constructs a link based on the repository using the logic from the main function.
func constructLink(repository string) string {
	prefix, repository := resolveLinkPrefix(repository)

	if prefix == "" {
		log.Warn().Msgf("WARNING: Could not determine source repository url for [%s]", repository)
		return ""
	}

	containerURL := fmt.Sprintf("%s%s", prefix, repository)
	return containerURL
}

func resolveLinkPrefix(repository string) (string, string) {
	switch {
	case strings.HasPrefix(repository, "lscr.io/linuxserver/"):
		return "https://fleet.linuxserver.io/image?name=", strings.TrimPrefix(repository, "lscr.io/")
	case strings.HasPrefix(repository, "oci.trueforge.org/containerforge/"):
		return "https://github.com/trueforge-org/containers/tree/main/apps/", strings.TrimPrefix(repository, "oci.trueforge.org/containerforge/")
	case strings.HasPrefix(repository, "mcr.microsoft.com/"):
		return "https://mcr.microsoft.com/en-us/product/", strings.TrimPrefix(repository, "mcr.microsoft.com/")
	case strings.HasPrefix(repository, "public.ecr.aws/"):
		return "https://gallery.ecr.aws/", strings.TrimPrefix(repository, "public.ecr.aws/")
	case strings.HasPrefix(repository, "ghcr.io/"):
		return "https://", repository
	case strings.HasPrefix(repository, "quay.io/"):
		return "https://", repository
	case strings.HasPrefix(repository, "gcr.io/"):
		return "https://", repository
	case strings.Contains(repository, ".azurecr.io/"):
		reg := fmt.Sprintf(`%s.azurecr.io/`, strings.Split(repository, ".")[0])
		return fmt.Sprintf("https://%s", reg), strings.TrimPrefix(repository, reg)
	case strings.Contains(repository, ".ocir.io/"):
		return "", repository
	default:
		return dockerHubLinkParts(repository)
	}
}

func dockerHubLinkParts(repository string) (string, string) {
	prefix := "https://hub.docker.com/r/"
	repository = strings.TrimPrefix(repository, "docker.io/")
	repository = strings.TrimPrefix(repository, "index.docker.io/")
	repository = strings.TrimPrefix(repository, "registry-1.docker.io/")
	repository = strings.TrimPrefix(repository, "registry.hub.docker.com/")

	if strings.Count(repository, "/") == 0 || strings.HasPrefix(repository, "library/") {
		prefix = "https://hub.docker.com/_/"
		repository = strings.TrimPrefix(repository, "library/")
	}

	if strings.Count(repository, "/") > 1 {
		log.Warn().Msgf("WARNING: Could not determine source repository url for [%s]", repository)
		return "", repository
	}

	return prefix, repository
}
