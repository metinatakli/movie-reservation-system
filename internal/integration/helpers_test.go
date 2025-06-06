package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
)

var keysToIgnore = map[string]struct{}{
	"timestamp": {},
	"requestId": {},
	"createdAt": {},
}

func prepareRequest(method, path string, body io.Reader, headers map[string]string) (*http.Request, error) {
	req := httptest.NewRequest(method, path, body)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

func compareResponse(t *testing.T, body io.Reader, expectedResponse string) {
	var actual map[string]any
	require.NoError(t, json.NewDecoder(body).Decode(&actual))

	cleanMap(actual)

	var expected map[string]any
	require.NoError(t, json.Unmarshal([]byte(expectedResponse), &expected))

	// ignore indetermistic fields while comparing
	opts := cmpopts.IgnoreMapEntries(func(k string, _ any) bool {
		return k == "timestamp" || k == "requestId" || k == "createdAt"
	})

	if diff := cmp.Diff(expected, actual, opts); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

func cleanMap(m map[string]any) {
	for k := range m {
		if _, ok := keysToIgnore[k]; ok {
			delete(m, k)
			continue
		}
		if nested, ok := m[k].(map[string]any); ok {
			cleanMap(nested)
		}
	}
}
