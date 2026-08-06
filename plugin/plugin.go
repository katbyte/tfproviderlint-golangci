// Package plugin registers tfproviderlint's analyzers as a golangci-lint module plugin,
// so they can run inside a custom golangci-lint binary (sharing one package-load, config
// and report stream) instead of via the standalone tfproviderlint/tfproviderlintx binaries.
package plugin

import (
	"fmt"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/bflad/tfproviderlint/passes"
	"github.com/bflad/tfproviderlint/xpasses"
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("tfproviderlint", New)

	version := tfproviderlintVersion()
	for _, a := range passes.AllChecks {
		decorate(a, "passes", version)
	}
	for _, a := range xpasses.AllChecks {
		decorate(a, "xpasses", version)
	}
}

func decorate(a *analysis.Analyzer, dir, version string) {
	stripSelfPrefix(a)

	// upstream sets no URL but documents every check; link the exact wrapped version
	if a.URL == "" {
		a.URL = fmt.Sprintf("https://github.com/bflad/tfproviderlint/blob/%s/%s/%s/README.md", version, dir, a.Name)
	}
}

// tfproviderlintVersion returns the wrapped tfproviderlint module version from build info,
// so doc URLs always match the go.mod pin.
func tfproviderlintVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/bflad/tfproviderlint" {
				return dep.Version
			}
		}
	}
	return "main"
}

// stripSelfPrefix rewrites diagnostics as they are reported: tfproviderlint embeds the check
// name in its messages ("S006: schema ...") and golangci-lint prefixes the analyzer name
// again ("S006: S006: schema ..."), so drop the embedded copy.
func stripSelfPrefix(a *analysis.Analyzer) {
	run := a.Run
	prefix := a.Name + ": "
	a.Run = func(pass *analysis.Pass) (any, error) {
		report := pass.Report
		pass.Report = func(d analysis.Diagnostic) {
			d.Message = strings.TrimPrefix(d.Message, prefix)
			report(d)
		}
		return run(pass)
	}
}

// Settings configures the plugin from .golangci.yml via
// linters.settings.custom.tfproviderlint.settings:
//
//	settings:
//	  extended: false                 # also include the tfproviderlintx (X*) checks
//	  enable: [AT001, R001]           # empty means all checks
//	  disable: [S001]
//	  flags:                          # per-check flags, named as on the tfproviderlint CLI
//	    - check: AT001
//	      flag: ignored-filename-suffixes
//	      value: _data_source_test.go
//
// Flags are a list rather than a "AT001.ignored-filename-suffixes" style map because
// golangci-lint's config loader lowercases map keys and treats dots as nesting.
type Settings struct {
	Enable   []string `json:"enable"`
	Disable  []string `json:"disable"`
	Extended bool     `json:"extended"`
	Flags    []Flag   `json:"flags"`
}

// Flag sets one flag on one check, e.g. tfproviderlint's -AT001.ignored-filename-suffixes
// is {Check: AT001, Flag: ignored-filename-suffixes, Value: ...}.
type Flag struct {
	Check string `json:"check"`
	Flag  string `json:"flag"`
	Value any    `json:"value"`
}

func New(settings any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	return &Plugin{settings: s}, nil
}

type Plugin struct {
	settings Settings
}

func (p *Plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	all := passes.AllChecks
	if p.settings.Extended {
		all = slices.Concat(passes.AllChecks, xpasses.AllChecks)
	}

	known := make(map[string]*analysis.Analyzer, len(all))
	for _, a := range all {
		known[a.Name] = a
	}
	for _, name := range slices.Concat(p.settings.Enable, p.settings.Disable) {
		if known[name] == nil {
			return nil, fmt.Errorf("unknown tfproviderlint check %q in settings", name)
		}
	}

	// Per-check flags — applied before filtering so a flag on a disabled check is not an
	// error, but a typo in the check or flag name is.
	for _, f := range p.settings.Flags {
		a := known[f.Check]
		if a == nil {
			return nil, fmt.Errorf("unknown tfproviderlint check %q in flags settings", f.Check)
		}
		if a.Flags.Lookup(f.Flag) == nil {
			return nil, fmt.Errorf("unknown flag %q for tfproviderlint check %q", f.Flag, f.Check)
		}
		if err := a.Flags.Set(f.Flag, fmt.Sprint(f.Value)); err != nil {
			return nil, fmt.Errorf("setting tfproviderlint flag %s.%s: %w", f.Check, f.Flag, err)
		}
	}

	analyzers := make([]*analysis.Analyzer, 0, len(all))
	for _, a := range all {
		if len(p.settings.Enable) > 0 && !slices.Contains(p.settings.Enable, a.Name) {
			continue
		}
		if slices.Contains(p.settings.Disable, a.Name) {
			continue
		}
		analyzers = append(analyzers, a)
	}

	return analyzers, nil
}

func (p *Plugin) GetLoadMode() string {
	// tfproviderlint checks inspect schema/type information
	return register.LoadModeTypesInfo
}
