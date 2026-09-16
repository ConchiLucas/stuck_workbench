package taskgen

import (
	"context"
	"encoding/json"
	"github.com/conchi/study-learning/literacycontract"
	"github.com/conchi/study-task-admin/internal/generation"
	"io"
	"net/http"
	"time"
)

type RuntimeSupport struct {
	LearningURL    string
	AppURL         string
	Client         *http.Client
	WritingEnabled bool
}

func NewRuntimeSupport(learning, app string) *RuntimeSupport {
	return &RuntimeSupport{LearningURL: learning, AppURL: app, Client: &http.Client{Timeout: 3 * time.Second}, WritingEnabled: true}
}

type TypeSupport struct {
	literacycontract.Definition
	Ready   bool     `json:"ready"`
	Reasons []string `json:"reasons"`
}
type SupportResult struct {
	ContractVersion int           `json:"contractVersion"`
	Types           []TypeSupport `json:"types"`
}

func (r *RuntimeSupport) fetch(ctx context.Context, base, path string) (map[string]bool, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", base+path, nil)
	if e != nil {
		return nil, e
	}
	res, e := r.Client.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fault(503, "unsupported_runtime", "练习服务能力未就绪")
	}
	type capability struct {
		ContractVersion int      `json:"contractVersion"`
		QuestionTypes   []string `json:"questionTypes"`
		Types           []struct {
			Code  string `json:"code"`
			Ready *bool  `json:"ready"`
		} `json:"types"`
	}
	var raw json.RawMessage
	if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&raw); e != nil {
		return nil, e
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if e = json.Unmarshal(raw, &envelope); e != nil {
		return nil, e
	}
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		raw = envelope.Data
	}
	var body capability
	if e = json.Unmarshal(raw, &body); e != nil {
		return nil, e
	}
	if body.ContractVersion < 2 {
		return nil, fault(503, "unsupported_runtime", "练习服务版本需要更新")
	}
	out := map[string]bool{}
	for _, t := range body.Types {
		out[t.Code] = t.Ready == nil || *t.Ready
	}
	for _, t := range body.QuestionTypes {
		out[t] = true
	}
	return out, nil
}
func (r *RuntimeSupport) Check(ctx context.Context) SupportResult {
	learner, le := r.fetch(ctx, r.LearningURL, "/api/v1/question-types/literacy")
	app, ae := r.fetch(ctx, r.AppURL, "/literacy-capabilities.json")
	out := SupportResult{ContractVersion: 2, Types: []TypeSupport{}}
	for _, t := range literacycontract.Registry().Types {
		d := TypeSupport{Definition: t, Ready: true, Reasons: []string{}}
		if t.Code == "write_char" && !r.WritingEnabled {
			d.Ready = false
			d.Reasons = append(d.Reasons, "已暂停发布新的听写题；已有练习仍可完成")
		}
		if le != nil || !learner[t.Code] {
			d.Ready = false
			d.Reasons = append(d.Reasons, "学习服务尚未支持该题型")
		}
		if ae != nil || !app[t.Code] {
			d.Ready = false
			d.Reasons = append(d.Reasons, "识字App尚未部署该题型")
		}
		out.Types = append(out.Types, d)
	}
	return out
}
func (r *RuntimeSupport) Verify(ctx context.Context, qs []generation.Snapshot) error {
	needs := map[string]bool{}
	for _, q := range qs {
		if q.SchemaVersion >= 2 {
			needs[q.QuestionType] = true
		}
	}
	if len(needs) == 0 {
		return nil
	}
	for _, d := range r.Check(ctx).Types {
		if needs[d.Code] && !d.Ready {
			return fault(503, "runtime_unavailable", d.Label+"："+d.Reasons[0])
		}
	}
	return nil
}
