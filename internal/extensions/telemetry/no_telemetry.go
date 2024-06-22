//go:build no_telemetry

package telemetry

import "github.com/dgate-io/dgate/internal/config"

func SetupTelemetry(conf *config.DGateConfig) func() {
	return func() {}
}
