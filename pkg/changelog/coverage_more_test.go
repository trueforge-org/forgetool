package changelog

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

func TestChangedData_UnmarshalJSON_LegacyMerge(t *testing.T) {
	// Case: only legacy "charts" present.
	cd := &ChangedData{mu: &sync.RWMutex{}}
	if err := cd.UnmarshalJSON([]byte(`{"charts":{"a":{"versions":{}}}}`)); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, ok := cd.Apps["a"]; !ok {
		t.Fatalf("expected legacy charts merged into Apps")
	}

	// Case: both apps and charts present, charts merged for missing keys only.
	cd2 := &ChangedData{mu: &sync.RWMutex{}, Apps: map[string]*App{"a": {}}}
	raw := []byte(`{"apps":{"a":{"versions":{}}},"charts":{"a":{"versions":{}},"b":{"versions":{}}}}`)
	if err := cd2.UnmarshalJSON(raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, ok := cd2.Apps["b"]; !ok {
		t.Fatalf("expected legacy entry b merged")
	}
}

func TestChangedData_UnmarshalJSON_BadJSON(t *testing.T) {
	cd := &ChangedData{}
	if err := cd.UnmarshalJSON([]byte(`{"apps":[]}`)); err == nil {
		t.Fatalf("expected unmarshal error")
	}
	// Sanity for json package.
	_ = json.Unmarshal
}

func TestRender_ConfigureForAppTypeError(t *testing.T) {
	o := &ChangelogOptions{AppType: AppType("not-a-real-type")}
	if err := o.Render(); err == nil {
		t.Fatalf("expected configureForAppType error")
	}
}

func TestGenerate_ConfigureForAppTypeError(t *testing.T) {
	o := &ChangelogOptions{AppType: AppType("not-a-real-type")}
	if err := o.Generate(); err == nil {
		t.Fatalf("expected configureForAppType error")
	}
}

// Sentinel to catch accidental imports being trimmed.
var _ = errors.New
