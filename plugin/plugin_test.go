package plugin

import (
	"testing"

	"github.com/bflad/tfproviderlint/passes"
	"github.com/bflad/tfproviderlint/xpasses"
)

func buildAnalyzers(t *testing.T, settings any) ([]string, error) {
	t.Helper()

	p, err := New(settings)
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
	names, err := buildAnalyzers(t, map[string]any{"extended": true})
	if err != nil {
		t.Fatal(err)
	}
	want := len(passes.AllChecks) + len(xpasses.AllChecks)
	if len(names) != want {
		t.Fatalf("expected %d checks with extended, got %d", want, len(names))
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

func TestBuildAnalyzersUnknownCheck(t *testing.T) {
	if _, err := buildAnalyzers(t, map[string]any{"enable": []string{"NOPE001"}}); err == nil {
		t.Fatal("expected an error for an unknown check name")
	}
}

func TestBuildAnalyzersExtendedCheckRequiresExtended(t *testing.T) {
	if _, err := buildAnalyzers(t, map[string]any{"enable": []string{"XR001"}}); err == nil {
		t.Fatal("expected an error enabling an X check without extended: true")
	}
	if _, err := buildAnalyzers(t, map[string]any{"extended": true, "enable": []string{"XR001"}}); err != nil {
		t.Fatalf("expected XR001 to be known with extended: true, got %v", err)
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
