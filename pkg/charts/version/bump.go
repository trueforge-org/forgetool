package version

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/rs/zerolog/log"
)

const (
	Major = "major"
	Minor = "minor"
	Patch = "patch"
)

var patchAliases = map[string]struct{}{
	Patch:       {},
	"pindigest": {},
	"digest":    {},
	"pin":       {},
	"lockfile":  {},
}

func normalizeBumpKind(kind string) string {
	normalized := strings.ToLower(strings.TrimSpace(kind))

	if _, ok := patchAliases[normalized]; ok {
		return Patch
	}

	return normalized
}

func IncrementVersion(version, kind string) (string, error) {
	kind = normalizeBumpKind(kind)

	// Validate SemVer format
	semVerPattern := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
	if !semVerPattern.MatchString(version) {
		return "", fmt.Errorf("invalid SemVer format (Major.Minor.Patch): %s", version)
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}

	// Increment the specified version component
	switch kind {
	case Major:
		return v.IncMajor().String(), nil
	case Minor:
		return v.IncMinor().String(), nil
	case Patch:
		return v.IncPatch().String(), nil
	default:
		return "", fmt.Errorf("invalid bump kind: %s", kind)
	}
}

func Bump(semVer, kind string) error {

	newVersion, err := IncrementVersion(semVer, kind)
	if err != nil {
		return err
	}

	log.Info().Msgf("🆚 Updated SemVer from [%s] to [%s]", semVer, newVersion)
	return nil
}
