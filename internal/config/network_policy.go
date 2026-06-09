package config

import (
	"fmt"
	"net/url"
	"strings"
)

type NetworkPolicyList string

const (
	NetworkPolicyAllow NetworkPolicyList = "allow"
	NetworkPolicyDeny  NetworkPolicyList = "deny"
)

func AddShellNetworkHost(cfg Config, list NetworkPolicyList, host string) (Config, error) {
	host = normalizeNetworkHost(host)
	if host == "" {
		return cfg, fmt.Errorf("host is required")
	}
	policy := &cfg.Tools.Shell.NetworkPolicy
	switch list {
	case NetworkPolicyAllow:
		policy.AllowHosts = addUnique(policy.AllowHosts, host)
	case NetworkPolicyDeny:
		policy.DenyHosts = addUnique(policy.DenyHosts, host)
	default:
		return cfg, fmt.Errorf("unknown network policy list %q", list)
	}
	return cfg, nil
}

func RemoveShellNetworkHost(cfg Config, list NetworkPolicyList, host string) (Config, error) {
	host = normalizeNetworkHost(host)
	if host == "" {
		return cfg, fmt.Errorf("host is required")
	}
	policy := &cfg.Tools.Shell.NetworkPolicy
	switch list {
	case NetworkPolicyAllow:
		policy.AllowHosts = removeValue(policy.AllowHosts, host)
	case NetworkPolicyDeny:
		policy.DenyHosts = removeValue(policy.DenyHosts, host)
	default:
		return cfg, fmt.Errorf("unknown network policy list %q", list)
	}
	return cfg, nil
}

func SetShellNetworkPolicyEnabled(cfg Config, enabled bool) Config {
	cfg.Tools.Shell.NetworkPolicy.Enabled = enabled
	return cfg
}

func SetShellNetworkPolicyDenyPrivate(cfg Config, enabled bool) Config {
	cfg.Tools.Shell.NetworkPolicy.DenyPrivate = enabled
	return cfg
}

func SetShellNetworkPolicyRestrictToAllow(cfg Config, enabled bool) Config {
	cfg.Tools.Shell.NetworkPolicy.RestrictToAllowHosts = enabled
	return cfg
}

func normalizeNetworkHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if strings.Contains(host, "://") {
		parsed, err := url.Parse(host)
		if err == nil && parsed.Hostname() != "" {
			host = parsed.Hostname()
		}
	}
	host = strings.Trim(host, "/.")
	return host
}

func addUnique(values []string, value string) []string {
	for _, item := range values {
		if strings.EqualFold(item, value) {
			return values
		}
	}
	return append(values, value)
}

func removeValue(values []string, value string) []string {
	out := values[:0]
	for _, item := range values {
		if !strings.EqualFold(item, value) {
			out = append(out, item)
		}
	}
	return out
}
