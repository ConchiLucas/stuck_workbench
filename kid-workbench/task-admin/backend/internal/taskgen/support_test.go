package taskgen

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type supportTransport func(*http.Request) (*http.Response, error)

func (f supportTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRuntimeSupportAcceptsEnvelopeAndHonorsReadiness(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		ready      bool
	}{
		{"learning envelope", `{"data":{"contractVersion":2,"types":[{"code":"write_char"}]},"error":null}`, true},
		{"explicit unavailable", `{"contractVersion":2,"types":[{"code":"write_char","ready":false}]}`, false},
		{"renderer list", `{"contractVersion":2,"questionTypes":["write_char"]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRuntimeSupport("http://learning", "http://app")
			r.Client = &http.Client{Transport: supportTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}
			got, err := r.fetch(context.Background(), r.LearningURL, "/capabilities")
			if err != nil || got["write_char"] != tc.ready {
				t.Fatalf("support=%v err=%v, want ready=%v", got, err, tc.ready)
			}
		})
	}
}

func TestWritingPublishCanPauseWithoutDisablingChoice(t *testing.T) {
	r := NewRuntimeSupport("http://learning", "http://app")
	r.WritingEnabled = false
	r.Client = &http.Client{Transport: supportTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"contractVersion":2,"questionTypes":["glyph_sense","sense_char","write_char"]}`))}, nil
	})}
	for _, typ := range r.Check(context.Background()).Types {
		if typ.Ready != (typ.Code != "write_char") {
			t.Fatalf("unexpected support: %+v", typ)
		}
	}
}
