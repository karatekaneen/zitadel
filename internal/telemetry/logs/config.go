package logs

import (
	"fmt"
	"github.com/zitadel/zitadel/internal/telemetry/logs/record"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter"
	"github.com/sirupsen/logrus"
	"github.com/zitadel/logging"
	"github.com/zitadel/logging/otel"
	"go.opentelemetry.io/collector/exporter"
)

type Hook string

const (
	GCPLoggingOtelExporter Hook = "GCPLoggingOtelExporter"
)

type Config struct {
	Log   logging.Config `mapstructure:",squash"`
	Hooks map[string]map[string]interface{}
}

func (c *Config) SetLogger() (err error) {
	var hooks []logrus.Hook
	for name, rawCfg := range c.Hooks {
		switch name {
		case strings.ToLower(string(GCPLoggingOtelExporter)):
			var hook *otel.GcpLoggingExporterHook
			hook, err = otel.NewGCPLoggingExporterHook(
				otel.WithChangedDefaultExporterConfig(func(cfg *googlecloudexporter.Config) {
					cfg.LogConfig.DefaultLogName = "zitadel"
					err = decodeRawConfig(rawCfg, cfg)
				}),
				otel.WithChangedDefaultOtelSettings(func(cfg *exporter.Settings) {
					err = decodeRawConfig(rawCfg, cfg)
				}),
				otel.WithInclude(func(entry *logrus.Entry) bool {
					return entry.Data["stream"] == record.StreamActivity
				}),
				otel.WithChangedLevels([]logrus.Level{logrus.InfoLevel}),
			)
			if err != nil {
				return err
			}
			if err = hook.Start(); err != nil {
				return err
			}
			hooks = append(hooks, hook)
		default:
			return fmt.Errorf("unknown hook: %s", name)
		}
	}
	return c.Log.SetLogger(
		logging.AddHooks(hooks...),
	)
}

func decodeRawConfig(rawConfig map[string]interface{}, typedConfig any) (err error) {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		MatchName: func(mapKey, fieldName string) bool {
			return strings.ToLower(mapKey) == strings.ToLower(fieldName)
		},
		WeaklyTypedInput: true,
		Result:           typedConfig,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(rawConfig)
}
