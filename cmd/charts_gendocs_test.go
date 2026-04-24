package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/trueforge-org/forgetool/pkg/website"
)

func setupChartsGenDocsStubs(t *testing.T) {
	t.Helper()
	oldPrep := chartsGenDocsPrepareWebsite
	oldDisc := chartsGenDocsDiscoverCharts
	oldProc := chartsGenDocsProcessChart
	oldFin := chartsGenDocsFinalizeWebsite
	t.Cleanup(func() {
		chartsGenDocsPrepareWebsite = oldPrep
		chartsGenDocsDiscoverCharts = oldDisc
		chartsGenDocsProcessChart = oldProc
		chartsGenDocsFinalizeWebsite = oldFin
	})
	chartsGenDocsPrepareWebsite = func(website.ChartOptions) error { return nil }
	chartsGenDocsDiscoverCharts = func(string) ([][2]string, error) { return nil, nil }
	chartsGenDocsProcessChart = func(website.ChartOptions) error { return nil }
	chartsGenDocsFinalizeWebsite = func(website.ChartOptions, string) error { return nil }
}

func TestRunChartsGenDocs_PrepareError(t *testing.T) {
	setupChartsGenDocsStubs(t)
	chartsGenDocsPrepareWebsite = func(website.ChartOptions) error { return errors.New("boom") }
	if err := runChartsGenDocs(nil); err == nil || !strings.Contains(err.Error(), "prepare website") {
		t.Fatalf("expected prepare error, got %v", err)
	}
}

func TestRunChartsGenDocs_DiscoverError(t *testing.T) {
	setupChartsGenDocsStubs(t)
	chartsGenDocsDiscoverCharts = func(string) ([][2]string, error) { return nil, errors.New("nope") }
	if err := runChartsGenDocs(nil); err == nil || !strings.Contains(err.Error(), "discover charts") {
		t.Fatalf("expected discover error, got %v", err)
	}
}

func TestRunChartsGenDocs_DiscoverSuccess(t *testing.T) {
	setupChartsGenDocsStubs(t)
	chartsGenDocsDiscoverCharts = func(string) ([][2]string, error) {
		return [][2]string{{"stable", "myapp"}}, nil
	}
	calls := 0
	chartsGenDocsProcessChart = func(opts website.ChartOptions) error {
		calls++
		if opts.Train != "stable" || opts.Chart != "myapp" {
			t.Fatalf("unexpected opts: %#v", opts)
		}
		return nil
	}
	if err := runChartsGenDocs(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected ProcessChart called once, got %d", calls)
	}
}

func TestRunChartsGenDocs_InvalidIdentifier(t *testing.T) {
	setupChartsGenDocsStubs(t)
	if err := runChartsGenDocs([]string{"bogus"}); err == nil || !strings.Contains(err.Error(), "invalid chart identifier") {
		t.Fatalf("expected invalid identifier error, got %v", err)
	}
	if err := runChartsGenDocs([]string{"/missing"}); err == nil {
		t.Fatalf("expected error for empty train")
	}
	if err := runChartsGenDocs([]string{"train/"}); err == nil {
		t.Fatalf("expected error for empty chart")
	}
}

func TestRunChartsGenDocs_ProcessError(t *testing.T) {
	setupChartsGenDocsStubs(t)
	chartsGenDocsProcessChart = func(website.ChartOptions) error { return errors.New("fail") }
	if err := runChartsGenDocs([]string{"stable/app"}); err == nil || !strings.Contains(err.Error(), "process stable/app") {
		t.Fatalf("expected process error, got %v", err)
	}
}

func TestRunChartsGenDocs_FinalizeError(t *testing.T) {
	setupChartsGenDocsStubs(t)
	chartsGenDocsFinalizeWebsite = func(website.ChartOptions, string) error { return errors.New("end") }
	if err := runChartsGenDocs([]string{"stable/app"}); err == nil || !strings.Contains(err.Error(), "finalize website") {
		t.Fatalf("expected finalize error, got %v", err)
	}
}

func TestRunChartsGenDocs_HappyPathExplicit(t *testing.T) {
	setupChartsGenDocsStubs(t)
	if err := runChartsGenDocs([]string{"stable/app", "incubator/other"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChartsGenDocsCmdRunCallsRunner(t *testing.T) {
	oldRunner := chartsGenDocsRunner
	t.Cleanup(func() { chartsGenDocsRunner = oldRunner })
	called := false
	chartsGenDocsRunner = func([]string) error { called = true; return nil }
	chartsGenDocsCmd.Run(chartsGenDocsCmd, []string{"stable/app"})
	if !called {
		t.Fatalf("expected runner to be called")
	}
}

func TestChartsGenDocsCmdRunCallsOnError(t *testing.T) {
	oldRunner := chartsGenDocsRunner
	oldOnError := chartsGenDocsOnError
	t.Cleanup(func() {
		chartsGenDocsRunner = oldRunner
		chartsGenDocsOnError = oldOnError
	})
	want := errors.New("boom")
	chartsGenDocsRunner = func([]string) error { return want }
	called := false
	chartsGenDocsOnError = func(err error) {
		called = true
		if !errors.Is(err, want) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	chartsGenDocsCmd.Run(chartsGenDocsCmd, nil)
	if !called {
		t.Fatalf("expected onError to be called")
	}
}
