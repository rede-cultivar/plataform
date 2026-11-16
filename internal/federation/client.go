package federation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	remoteDialTimeout    = 10 * time.Second
	remoteRequestTimeout = 20 * time.Second
	maxRemoteRedirects   = 3
)

// NewHTTPClient cria um cliente com proteção contra requisições a endereços
// internos. O parâmetro allowPrivate libera redes privadas apenas em
// desenvolvimento, onde instâncias de teste costumam rodar em localhost.
func NewHTTPClient(allowPrivate bool) *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           safeDialContext(allowPrivate),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   remoteDialTimeout,
		ResponseHeaderTimeout: remoteDialTimeout,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   remoteRequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRemoteRedirects {
				return errors.New("redirecionamentos demais")
			}
			return validateRemoteURL(req.URL, allowPrivate)
		},
	}
}

func safeDialContext(allowPrivate bool) func(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   remoteDialTimeout,
		KeepAlive: 30 * time.Second,
	}

	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		if allowPrivate {
			return dialer.DialContext(ctx, network, address)
		}

		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolver endereço remoto: %w", err)
		}

		for _, candidate := range addresses {
			if isBlockedIP(candidate.IP) {
				continue
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
		}

		return nil, fmt.Errorf("endereço remoto %q não permitido", host)
	}
}

func validateRemoteURL(target *url.URL, allowPrivate bool) error {
	if target == nil {
		return errors.New("endereço remoto ausente")
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return fmt.Errorf("esquema remoto %q não permitido", target.Scheme)
	}
	if strings.TrimSpace(target.Host) == "" {
		return errors.New("host remoto ausente")
	}

	if allowPrivate {
		return nil
	}

	host := target.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("endereço remoto %q não permitido", host)
		}
		return nil
	}

	addresses, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return fmt.Errorf("resolver endereço remoto: %w", err)
	}
	for _, candidate := range addresses {
		if isBlockedIP(candidate.IP) {
			return fmt.Errorf("endereço remoto %q não permitido", host)
		}
	}

	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if !ip.IsGlobalUnicast() {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	// Faixa de NAT de operadora (100.64.0.0/10).
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}

	return false
}
