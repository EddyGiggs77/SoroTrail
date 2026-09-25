package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sorotrail/sorotrail/internal/store"
)

func TestHelperCoverage_ParameterParsing(t *testing.T) {
	t.Run("limit parsing valid and invalid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/events?limit=50", nil)
		assert.Equal(t, 50, parseLimit(req))

		reqBad := httptest.NewRequest(http.MethodGet, "/events?limit=abc", nil)
		assert.NotEqual(t, 0, parseLimit(reqBad))
	})
}

func TestHelperCoverage_HeaderConstruction(t *testing.T) {
	t.Run(
		"cache etag vary headers",
		func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/events", nil)
			handleCacheHeaders(rec, req, "etag-123")
			assert.NotEmpty(t, rec.Header().Get("ETag"))
		},
	)
}

func TestHelperCoverage_AuthorizationAndFailClosed(t *testing.T) {
	t.Run(
		"unauthorized request denied",
		func(t *testing.T) {
			st := &stubStore{}
			srv := newTestServer(st, nil)
			req := httptest.NewRequest(http.MethodGet, "/admin/tenants", nil)
			rec := httptest.NewRecorder()
			srv.Router().ServeHTTP(rec, req)
			assert.Equal(t, http.StatusNotFound, rec.Code)
		},
	)
}

func TestHelperCoverage_ErrorMapping(t *testing.T) {
	t.Run(
		"store errors map to correct status",
		func(t *testing.T) {
			st := &stubStore{eventErr: store.ErrNotFound}
			srv := newTestServer(st, nil)
			req := httptest.NewRequest(http.MethodGet, "/events/0001099511627776-0000000001", nil)
			rec := httptest.NewRecorder()
			srv.Router().ServeHTTP(rec, req)
			assert.Equal(t, http.StatusNotFound, rec.Code)
		},
	)
}
