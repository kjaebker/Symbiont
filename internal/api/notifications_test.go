package api

import "testing"

func TestValidateNotificationURL(t *testing.T) {
	allowed := []string{"127.0.0.1:8790"}
	cases := []struct {
		url     string
		allowed []string
		ok      bool
	}{
		{"https://ntfy.sh/my-topic", nil, true},
		{"http://ntfy.sh/my-topic", nil, false},
		{"https://127.0.0.1/x", nil, false},
		{"https://192.168.1.10/x", nil, false},
		{"http://127.0.0.1:8790/topic", nil, false},
		{"http://127.0.0.1:8790/topic", allowed, true},
		{"https://127.0.0.1:8790/topic", allowed, true},
		{"http://127.0.0.1:8791/topic", allowed, false}, // other port
		{"http://127.0.0.1/topic", allowed, false},      // no port
		{"ftp://127.0.0.1:8790/topic", allowed, false},
		{"not a url", allowed, false},
	}
	for _, c := range cases {
		got := validateNotificationURL(c.url, c.allowed) == ""
		if got != c.ok {
			t.Errorf("%s (allowed %v): ok=%v, want %v", c.url, c.allowed, got, c.ok)
		}
	}
}
