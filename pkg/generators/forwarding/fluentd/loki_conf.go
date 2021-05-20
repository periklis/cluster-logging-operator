package fluentd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ViaQ/logerr/log"
)

// LokiURL returns the endpoint url without any further processing
func (conf *outputLabelConf) LokiURL() string {
	url, err := url.Parse(conf.Target.URL)
	if err != nil {
		log.Error(err, "Invalid Loki URL", "url", conf.Target.URL)
		return ""
	}

	return fmt.Sprintf("%s://%s", url.Scheme, url.Host)
}

// LokiTenant returns the loki tenant ID
func (conf *outputLabelConf) LokiTenant() string {
	if conf.Target.Loki != nil {
		if conf.Target.Loki.TenantID != "" {
			return conf.Target.Loki.TenantID
		}
	}

	url, err := url.Parse(conf.Target.URL)
	if err != nil {
		log.Error(err, "Failed to extract Loki tenant", "url", conf.Target.URL)
		return ""
	}
	return strings.Trim(url.Path, "/")

	// FIXME error handling
}
