package version

import (
	"fmt"
	"regexp"
	"time"

	"github.com/Masterminds/semver/v3"
)

// VersionInfo holds the result of sanitising an upstream version string.
type VersionInfo struct {
	// IsValidSemver indicates whether the upstream version could be parsed as semver.
	IsValidSemver bool
	// Semantic is a clean major.minor.patch string. Falls back to calver when the
	// upstream version is not valid semver.
	Semantic string
	// Raw is the upstream version stripped of the leading "v" and any pre-release
	// suffix (everything after the first '-'). When the upstream version is not
	// valid semver the original string is returned unchanged.
	Raw string
	// Upstream is the original, unmodified version string that was passed in.
	Upstream string
}

// strictSemverPrefix matches an optional leading 'v' followed by 1-3 dotted
// numeric groups at the start of the string.
var strictSemverPrefix = regexp.MustCompile(`^v?(\d+(?:\.\d+)?(?:\.\d+)?)`)

// Sanitize takes an upstream version string and returns a VersionInfo with
// normalised semantic and raw representations.
//
// Behaviour mirrors the containerforge app-versions GitHub Action:
//   - Strip "v" prefix and pre-release info to produce the raw version.
//   - When the version matches semver, coerce it to major.minor.patch.
//   - When it does not, fall back to a calendar version (YYYY.M.D).
func Sanitize(upstream string) VersionInfo {
	return SanitizeAt(upstream, time.Now())
}

// SanitizeAt is like Sanitize but accepts an explicit timestamp for the calver
// fallback, making the output deterministic in tests.
func SanitizeAt(upstream string, now time.Time) VersionInfo {
	match := strictSemverPrefix.FindStringSubmatch(upstream)
	isValid := match != nil

	info := VersionInfo{
		IsValidSemver: isValid,
		Upstream:      upstream,
	}

	if isValid {
		coerced, err := semver.NewVersion(match[1])
		if err != nil {
			// Should not happen given the regex, but handle gracefully.
			info.IsValidSemver = false
			info.Semantic = calver(now)
			info.Raw = upstream

			return info
		}

		info.Semantic = fmt.Sprintf("%d.%d.%d", coerced.Major(), coerced.Minor(), coerced.Patch())
		info.Raw = sanitizeRaw(upstream)
	} else {
		info.Semantic = calver(now)
		info.Raw = upstream
	}

	return info
}

// sanitizeRaw strips the leading "v" and everything from the first '-' onward.
func sanitizeRaw(version string) string {
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}

	for i := 0; i < len(version); i++ {
		if version[i] == '-' {
			return version[:i]
		}
	}

	return version
}

// calver returns a calendar-based version string in the form YYYY.M.D.
func calver(t time.Time) string {
	return fmt.Sprintf("%d.%d.%d", t.Year(), int(t.Month()), t.Day())
}
