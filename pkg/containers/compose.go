package containers

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	composetypes "github.com/compose-spec/compose-go/v2/types"
)

// BuildComposeYAML assembles a single-service docker-compose document from
// the app settings using the canonical compose-spec types and renders it as
// YAML. The image tag falls back to "rolling" when no usable version is
// provided (empty or "latest"), so the snippet is always runnable.
func BuildComposeYAML(app, version string, settings AppSettings) (string, error) {
	tag := version
	if tag == "" || strings.EqualFold(tag, "latest") {
		tag = "rolling"
	}

	svc := composetypes.ServiceConfig{
		Name:          app,
		Image:         "ghcr.io/trueforge-org/" + app + ":" + tag,
		ContainerName: app,
		Restart:       "unless-stopped",
	}

	for _, p := range settings.Ports {
		proto := strings.ToLower(p.Protocol)
		if proto == "" {
			proto = "tcp"
		}
		svc.Ports = append(svc.Ports, composetypes.ServicePortConfig{
			Target:    uint32(p.Port),
			Published: strconv.Itoa(p.Port),
			Protocol:  proto,
		})
	}

	if len(settings.Env) > 0 {
		env := make(composetypes.MappingWithEquals, len(settings.Env))
		for _, e := range settings.Env {
			val := e.Default
			env[e.Name] = &val
		}
		svc.Environment = env
	}

	for _, v := range settings.Volumes {
		hostName := filepath.Base(v.Path)
		svc.Volumes = append(svc.Volumes, composetypes.ServiceVolumeConfig{
			Type:   composetypes.VolumeTypeBind,
			Source: "./" + hostName,
			Target: v.Path,
		})
	}

	project := &composetypes.Project{
		Services: composetypes.Services{app: svc},
	}
	out, err := project.MarshalYAML()
	if err != nil {
		return "", fmt.Errorf("marshal compose: %w", err)
	}
	return string(out), nil
}
