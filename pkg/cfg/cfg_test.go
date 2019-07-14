package cfg

import (
	"testing"

	// "github.com/rdeusser/oauth2-proxy/pkg/structs"
	"github.com/stretchr/testify/assert"
)

func init() {
	// log.SetLevel(log.DebugLevel)
	InitForTestPurposes()
}

func TestConfigParsing(t *testing.T) {

	// UnmarshalKey(Branding.LCName, &cfg)
	log.Debugf("cfgPort %d", Cfg.Port)
	log.Debugf("cfgDomains %s", Cfg.Domains[0])

	assert.Equal(t, Cfg.Port, 9090)

	assert.NotEmpty(t, Cfg.JWT.MaxAge)

}
