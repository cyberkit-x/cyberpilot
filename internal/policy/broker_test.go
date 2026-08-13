package policy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

type resolverFunc func(context.Context, string, string) ([]netip.Addr, error)

func (f resolverFunc) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return f(ctx, network, host)
}

func TestNetworkBrokerLocalRequestAndRedirectRevalidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/redirect" {
			http.Redirect(response, request, "http://outside.example.invalid/", http.StatusFound)
			return
		}
		_, _ = response.Write([]byte("ready"))
	}))
	defer server.Close()
	address := strings.TrimPrefix(server.URL, "http://")
	_, port, _ := net.SplitHostPort(address)
	broker := &NetworkBroker{Scope: []string{"http://fixture.local:" + port}, ResolvedScope: map[string][]netip.Prefix{"fixture.local": {netip.MustParsePrefix("127.0.0.1/32")}}, Resolver: resolverFunc(func(_ context.Context, _, host string) ([]netip.Addr, error) {
		if host != "fixture.local" {
			return nil, errors.New("unexpected host")
		}
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	})}
	client, err := broker.Client()
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Get("http://fixture.local:" + port + "/")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	redirectRequest, requestErr := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://fixture.local:"+port+"/redirect", nil)
	if requestErr != nil {
		t.Fatal(requestErr)
	}
	redirectResponse, err := client.Do(redirectRequest)
	if redirectResponse != nil {
		_ = redirectResponse.Body.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "outside confirmed scope") {
		t.Fatalf("redirect error=%v", err)
	}
}

func TestNetworkBrokerRejectsDNSChangeAndRateExcess(t *testing.T) {
	broker := &NetworkBroker{Scope: []string{"192.0.2.0/24"}, MaxPerSecond: 1, Resolver: resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("198.51.100.1")}, nil
	})}
	if broker.resolvedInScope("target.example", "80", netip.MustParseAddr("198.51.100.1")) {
		t.Fatal("out-of-scope DNS result accepted")
	}
	if err := broker.acquire(); err != nil {
		t.Fatal(err)
	}
	if err := broker.acquire(); err == nil || !strings.Contains(err.Error(), "rate") {
		t.Fatalf("rate error=%v", err)
	}
}

func TestNetworkBrokerNormalizesIPv4MappedResolution(t *testing.T) {
	broker := &NetworkBroker{Scope: []string{"http://127.0.0.1:8080"}, ResolvedScope: map[string][]netip.Prefix{"127.0.0.1": {netip.MustParsePrefix("127.0.0.1/32")}}}
	if !broker.resolvedInScope("127.0.0.1", "8080", netip.MustParseAddr("::ffff:127.0.0.1")) {
		t.Fatal("IPv4-mapped loopback was not normalized before scope comparison")
	}
}

func TestSandboxProxyEnvironmentFailsClosed(t *testing.T) {
	broker := &NetworkBroker{ProxyURL: "http://127.0.0.1:7777"}
	environment := broker.ProxyEnvironment()
	if environment["HTTP_PROXY"] == "" || environment["HTTPS_PROXY"] == "" || environment["NO_PROXY"] != "" {
		t.Fatalf("environment=%#v", environment)
	}
}
