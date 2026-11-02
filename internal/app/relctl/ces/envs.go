package ces

import (
	"os"
	"strings"

	"github.com/layer87-labs/relctl/internal/pkg/envvars"

	log "github.com/sirupsen/logrus"
)

// LegacyEnvVar enables the export of ACI_* aliases next to the RELCTL_*
// variables. It exists to ease the migration from awesome-ci.
const LegacyEnvVar = "RELCTL_LEGACY_ACI_ENV"

// withLegacyAliases appends an ACI_* alias for every RELCTL_* variable.
func withLegacyAliases(envVars []KeyValue) []KeyValue {
	out := make([]KeyValue, 0, len(envVars)*2)
	out = append(out, envVars...)
	for _, ev := range envVars {
		if strings.HasPrefix(ev.Name, "RELCTL_") {
			out = append(out, KeyValue{Name: "ACI_" + strings.TrimPrefix(ev.Name, "RELCTL_"), Value: ev.Value})
		}
	}
	return out
}

func legacyEnabled() bool {
	v := strings.ToLower(os.Getenv(LegacyEnvVar))
	return v == "true" || v == "1"
}

func (ces CES) ExportAsEnv(envVars []KeyValue) (err error) {
	if legacyEnabled() {
		envVars = withLegacyAliases(envVars)
	}
	envFile, err := envvars.OpenEnvFile(ces.EnvFile)
	if err != nil {
		return
	}

	if ces.OutFile != nil {
		outFile, err := envvars.OpenEnvFile(ces.EnvFile)
		if err != nil {
			return err
		}
		for _, ev := range envVars {
			envFile.Set(ev.Name, ev.Value)
			outFile.Set(ev.Name, ev.Value)
		}

		if err := envFile.CloseEnvFile(ces.EnvFile); err != nil {
			log.Warnln(err)
			return err
		}
		log.Tracef("succfully written env variables to %s", ces.EnvFile)

		if err := outFile.CloseEnvFile(*ces.OutFile); err != nil {
			log.Warnln(err)
			return err
		}
		log.Tracef("succfully written env variables to %s", *ces.OutFile)

	} else {

		for _, ev := range envVars {
			envFile.Set(ev.Name, ev.Value)
		}
		if err := envFile.CloseEnvFile(ces.EnvFile); err != nil {
			log.Warnln(err)
			return err
		}
	}

	return
}
