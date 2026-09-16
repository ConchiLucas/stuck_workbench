package taskgen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/stretchr/testify/require"
)

type materialTransport func(*http.Request) (*http.Response, error)

func (f materialTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMaterialClientRejectsUnavailableAndInvalidResponses(t *testing.T) {
	for _, operation := range []string{"list", "freeze"} {
		for _, scenario := range []struct {
			name   string
			status int
			body   string
			wait   bool
			code   string
		}{
			{name: "HTTP503EvenWithMaterials", status: 503, body: `{"items":[{"kpId":1}]}`, code: "materials_error"},
			{name: "nonJSON", status: 200, body: `<html>upstream proxy error</html>`, code: "materials_response"},
			{name: "truncatedJSON", status: 200, body: `{"items":[{"kpId":1}`, code: "materials_response"},
			{name: "timeout", wait: true, code: "materials_unavailable"},
		} {
			t.Run(operation+"/"+scenario.name, func(t *testing.T) {
				client := NewMaterialClient("http://materials.test")
				client.Client.Transport = materialTransport(func(r *http.Request) (*http.Response, error) {
					if scenario.wait {
						<-r.Context().Done()
						return nil, r.Context().Err()
					}
					w := httptest.NewRecorder()
					w.WriteHeader(scenario.status)
					_, _ = w.Write([]byte(scenario.body))
					return w.Result(), nil
				})
				client.Client.Timeout = 30 * time.Millisecond
				var rows []generation.Material
				var err error
				if operation == "list" {
					rows, err = client.List(context.Background(), "g1")
				} else {
					rows, err = client.Freeze(context.Background(), []generation.Material{{KpID: 1, SourceRevision: "source-v1"}})
				}
				require.Error(t, err)
				require.Nil(t, rows, "untrusted response must not become generation materials")
				var failure *Error
				require.ErrorAs(t, err, &failure)
				require.Equal(t, scenario.code, failure.Code)
				require.Equal(t, 503, failure.Status)
			})
		}
	}
}

func TestMaterialClientPreservesScopeAndFreezeSourceRevision(t *testing.T) {
	type request struct {
		method, path, module string
		items                []struct {
			KpID           int64  `json:"kpId"`
			SourceRevision string `json:"sourceRevision"`
		}
	}
	requests := make(chan request, 2)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := request{method: r.Method, path: r.URL.Path, module: r.URL.Query().Get("moduleCode")}
		if r.Method == http.MethodPost {
			var payload struct {
				Items []struct {
					KpID           int64  `json:"kpId"`
					SourceRevision string `json:"sourceRevision"`
				} `json:"items"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			got.items = payload.Items
		}
		requests <- got
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"kpId":7,"revisionId":"frozen-7","sourceRevision":"source-7"}]}`))
	})
	client := NewMaterialClient("http://materials.test/")
	client.Client.Transport = materialTransport(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Result(), nil
	})
	rows, err := client.List(context.Background(), "组 1&2")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	first := <-requests
	require.Equal(t, http.MethodGet, first.method)
	require.Equal(t, "/api/v1/generation-materials/literacy", first.path)
	require.Equal(t, "组 1&2", first.module)
	frozen, err := client.Freeze(context.Background(), rows)
	require.NoError(t, err)
	require.Equal(t, "frozen-7", frozen[0].RevisionID)
	second := <-requests
	require.Equal(t, http.MethodPost, second.method)
	require.Equal(t, "/api/v1/generation-materials/literacy/freeze", second.path)
	require.Len(t, second.items, 1)
	require.Equal(t, int64(7), second.items[0].KpID)
	require.Equal(t, "source-7", second.items[0].SourceRevision)
}
