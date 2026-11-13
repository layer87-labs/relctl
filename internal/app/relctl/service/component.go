package service

import (
	"fmt"

	scmportal "github.com/layer87-labs/relctl/internal/app/relctl/scm-portal"
	"github.com/layer87-labs/relctl/internal/pkg/relctlconfig"
)

// loadEffective reads the config and resolves the settings of the repository or of
// one of its components ("" = the repository).
func loadEffective(configFile, component string) (*relctlconfig.Effective, error) {
	if configFile == "" {
		configFile = relctlconfig.DefaultConfigFile
	}
	cfg, err := relctlconfig.Load(configFile)
	if err != nil {
		return nil, fmt.Errorf("loading relctl config: %w", err)
	}
	return cfg.Effective(component)
}

// applyNamespace sets the tag prefix of the repository or component on the layer, so
// that everything below works with plain versions. A component is released with the
// calver or changelog scheme: its version must not depend on the other components.
func applyNamespace(scmLayer *scmportal.SCMLayer, configFile, component, schemeOverride string) (*relctlconfig.Effective, error) {
	eff, err := loadEffective(configFile, component)
	if err != nil {
		return nil, err
	}
	if component != "" {
		scheme := eff.VersionScheme
		if schemeOverride != "" {
			scheme = schemeOverride
		}
		if scheme != relctlconfig.SchemeCalVer && scheme != relctlconfig.SchemeChangelog {
			return nil, fmt.Errorf("component %q uses the version scheme %q: components are released with %q or %q",
				component, scheme, relctlconfig.SchemeCalVer, relctlconfig.SchemeChangelog)
		}
	}
	scmLayer.TagPrefix = eff.TagPrefix
	scmLayer.BreakingMarkers = eff.BreakingMarkers
	return eff, nil
}

// releaseVars are the version variables `release create` exports.
type releaseVars struct {
	// Version is what was created: X.Y.Z, or X.Y.Z-rc.N for a release candidate.
	Version string
	// Next is the version of the next release, X.Y.Z. For a candidate this is the
	// release it is a candidate for.
	Next string
	// Latest is the latest published release before the call; empty if there is none.
	Latest string
	// TagPrefix and Component name the part of the repository that is released.
	TagPrefix  string
	Component  string
	Prerelease bool
}
