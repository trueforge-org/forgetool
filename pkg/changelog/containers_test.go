package changelog

import (
	"testing"
)

func TestContainerGetAppName(t *testing.T) {
	got := containerGetAppName("apps/mycontainer/docker-bake.hcl")
	if got != "mycontainer" {
		t.Fatalf("expected mycontainer, got %s", got)
	}

	got = containerGetAppName("apps/mycontainer/subdir/file.txt")
	if got != "mycontainer" {
		t.Fatalf("expected mycontainer, got %s", got)
	}

	got = containerGetAppName("onlyone")
	if got != invalidName {
		t.Fatalf("expected invalidName for single-segment path, got %s", got)
	}
}

func TestContainerGetAppTrain(t *testing.T) {
	got := containerGetAppTrain("apps/mycontainer/docker-bake.hcl")
	if got != "" {
		t.Fatalf("expected empty train for containers, got %s", got)
	}
}

func TestContainerGetAppPath(t *testing.T) {
	p, err := containerGetAppPath("apps/mycontainer/docker-bake.hcl")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "apps/mycontainer/docker-bake.hcl"
	if p != expected {
		t.Fatalf("expected %s, got %s", expected, p)
	}

	p, err = containerGetAppPath("apps/mycontainer/subdir/file.txt")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p != expected {
		t.Fatalf("expected %s, got %s", expected, p)
	}

	_, err = containerGetAppPath(".")
	if err == nil {
		t.Fatalf("expected error for short path")
	}
}

func TestContainerGetVersion(t *testing.T) {
	content := "variable \"APP\" {\n  default = \"myapp\"\n}\nvariable \"VERSION\" {\n  default = \"1.2.3\"\n}\n"
	v, err := containerGetVersion(content)
	if err != nil {
		t.Fatalf("containerGetVersion error: %v", err)
	}
	if v != "1.2.3" {
		t.Fatalf("expected 1.2.3, got %s", v)
	}

	// With comment (like renovate)
	content = "variable \"VERSION\" {\n  // renovate: datasource=github-releases\n  default = \"2.0.0\"\n}\n"
	v, err = containerGetVersion(content)
	if err != nil {
		t.Fatalf("containerGetVersion error: %v", err)
	}
	if v != "2.0.0" {
		t.Fatalf("expected 2.0.0, got %s", v)
	}

	// With v prefix — should be stripped
	content = "variable \"VERSION\" {\n  default = \"v3.1.0\"\n}\n"
	v, err = containerGetVersion(content)
	if err != nil {
		t.Fatalf("containerGetVersion error: %v", err)
	}
	if v != "3.1.0" {
		t.Fatalf("expected 3.1.0, got %s", v)
	}

	// With pre-release suffix — should be stripped
	content = "variable \"VERSION\" {\n  default = \"4.0.0-rc1\"\n}\n"
	v, err = containerGetVersion(content)
	if err != nil {
		t.Fatalf("containerGetVersion error: %v", err)
	}
	if v != "4.0.0" {
		t.Fatalf("expected 4.0.0, got %s", v)
	}

	// Partial version — should be coerced to full semver
	content = "variable \"VERSION\" {\n  default = \"5.1\"\n}\n"
	v, err = containerGetVersion(content)
	if err != nil {
		t.Fatalf("containerGetVersion error: %v", err)
	}
	if v != "5.1.0" {
		t.Fatalf("expected 5.1.0, got %s", v)
	}

	// Missing VERSION
	content = "variable \"APP\" {\n  default = \"myapp\"\n}\n"
	_, err = containerGetVersion(content)
	if err == nil {
		t.Fatalf("expected error when VERSION missing")
	}

	// VERSION block closes before default
	content = "variable \"VERSION\" {\n}\n"
	_, err = containerGetVersion(content)
	if err == nil {
		t.Fatalf("expected error when VERSION has no default")
	}
}

func TestContainerIsPreferredFile(t *testing.T) {
	if !containerIsPreferredFile("apps/mycontainer/docker-bake.hcl") {
		t.Fatalf("expected docker-bake.hcl to be preferred")
	}
	if containerIsPreferredFile("apps/mycontainer/values.yaml") {
		t.Fatalf("expected values.yaml to not be preferred")
	}
}

func TestContainerRenderOutputPath(t *testing.T) {
	got := containerRenderOutputPath("/output", "ignored", "mycontainer")
	if got != "/output/mycontainer" {
		t.Fatalf("expected /output/mycontainer, got %s", got)
	}
}

func TestContainerParseActiveApp(t *testing.T) {
	a := &ActiveApps{items: make(map[string]ActiveApp), mu: newRWMutex()}

	if err := containerParseActiveApp(a, "apps/mycontainer/docker-bake.hcl"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !a.isActiveApp("mycontainer") {
		t.Fatalf("expected mycontainer to be active")
	}
	app := a.items["mycontainer"]
	if app.Train != "" {
		t.Fatalf("expected empty train for container, got %s", app.Train)
	}

	// Duplicate
	if err := containerParseActiveApp(a, "apps/mycontainer/docker-bake.hcl"); err != nil {
		t.Fatalf("expected no error on duplicate, got %v", err)
	}
	if len(a.items) != 1 {
		t.Fatalf("expected 1 item after duplicate, got %d", len(a.items))
	}

	// Too short
	if err := containerParseActiveApp(a, "docker-bake.hcl"); err == nil {
		t.Fatalf("expected error for single-segment path")
	}
}
