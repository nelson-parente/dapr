package diagnostics

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dapr/dapr/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
)

func TestHTTPMiddleware(t *testing.T) {
	requestBody := "fake_requestDaprBody"
	responseBody := "fake_responseDaprBody"

	testRequest := fakeHTTPRequest(requestBody)

	// create test httpMetrics
	testHTTP := newHTTPMetrics()
	testHTTP.Init("fakeID", nil, false)

	handler := testHTTP.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte(responseBody))
	}))

	// act
	handler.ServeHTTP(httptest.NewRecorder(), testRequest)

	// assert
	rows, err := view.RetrieveData("http/server/request_count")
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "app_id", rows[0].Tags[0].Key.Name())
	assert.Equal(t, "fakeID", rows[0].Tags[0].Value)
	assert.Equal(t, "status", rows[0].Tags[1].Key.Name())
	assert.Equal(t, "200", rows[0].Tags[1].Value)

	rows, err = view.RetrieveData("http/server/request_bytes")
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "app_id", rows[0].Tags[0].Key.Name())
	assert.Equal(t, "fakeID", rows[0].Tags[0].Value)
	assert.InEpsilon(t, float64(len(requestBody)), (rows[0].Data).(*view.DistributionData).Min, 0)

	rows, err = view.RetrieveData("http/server/response_bytes")
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.InEpsilon(t, float64(len(responseBody)), (rows[0].Data).(*view.DistributionData).Min, 0)

	rows, err = view.RetrieveData("http/server/latency")
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.GreaterOrEqual(t, (rows[0].Data).(*view.DistributionData).Min, 100.0)
}

func TestHTTPMiddlewareWhenMetricsDisabled(t *testing.T) {
	requestBody := "fake_requestDaprBody"
	responseBody := "fake_responseDaprBody"

	testRequest := fakeHTTPRequest(requestBody)

	// create test httpMetrics
	testHTTP := newHTTPMetrics()
	testHTTP.enabled = false
	testHTTP.Init("fakeID", nil, false)
	v := view.Find("http/server/request_count")
	views := []*view.View{v}
	view.Unregister(views...)

	handler := testHTTP.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte(responseBody))
	}))

	// act
	handler.ServeHTTP(httptest.NewRecorder(), testRequest)

	// assert
	rows, err := view.RetrieveData("http/server/request_count")
	require.Error(t, err)
	assert.Nil(t, rows)
}

func TestHTTPMetricsPathNormalizationNotEnabled(t *testing.T) {
	testHTTP := newHTTPMetrics()
	testHTTP.enabled = false
	pathNormalization := &config.PathNormalization{
		Enabled: false,
	}
	testHTTP.Init("fakeID", pathNormalization, true)
	normalizedPath, ok := testHTTP.normalizePath("/orders", pathNormalization.IngressPaths)
	require.False(t, ok)
	require.Equal(t, normalizedPath, "")
}

func TestHTTPMetricsPathNormalizationLegacyIncreasedCardinality(t *testing.T) {
	testHTTP := newHTTPMetrics()
	testHTTP.enabled = false
	pathNormalization := &config.PathNormalization{
		Enabled: true,
		IngressPaths: []string{
			"/orders/{orderID}/items/{itemID}",
			"/orders/{orderID}",
			"/items/{itemID}",
		},
		EgressPaths: []string{
			"/orders/{orderID}/items/{itemID}",
		},
	}
	testHTTP.Init("fakeID", pathNormalization, true)

	tt := []struct {
		pathsList      []string
		path           string
		normalizedPath string
		normalized     bool
	}{
		{pathNormalization.IngressPaths, "", "", false},
		{pathNormalization.IngressPaths, "/orders/12345/items/12345", "/orders/{orderID}/items/{itemID}", true},
		{pathNormalization.EgressPaths, "/orders/12345/items/12345", "/orders/{orderID}/items/{itemID}", true},
		{pathNormalization.IngressPaths, "/items/12345", "/items/{itemID}", true},
		{pathNormalization.EgressPaths, "/items/12345", "/items/12345", true},
		{pathNormalization.IngressPaths, "/basket/12345", "/basket/12345", true},
		{pathNormalization.IngressPaths, "dapr/config", "/dapr/config", true},
	}

	for _, tc := range tt {
		normalizedPath, ok := testHTTP.normalizePath(tc.path, tc.pathsList)
		require.Equal(t, ok, tc.normalized)
		if ok {
			assert.Equal(t, tc.normalizedPath, normalizedPath)
		}
	}
}

func TestHTTPMetricsPathNormalizationLowCardinality(t *testing.T) {
	testHTTP := newHTTPMetrics()
	testHTTP.enabled = false
	pathNormalization := &config.PathNormalization{
		Enabled: true,
		IngressPaths: []string{
			"/orders/{orderID}/items/{itemID}",
			"/orders/{orderID}",
			"/items/{itemID}",
		},
		EgressPaths: []string{
			"/orders/{orderID}/items/{itemID}",
		},
	}
	testHTTP.Init("fakeID", pathNormalization, false)

	tt := []struct {
		pathsList      []string
		path           string
		normalizedPath string
		normalized     bool
	}{
		{pathNormalization.IngressPaths, "", "", false},
		{pathNormalization.IngressPaths, "/orders/12345/items/12345", "/orders/{orderID}/items/{itemID}", true},
		{pathNormalization.EgressPaths, "/orders/12345/items/12345", "/orders/{orderID}/items/{itemID}", true},
		{pathNormalization.IngressPaths, "/items/12345", "/items/{itemID}", true},
		{pathNormalization.EgressPaths, "/items/12345", "/unmatchedpath", true},
		{pathNormalization.IngressPaths, "/basket/12345", "/unmatchedpath", true},
		{pathNormalization.IngressPaths, "dapr/config", "/unmatchedpath", true},
	}

	for _, tc := range tt {
		normalizedPath, ok := testHTTP.normalizePath(tc.path, tc.pathsList)
		require.Equal(t, ok, tc.normalized)
		if ok {
			assert.Equal(t, tc.normalizedPath, normalizedPath)
		}
	}
}

func fakeHTTPRequest(body string) *http.Request {
	req, err := http.NewRequest(http.MethodPost, "http://dapr.io/invoke/method/testmethod", strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Correlation-ID", "e6f4bb20-96c0-426a-9e3d-991ba16a3ebb")
	req.Header.Set("XXX-Remote-Addr", "192.168.0.100")
	req.Header.Set("Transfer-Encoding", "encoding")
	// This is normally set automatically when the request is sent to a server, but in this case we are not using a real server
	req.Header.Set("Content-Length", strconv.FormatInt(req.ContentLength, 10))

	return req
}
