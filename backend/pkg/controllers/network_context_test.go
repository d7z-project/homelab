package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	intservice "homelab/pkg/services/network/intelligence"
	ipservice "homelab/pkg/services/network/ip"
	siteservice "homelab/pkg/services/network/site"
)

func TestWithIPControllerDepsInjectsDependencies(t *testing.T) {
	t.Parallel()

	poolService := &ipservice.IPPoolService{}
	engine := &ipservice.AnalysisEngine{}
	exports := &ipservice.ExportManager{}
	handler := WithIPControllerDeps(poolService, engine, exports)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deps, ok := ipDepsFromRequest(w, r)
		if !ok {
			t.Fatal("expected ip dependencies")
		}
		if deps.poolService != poolService || deps.analysis != engine || deps.exports != exports {
			t.Fatal("unexpected ip dependencies")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestWithSiteControllerDepsInjectsDependencies(t *testing.T) {
	t.Parallel()

	poolService := &siteservice.SitePoolService{}
	engine := &siteservice.AnalysisEngine{}
	exports := &siteservice.ExportManager{}
	handler := WithSiteControllerDeps(poolService, engine, exports)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deps, ok := siteDepsFromRequest(w, r)
		if !ok {
			t.Fatal("expected site dependencies")
		}
		if deps.poolService != poolService || deps.analysis != engine || deps.exports != exports {
			t.Fatal("unexpected site dependencies")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestWithIntelligenceControllerDepsInjectsDependencies(t *testing.T) {
	t.Parallel()

	service := &intservice.IntelligenceService{}
	handler := WithIntelligenceControllerDeps(service)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deps, ok := intelligenceDepsFromRequest(w, r)
		if !ok {
			t.Fatal("expected intelligence dependencies")
		}
		if deps.service != service {
			t.Fatal("unexpected intelligence dependencies")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestIPInfoHandlerRequiresInjectedDependencies(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/network/ip/analysis/info?ip=1.1.1.1", nil)
	IPInfoHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 without dependencies, got %d", rec.Code)
	}
}
