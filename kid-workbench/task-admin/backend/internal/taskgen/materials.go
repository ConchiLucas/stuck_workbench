package taskgen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/conchi/study-learning/literacycontract"
	"github.com/conchi/study-task-admin/internal/generation"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type MaterialClient struct {
	Base   string
	Client *http.Client
}

func NewMaterialClient(base string) *MaterialClient {
	return &MaterialClient{Base: strings.TrimRight(base, "/"), Client: &http.Client{Timeout: 35 * time.Second}}
}
func (m *MaterialClient) request(ctx context.Context, path string, body any) ([]generation.Material, error) {
	method := "GET"
	var r io.Reader
	if body != nil {
		method = "POST"
		b, e := json.Marshal(body)
		if e != nil {
			return nil, e
		}
		r = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, m.Base+path, r)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	res, e := m.Client.Do(req)
	if e != nil {
		return nil, fault(503, "materials_unavailable", "素材服务连接失败")
	}
	defer res.Body.Close()
	data, e := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if e != nil {
		return nil, e
	}
	if res.StatusCode >= 300 {
		return nil, fault(res.StatusCode, "materials_error", fmt.Sprintf("素材服务：%s", strings.TrimSpace(string(data))))
	}
	var payload struct {
		Items []generation.Material `json:"items"`
	}
	if e = json.Unmarshal(data, &payload); e != nil {
		return nil, fault(503, "materials_response", "素材服务返回无效数据")
	}
	return payload.Items, nil
}
func (m *MaterialClient) List(ctx context.Context, module string) ([]generation.Material, error) {
	return m.request(ctx, "/api/v1/generation-materials/literacy?moduleCode="+url.QueryEscape(module), nil)
}
func (m *MaterialClient) Freeze(ctx context.Context, items []generation.Material) ([]generation.Material, error) {
	rows := []map[string]any{}
	for _, m := range items {
		row := map[string]any{"kpId": m.KpID, "sourceRevision": m.SourceRevision}
		if len(m.QuestionTypes) > 0 {
			row["questionTypes"] = m.QuestionTypes
		}
		rows = append(rows, row)
	}
	return m.request(ctx, "/api/v1/generation-materials/literacy/freeze", map[string]any{"items": rows})
}

// Verify rechecks the actual immutable media before a draft can be published.
func (m *MaterialClient) Verify(ctx context.Context, qs []generation.Snapshot) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	refs := map[string]*generation.MediaRef{}
	add := func(r *generation.MediaRef) {
		if r != nil {
			refs[r.RevisionID+":"+r.Kind] = r
		}
	}
	for _, q := range qs {
		add(q.Stem.Image)
		add(q.Stem.Audio)
		add(q.WritingTemplate)
		for _, o := range q.Options {
			add(o.Image)
			add(o.Audio)
		}
	}
	for _, ref := range refs {
		path := m.Base + "/api/v1/material-revisions/" + url.PathEscape(ref.RevisionID) + "/media/" + url.PathEscape(ref.Kind)
		req, e := http.NewRequestWithContext(ctx, "GET", path, nil)
		if e != nil {
			return e
		}
		res, e := m.Client.Do(req)
		if e != nil {
			return e
		}
		data, e := io.ReadAll(io.LimitReader(res.Body, (20<<20)+1))
		res.Body.Close()
		if e != nil {
			return e
		}
		if res.StatusCode != 200 || len(data) == 0 || len(data) > 20<<20 {
			return fmt.Errorf("资源不可用（%d）", res.StatusCode)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != ref.SHA256 {
			return fmt.Errorf("资源摘要不匹配")
		}
	}
	return nil
}

func usableMaterials(candidates []generation.Material, spec generation.Spec) []generation.Material {
	out := []generation.Material{}
	selected := map[int64]bool{}
	for _, id := range spec.Scope.KpIDs {
		selected[id] = true
	}
	for _, m := range candidates {
		m.QuestionTypes = nil
		target := len(spec.Scope.ModuleCodes) > 0 && m.ModuleCode == spec.Scope.ModuleCodes[0] && (len(selected) == 0 || selected[m.KpID])
		for _, d := range literacycontract.Registry().Types {
			if spec.TypeCounts[d.Code] > 0 && m.Capabilities[d.Code].Ready && (d.Interaction == "choice" || target) {
				m.QuestionTypes = append(m.QuestionTypes, d.Code)
			}
		}
		if len(m.QuestionTypes) > 0 {
			out = append(out, m)
		}
	}
	return out
}
