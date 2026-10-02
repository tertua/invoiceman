package controllers

import (
	"net/netip"
	"net/url"
	"strings"

	"github.com/tertua/tupay/pkg/configs"
)

// validateEndpointURL rejects non-http(s) targets and SSRF-prone hosts
// (loopback / private / link-local) when running in prod stage.
func validateEndpointURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Hostname() == "" {
		return false
	}
	if strings.EqualFold(configs.Get().Stage, "prod") {
		if isPrivateHost(parsed.Hostname()) {
			return false
		}
	}
	return true
}

// isPrivateHost reports loopback, private, link-local or unspecified IPs,
// plus "localhost" itself. Hostnames that do not parse as IPs are treated
// as public (DNS is resolved by the forwarder's HTTP client).
func isPrivateHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}
