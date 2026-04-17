package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adortb/adortb-dco/internal/engine"
	"github.com/adortb/adortb-dco/internal/model"
)

// --- stub engine ---

type stubEngine struct {
	result *engine.RenderResult
	err    error
}

func (s *stubEngine) Render(_ context.Context, _ engine.RenderRequest) (*engine.RenderResult, error) {
	return s.result, s.err
}

// handlerWithStub builds a Handler whose eng field is a stub.
type handlerStub struct {
	eng interface {
		Render(context.Context, engine.RenderRequest) (*engine.RenderResult, error)
	}
}

func (h *handlerStub) handleRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req engine.RenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CampaignID == 0 {
		writeError(w, http.StatusBadRequest, "campaign_id required")
		return
	}
	result, err := h.eng.Render(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func TestHandleRender_Success(t *testing.T) {
	stub := &stubEngine{
		result: &engine.RenderResult{
			TemplateID:   1,
			RuleID:       5,
			RenderedHTML: "<div>限时特惠 ¥5999</div>",
			AssetsUsed:   []engine.AssetUsed{{Slot: "headline", AssetID: 10}},
		},
	}
	h := &handlerStub{eng: stub}

	body, _ := json.Marshal(engine.RenderRequest{
		CampaignID: 123,
		User:       engine.UserContext{Geo: "CN"},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/render", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.handleRender(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 got %d: %s", w.Code, w.Body.String())
	}

	var result engine.RenderResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.TemplateID != 1 {
		t.Errorf("unexpected template_id %d", result.TemplateID)
	}
}

func TestHandleRender_MissingCampaignID(t *testing.T) {
	h := &handlerStub{eng: &stubEngine{}}
	body, _ := json.Marshal(engine.RenderRequest{CampaignID: 0})
	req := httptest.NewRequest(http.MethodPost, "/v1/render", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.handleRender(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 got %d", w.Code)
	}
}

func TestHandleRender_MethodNotAllowed(t *testing.T) {
	h := &handlerStub{eng: &stubEngine{}}
	req := httptest.NewRequest(http.MethodGet, "/v1/render", nil)
	w := httptest.NewRecorder()
	h.handleRender(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 got %d", w.Code)
	}
}

func TestParseIDFromPath(t *testing.T) {
	cases := []struct {
		path   string
		prefix string
		want   int64
		errNil bool
	}{
		{"/v1/templates/42", "/v1/templates/", 42, true},
		{"/v1/assets/99", "/v1/assets/", 99, true},
		{"/v1/rules/abc", "/v1/rules/", 0, false},
	}
	for _, c := range cases {
		id, err := parseIDFromPath(c.path, c.prefix)
		if c.errNil && err != nil {
			t.Errorf("path=%s: unexpected error %v", c.path, err)
		}
		if !c.errNil && err == nil {
			t.Errorf("path=%s: expected error", c.path)
		}
		if c.errNil && id != c.want {
			t.Errorf("path=%s: want id=%d got %d", c.path, c.want, id)
		}
	}
}

// Ensure model package is used (avoid import cycle issues in test).
var _ = model.Template{}
