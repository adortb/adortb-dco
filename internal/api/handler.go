package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/adortb/adortb-dco/internal/engine"
	"github.com/adortb/adortb-dco/internal/model"
	"github.com/adortb/adortb-dco/internal/repo"
)

type Handler struct {
	eng  *engine.Engine
	repo *repo.PGRepo
}

func NewHandler(eng *engine.Engine, r *repo.PGRepo) *Handler {
	return &Handler{eng: eng, repo: r}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/render", h.handleRender)
	mux.HandleFunc("/v1/templates", h.handleTemplates)
	mux.HandleFunc("/v1/templates/", h.handleTemplateByID)
	mux.HandleFunc("/v1/assets", h.handleAssets)
	mux.HandleFunc("/v1/assets/", h.handleAssetByID)
	mux.HandleFunc("/v1/rules", h.handleRules)
	mux.HandleFunc("/v1/rules/", h.handleRuleByID)
}

// POST /v1/render
func (h *Handler) handleRender(w http.ResponseWriter, r *http.Request) {
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

// /v1/templates (GET list, POST create)
func (h *Handler) handleTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		list, err := h.repo.ListTemplates(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var t model.Template
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.repo.CreateTemplate(ctx, &t); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, t)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// /v1/templates/{id} (PUT update)
func (h *Handler) handleTemplateByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, err := parseIDFromPath(r.URL.Path, "/v1/templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var t model.Template
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t.ID = id
	if err := h.repo.UpdateTemplate(r.Context(), &t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// /v1/assets
func (h *Handler) handleAssets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		list, err := h.repo.ListAssets(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var a model.Asset
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.repo.CreateAsset(ctx, &a); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, a)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// /v1/assets/{id}
func (h *Handler) handleAssetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, err := parseIDFromPath(r.URL.Path, "/v1/assets/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var a model.Asset
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	a.ID = id
	if err := h.repo.UpdateAsset(r.Context(), &a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// /v1/rules
func (h *Handler) handleRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		list, err := h.repo.ListRules(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var rule model.Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.repo.CreateRule(ctx, &rule); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, rule)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// /v1/rules/{id}
func (h *Handler) handleRuleByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, err := parseIDFromPath(r.URL.Path, "/v1/rules/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var rule model.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rule.ID = id
	if err := h.repo.UpdateRule(r.Context(), &rule); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func parseIDFromPath(path, prefix string) (int64, error) {
	seg := strings.TrimPrefix(path, prefix)
	seg = strings.Split(seg, "/")[0]
	return strconv.ParseInt(seg, 10, 64)
}

// ensure Handler satisfies context usage (suppress unused import)
var _ context.Context = (context.Context)(nil)
