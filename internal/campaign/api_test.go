package campaign

import (
	"github.com/warnerktsang/lucid-ota-campaign-api/internal/auth"
	"github.com/warnerktsang/lucid-ota-campaign-api/internal/entity"
	"github.com/warnerktsang/lucid-ota-campaign-api/internal/test"
	"github.com/warnerktsang/lucid-ota-campaign-api/pkg/log"
	"net/http"
	"testing"
	"time"
)

func TestAPI(t *testing.T) {
	logger, _ := log.NewForTest()
	router := test.MockRouter(logger)
	repo := &mockRepository{items: []entity.Campaign{
		{"123", "campaign123", "2.3.1", "fleet-sf-01", time.Now(), time.Now()},
	}}
	RegisterHandlers(router.Group(""), NewService(repo, logger), auth.MockAuthHandler, logger)
	header := auth.MockAuthHeader()

	tests := []test.APITestCase{
		{"get all", "GET", "/campaigns", "", nil, http.StatusOK, `*"total_count":1*`},
		{"get 123", "GET", "/campaigns/123", "", nil, http.StatusOK, `*campaign123*`},
		{"get unknown", "GET", "/campaigns/1234", "", nil, http.StatusNotFound, ""},
		{"create ok", "POST", "/campaigns", `{"name":"test","target_version":"2.4.0","fleet_id":"fleet-la-02"}`, header, http.StatusCreated, "*test*"},
		{"create ok count", "GET", "/campaigns", "", nil, http.StatusOK, `*"total_count":2*`},
		{"create auth error", "POST", "/campaigns", `{"name":"test","target_version":"2.4.0","fleet_id":"fleet-la-02"}`, nil, http.StatusUnauthorized, ""},
		{"create input error", "POST", "/campaigns", `"name":"test"}`, header, http.StatusBadRequest, ""},
		{"create validation error", "POST", "/campaigns", `{"name":"test"}`, header, http.StatusBadRequest, ""},
		{"update ok", "PUT", "/campaigns/123", `{"name":"campaignxyz","target_version":"2.4.1","fleet_id":"fleet-sf-01"}`, header, http.StatusOK, "*campaignxyz*"},
		{"update verify", "GET", "/campaigns/123", "", nil, http.StatusOK, `*2.4.1*`},
		{"update auth error", "PUT", "/campaigns/123", `{"name":"campaignxyz","target_version":"2.4.1","fleet_id":"fleet-sf-01"}`, nil, http.StatusUnauthorized, ""},
		{"update input error", "PUT", "/campaigns/123", `"name":"campaignxyz"}`, header, http.StatusBadRequest, ""},
		{"delete ok", "DELETE", "/campaigns/123", ``, header, http.StatusOK, "*campaignxyz*"},
		{"delete verify", "DELETE", "/campaigns/123", ``, header, http.StatusNotFound, ""},
		{"delete auth error", "DELETE", "/campaigns/123", ``, nil, http.StatusUnauthorized, ""},
	}
	for _, tc := range tests {
		test.Endpoint(t, router, tc)
	}
}
