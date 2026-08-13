package policy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type NetworkBroker struct {
	Scope          []string
	ResolvedScope  map[string][]netip.Prefix
	Resolver       Resolver
	MaxPerSecond   int
	ProxyURL       string
	DialTimeout    time.Duration
	mu             sync.Mutex
	windowStarted  time.Time
	windowRequests int
}

func (b *NetworkBroker) Client() (*http.Client, error) {
	if len(b.Scope) == 0 {
		return nil, errors.New("network broker requires confirmed scope")
	}
	if b.Resolver == nil {
		b.Resolver = net.DefaultResolver
	}
	if b.DialTimeout <= 0 {
		b.DialTimeout = 10 * time.Second
	}
	dialer := &net.Dialer{Timeout: b.DialTimeout}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			if err := b.acquire(); err != nil {
				return nil, err
			}
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, fmt.Errorf("invalid destination: %w", err)
			}
			if !b.hostInScope(host, port) {
				return nil, fmt.Errorf("destination %s is outside confirmed scope", address)
			}
			addresses, err := b.Resolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("resolve destination: %w", err)
			}
			if len(addresses) == 0 {
				return nil, errors.New("destination resolved to no addresses")
			}
			var last error
			for _, resolved := range addresses {
				if !b.resolvedInScope(host, port, resolved) {
					last = fmt.Errorf("resolved address %s is outside confirmed scope", resolved)
					continue
				}
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.String(), port))
				if err == nil {
					return conn, nil
				}
				last = err
			}
			if last == nil {
				last = errors.New("no allowed resolved destination")
			}
			return nil, last
		},
	}
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(request *http.Request, previous []*http.Request) error {
			if len(previous) >= 10 {
				return errors.New("redirect limit exceeded")
			}
			if !b.urlInScope(request.URL.String()) {
				return fmt.Errorf("redirect destination %s is outside confirmed scope", request.URL.Redacted())
			}
			return nil
		},
	}, nil
}

func (b *NetworkBroker) ProxyEnvironment() map[string]string {
	if strings.TrimSpace(b.ProxyURL) == "" {
		return nil
	}
	return map[string]string{"HTTP_PROXY": b.ProxyURL, "HTTPS_PROXY": b.ProxyURL, "ALL_PROXY": "", "NO_PROXY": ""}
}

func (b *NetworkBroker) acquire() error {
	if b.MaxPerSecond <= 0 {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if b.windowStarted.IsZero() || now.Sub(b.windowStarted) >= time.Second {
		b.windowStarted, b.windowRequests = now, 0
	}
	if b.windowRequests >= b.MaxPerSecond {
		return errors.New("network rate limit exceeded")
	}
	b.windowRequests++
	return nil
}

func (b *NetworkBroker) urlInScope(value string) bool {
	return inScope(value, b.Scope)
}

func (b *NetworkBroker) hostInScope(host, port string) bool {
	for _, raw := range b.Scope {
		scope, err := ParseTarget(raw)
		if err != nil {
			continue
		}
		if scope.Kind == TargetCIDR {
			continue
		}
		if scope.Host == strings.ToLower(strings.TrimSuffix(host, ".")) && (scope.Port == 0 || strconv.Itoa(int(scope.Port)) == port) {
			return true
		}
		if scope.Kind == TargetWildcard && strings.HasSuffix(strings.ToLower(host), "."+scope.Host) {
			return true
		}
	}
	return false
}

func (b *NetworkBroker) resolvedInScope(host, port string, address netip.Addr) bool {
	address = address.Unmap()
	normalizedHost := strings.ToLower(strings.TrimSuffix(host, "."))
	for _, raw := range b.Scope {
		scope, err := ParseTarget(raw)
		if err != nil {
			continue
		}
		if scope.Kind == TargetCIDR && scope.Prefix.Contains(address) {
			return true
		}
		if scope.Host == normalizedHost && (scope.Port == 0 || strconv.Itoa(int(scope.Port)) == port) {
			for _, prefix := range b.ResolvedScope[scope.Host] {
				if prefix.Contains(address) {
					return true
				}
			}
		}
		if scope.Kind == TargetWildcard && strings.HasSuffix(strings.ToLower(host), "."+scope.Host) {
			for _, prefix := range b.ResolvedScope[normalizedHost] {
				if prefix.Contains(address) {
					return true
				}
			}
		}
	}
	return false
}
