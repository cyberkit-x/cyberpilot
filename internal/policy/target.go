package policy

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

type TargetKind string

const (
	TargetURL      TargetKind = "url"
	TargetHost     TargetKind = "hostname"
	TargetIP       TargetKind = "ip"
	TargetCIDR     TargetKind = "cidr"
	TargetWildcard TargetKind = "wildcard"
)

type Target struct {
	Kind         TargetKind
	Scheme, Host string
	Port         uint16
	Prefix       netip.Prefix
	Canonical    string
}

func ParseTarget(input string) (Target, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return Target{}, errors.New("target is empty")
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
			return Target{}, errors.New("target URL must be an absolute HTTP or HTTPS URL without userinfo")
		}
		host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
		port, err := parsePort(parsed.Port())
		if err != nil {
			return Target{}, err
		}
		if port == 0 {
			if parsed.Scheme == "https" {
				port = 443
			} else {
				port = 80
			}
		}
		parsed.Host = net.JoinHostPort(host, strconv.Itoa(int(port)))
		parsed.Fragment = ""
		return Target{Kind: TargetURL, Scheme: parsed.Scheme, Host: host, Port: port, Canonical: parsed.String()}, nil
	}
	if strings.HasPrefix(value, "*.") {
		host := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(value, "*."), "."))
		if !validHostname(host) {
			return Target{}, errors.New("wildcard target requires an explicit valid DNS suffix")
		}
		return Target{Kind: TargetWildcard, Host: host, Canonical: "*." + host}, nil
	}
	if prefix, err := netip.ParsePrefix(value); err == nil {
		prefix = prefix.Masked()
		return Target{Kind: TargetCIDR, Prefix: prefix, Canonical: prefix.String()}, nil
	}
	host, portText, err := net.SplitHostPort(value)
	if err == nil {
		port, err := parsePort(portText)
		if err != nil {
			return Target{}, err
		}
		host = strings.Trim(host, "[]")
		return parseHost(host, port)
	}
	if strings.Count(value, ":") == 1 {
		parts := strings.SplitN(value, ":", 2)
		if parts[1] != "" {
			port, portErr := parsePort(parts[1])
			if portErr == nil {
				return parseHost(parts[0], port)
			}
		}
	}
	return parseHost(value, 0)
}
func parseHost(value string, port uint16) (Target, error) {
	host := strings.ToLower(strings.TrimSuffix(value, "."))
	if address, err := netip.ParseAddr(host); err == nil {
		canonical := address.String()
		if port > 0 {
			canonical = net.JoinHostPort(canonical, strconv.Itoa(int(port)))
		}
		return Target{Kind: TargetIP, Host: address.String(), Port: port, Canonical: canonical}, nil
	}
	if !validHostname(host) {
		return Target{}, fmt.Errorf("ambiguous or invalid target %q; use URL, hostname, IP, CIDR, or *.example.com", value)
	}
	canonical := host
	if port > 0 {
		canonical = net.JoinHostPort(host, strconv.Itoa(int(port)))
	}
	return Target{Kind: TargetHost, Host: host, Port: port, Canonical: canonical}, nil
}
func parsePort(value string) (uint16, error) {
	if value == "" {
		return 0, nil
	}
	port, err := strconv.ParseUint(value, 10, 16)
	if err != nil || port == 0 {
		return 0, errors.New("target port must be between 1 and 65535")
	}
	return uint16(port), nil
}
func validHostname(value string) bool {
	if len(value) == 0 || len(value) > 253 || !strings.Contains(value, ".") {
		return false
	}
	allNumeric := true
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
			if r < '0' || r > '9' {
				allNumeric = false
			}
		}
	}
	return !allNumeric
}
