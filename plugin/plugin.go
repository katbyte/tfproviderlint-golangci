// Package plugin registers tfproviderlint's analyzers as a golangci-lint module plugin,
// so they can run inside a custom golangci-lint binary (sharing one package-load, config
// and report stream) instead of via the standalone tfproviderlint/tfproviderlintx binaries.
package plugin

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bflad/tfproviderlint/passes"
	"github.com/bflad/tfproviderlint/xpasses"
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("tfproviderlint", New)
}

// Settings configures the plugin from .golangci.yml via
// linters.settings.custom.tfproviderlint.settings:
//
//	settings:
//	  extended: false                 # also include the tfproviderlintx (X*) checks
//	  enable: [AT001, R001]           # empty means all checks
//	  disable: [S001]
//	  flags:                          # per-check flags, named as on the tfproviderlint CLI
//	    AT001.ignored-filename-suffixes: _data_source_test.go
//	    R006.package-aliases: pluginsdk
type Settings struct {
	Enable   []string       `json:"enable"`
	Disable  []string       `json:"disable"`
	Extended bool           `json:"extended"`
	Flags    map[string]any `json:"flags"`
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

	// Per-check flags, e.g. "AT001.ignored-filename-suffixes" — applied before filtering so a
	// flag on a disabled check is not an error, but a typo in the check or flag name is.
	for key, value := range p.settings.Flags {
		name, flagName, ok := strings.Cut(key, ".")
		if !ok {
			return nil, fmt.Errorf("invalid tfproviderlint flag %q in settings, expected <check>.<flag>", key)
		}
		a := known[name]
		if a == nil {
			return nil, fmt.Errorf("unknown tfproviderlint check %q in flag %q", name, key)
		}
		if a.Flags.Lookup(flagName) == nil {
			return nil, fmt.Errorf("unknown flag %q for tfproviderlint check %q", flagName, name)
		}
		if err := a.Flags.Set(flagName, fmt.Sprint(value)); err != nil {
			return nil, fmt.Errorf("setting tfproviderlint flag %q: %w", key, err)
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
