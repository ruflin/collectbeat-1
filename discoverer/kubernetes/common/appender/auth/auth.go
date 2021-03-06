package auth

import (
	"fmt"
	"io/ioutil"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/appender"
	"github.com/ebay/collectbeat/discoverer/common/registry"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

const (
	Auth = "auth"
)

func init() {
	registry.BuilderRegistry.AddAppender(Auth, NewSecurityAppender)

	cfg := common.NewConfig()
	// Register default builders
	registry.BuilderRegistry.AddDefaultAppenderConfig(Auth, *cfg)
}

type SecurityAppender struct {
	Namespaces []string
	TokenPath  string
}

func NewSecurityAppender(cfg *common.Config) (appender.Appender, error) {

	config := struct {
		Namespaces []string `config:"namespaces"`
		TokenPath  string   `config:"token_path"`
	}{
		Namespaces: []string{"apiserver", "scheduler", "controller_manager"},
		TokenPath:  "/var/run/secrets/kubernetes.io/serviceaccount/token",
	}

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	return &SecurityAppender{
		Namespaces: config.Namespaces,
		TokenPath:  config.TokenPath,
	}, nil
}

func (i *SecurityAppender) Append(configHolder *dcommon.ConfigHolder) {
	config := configHolder.Config
	if config == nil {
		return
	}
	moduleConfig := mb.ModuleConfig{}
	config.Unpack(&moduleConfig)

	if moduleConfig.Module == "prometheus" {
		namespace, err := config.String("namespace", -1)
		if err == nil {

			for _, ns := range i.Namespaces {
				if ns == namespace {
					token := i.getAuthHeaderFromToken()
					if token != "" {
						headers := map[string]interface{}{}
						headers["Authorization"] = token

						hconf, err := common.NewConfigFrom(headers)
						if err == nil {
							config.SetChild("headers", -1, hconf)
						}

					}
					return
				}
			}
		}
	}
}

func (i *SecurityAppender) getAuthHeaderFromToken() string {
	var token string

	if i.TokenPath != "" {
		b, err := ioutil.ReadFile(i.TokenPath)
		if err != nil {
			logp.Err("Reading token file failed with err: %v", err)
		}

		if len(b) != 0 {
			if b[len(b)-1] == '\n' {
				b = b[0 : len(b)-1]
			}
			token = fmt.Sprintf("Bearer %s", string(b))
		}
	}

	return token
}
