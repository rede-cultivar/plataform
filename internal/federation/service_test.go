package federation

import (
	"errors"
	"net"
	"net/url"
	"testing"
	"time"
)

func TestObjectHelpers(t *testing.T) {
	if got := objectID("https://remote.example/objects/1"); got != "https://remote.example/objects/1" {
		t.Errorf("objectID(string) = %q", got)
	}
	if got := objectID(map[string]any{"id": "https://remote.example/objects/1"}); got != "https://remote.example/objects/1" {
		t.Errorf("objectID(map) = %q", got)
	}
	if got := objectID(nil); got != "" {
		t.Errorf("objectID(nil) = %q, want empty", got)
	}

	if got := objectType(map[string]any{"type": "Follow"}); got != "Follow" {
		t.Errorf("objectType(map) = %q, want Follow", got)
	}
	if got := objectType("Follow"); got != "" {
		t.Errorf("objectType(string) = %q, want empty", got)
	}

	if got := embeddedActor(map[string]any{"actor": "https://remote.example/users/bob"}); got != "https://remote.example/users/bob" {
		t.Errorf("embeddedActor(map) = %q", got)
	}
	if got := embeddedActor(nil); got != "" {
		t.Errorf("embeddedActor(nil) = %q, want empty", got)
	}
}

func TestDeliveryBackoff(t *testing.T) {
	cases := []struct {
		attempts int
		want     time.Duration
	}{
		{0, 2 * time.Minute},
		{1, 2 * time.Minute},
		{2, 4 * time.Minute},
		{3, 8 * time.Minute},
		{6, time.Hour},
		{20, time.Hour},
	}

	for _, tc := range cases {
		if got := deliveryBackoff(tc.attempts); got != tc.want {
			t.Errorf("deliveryBackoff(%d) = %s, want %s", tc.attempts, got, tc.want)
		}
	}
}

func TestTruncateError(t *testing.T) {
	short := errors.New("falha curta")
	if got := truncateError(short); got != "falha curta" {
		t.Errorf("truncateError(short) = %q", got)
	}

	long := errors.New(stringOfLength(600))
	if got := truncateError(long); len(got) != 500 {
		t.Errorf("truncateError(long) length = %d, want 500", len(got))
	}
}

func stringOfLength(length int) string {
	value := make([]byte, length)
	for i := range value {
		value[i] = 'x'
	}
	return string(value)
}

func TestSameURL(t *testing.T) {
	cases := []struct {
		left  string
		right string
		want  bool
	}{
		{"https://Cultivar.example/actors/ana", "https://cultivar.example/actors/ana/", true},
		{"https://cultivar.example/actors/ana", "https://cultivar.example/actors/bob", false},
		{"https://cultivar.example/actors/ana", "http://cultivar.example/actors/ana", false},
		{"", "https://cultivar.example", false},
	}

	for _, tc := range cases {
		if got := sameURL(tc.left, tc.right); got != tc.want {
			t.Errorf("sameURL(%q, %q) = %t, want %t", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestValidateRemoteURL(t *testing.T) {
	cases := []struct {
		raw          string
		allowPrivate bool
		wantErr      bool
	}{
		{"https://1.1.1.1/actors/ana", false, false},
		{"http://8.8.8.8/actors/ana", false, false},
		{"http://127.0.0.1:8080/actors/ana", false, true},
		{"http://10.0.0.2/actors/ana", false, true},
		{"http://192.168.1.2/actors/ana", false, true},
		{"http://169.254.169.254/latest/meta-data", false, true},
		{"http://[::1]/actors/ana", false, true},
		{"http://127.0.0.1:8080/actors/ana", true, false},
		{"ftp://cultivar.example/actors/ana", false, true},
		{"https:///actors/ana", false, true},
	}

	for _, tc := range cases {
		target, err := url.Parse(tc.raw)
		if err != nil {
			t.Fatalf("url.Parse(%q) error = %v", tc.raw, err)
		}

		err = validateRemoteURL(target, tc.allowPrivate)
		if (err != nil) != tc.wantErr {
			t.Errorf("validateRemoteURL(%q, private=%t) error = %v, wantErr = %t", tc.raw, tc.allowPrivate, err, tc.wantErr)
		}
	}
}

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1",
		"10.1.2.3",
		"172.16.0.1",
		"192.168.0.1",
		"169.254.1.1",
		"100.64.0.1",
		"::1",
		"fc00::1",
		"fe80::1",
		"0.0.0.0",
	}
	for _, raw := range blocked {
		ip := net.ParseIP(raw)
		if !isBlockedIP(ip) {
			t.Errorf("isBlockedIP(%q) = false, want true", raw)
		}
	}

	allowed := []string{"8.8.8.8", "1.1.1.1", "2001:4860:4860::8888"}
	for _, raw := range allowed {
		ip := net.ParseIP(raw)
		if isBlockedIP(ip) {
			t.Errorf("isBlockedIP(%q) = true, want false", raw)
		}
	}
}
