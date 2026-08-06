package plugin

import (
	"strings"
	"testing"

	"github.com/bflad/tfproviderlint/passes"
	"github.com/bflad/tfproviderlint/xpasses"
	"github.com/golangci/plugin-module-register/register"
)

func buildWith(t *testing.T, constructor func(any) (register.LinterPlugin, error), settings any) ([]string, error) {
	t.Helper()

	p, err := constructor(settings)
	if err != nil {
		return nil, err
	}

	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(analyzers))
	for _, a := range analyzers {
		names = append(names, a.Name)
	}
	return names, nil
}

func buildAnalyzers(t *testing.T, settings any) ([]string, error) {
	t.Helper()
	return buildWith(t, New, settings)
}

func TestBuildAnalyzersDefault(t *testing.T) {
	names, err := buildAnalyzers(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != len(passes.AllChecks) {
		t.Fatalf("expected all %d checks with no settings, got %d", len(passes.AllChecks), len(names))
	}
}

func TestBuildAnalyzersExtended(t *testing.T) {
	names, err := buildWith(t, NewX, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := len(passes.AllChecks) + len(xpasses.AllChecks)
	if len(names) != want {
		t.Fatalf("expected %d checks from tfproviderlintx, got %d", want, len(names))
	}
}

func TestBuildAnalyzersEnable(t *testing.T) {
	names, err := buildAnalyzers(t, map[string]any{"enable": []string{"AT001", "R001"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 checks, got %d: %v", len(names), names)
	}
}

func TestBuildAnalyzersDisable(t *testing.T) {
	names, err := buildAnalyzers(t, map[string]any{"disable": []string{"AT001"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != len(passes.AllChecks)-1 {
		t.Fatalf("expected %d checks, got %d", len(passes.AllChecks)-1, len(names))
	}
	for _, name := range names {
		if name == "AT001" {
			t.Fatal("AT001 should have been disabled")
		}
	}
}

func TestAnalyzerURLs(t *testing.T) {
	for _, a := range passes.AllChecks {
		if !strings.Contains(a.URL, "bflad/tfproviderlint") || !strings.HasSuffix(a.URL, "/passes/"+a.Name+"/README.md") {
			t.Fatalf("%s URL = %q, want a passes/%s/README.md link", a.Name, a.URL, a.Name)
		}
	}
	for _, a := range xpasses.AllChecks {
		if !strings.HasSuffix(a.URL, "/xpasses/"+a.Name+"/README.md") {
			t.Fatalf("%s URL = %q, want an xpasses/%s/README.md link", a.Name, a.URL, a.Name)
		}
	}
}

func TestBuildAnalyzersUnknownCheck(t *testing.T) {
	if _, err := buildAnalyzers(t, map[string]any{"enable": []string{"NOPE001"}}); err == nil {
		t.Fatal("expected an error for an unknown check name")
	}
}

func TestBuildAnalyzersExtendedCheckRequiresX(t *testing.T) {
	if _, err := buildWith(t, New, map[string]any{"enable": []string{"XR001"}}); err == nil {
		t.Fatal("expected an error enabling an X check on the tfproviderlint plugin")
	}
	if _, err := buildWith(t, NewX, map[string]any{"enable": []string{"XR001"}}); err != nil {
		t.Fatalf("expected XR001 to be known to the tfproviderlintx plugin, got %v", err)
	}
}

func TestBuildAnalyzersFlags(t *testing.T) {
	settings := map[string]any{
		"flags": []map[string]any{
			{"check": "AT001", "flag": "ignored-filename-suffixes", "value": "_data_source_test.go"},
			{"check": "R006", "flag": "package-aliases", "value": "pluginsdk"},
		},
	}
	if _, err := buildAnalyzers(t, settings); err != nil {
		t.Fatal(err)
	}

	for _, a := range passes.AllChecks {
		if a.Name == "AT001" {
			if got := a.Flags.Lookup("ignored-filename-suffixes").Value.String(); got != "_data_source_test.go" {
				t.Fatalf("AT001 ignored-filename-suffixes = %q, want _data_source_test.go", got)
			}
		}
	}
}

func TestBuildAnalyzersUnknownFlag(t *testing.T) {
	if _, err := buildAnalyzers(t, map[string]any{"flags": []map[string]any{{"check": "AT001", "flag": "nope", "value": "x"}}}); err == nil {
		t.Fatal("expected an error for an unknown flag name")
	}
	if _, err := buildAnalyzers(t, map[string]any{"flags": []map[string]any{{"check": "NOPE001", "flag": "flag", "value": "x"}}}); err == nil {
		t.Fatal("expected an error for an unknown check in a flag")
	}
}
