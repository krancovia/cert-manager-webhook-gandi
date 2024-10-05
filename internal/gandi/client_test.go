package gandi

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	domainsPath = "/domains/"

	testZone      = "example.com"
	testEntryName = "_acme-challenge"
	testToken     = "fakeToken"
)

func TestNewClient(t *testing.T) {
	c := newClient(testToken)
	require.NotNil(t, c)
	require.Equal(t, testToken, c.accessToken)
	require.NotNil(t, c.client)
}

func TestGetTxtRecordValues(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(*testing.T) string
		assertions func(*testing.T, []string, error)
	}{
		{
			name: "not found",
			setup: func(t *testing.T) string {
				mux := http.NewServeMux()
				mux.HandleFunc(domainsPath, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				return srv.URL
			},
			assertions: func(t *testing.T, values []string, err error) {
				require.NoError(t, err)
				require.Empty(t, values)
			},
		},
		{
			name: "unexpected status code",
			setup: func(t *testing.T) string {
				mux := http.NewServeMux()
				mux.HandleFunc(domainsPath, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				})
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				return srv.URL
			},
			assertions: func(t *testing.T, values []string, err error) {
				require.ErrorContains(t, err, "unexpected HTTP status")
				require.ErrorContains(t, err, strconv.Itoa(http.StatusBadRequest))
				require.Empty(t, values)
			},
		},
		{
			name: "success",
			setup: func(t *testing.T) string {
				mux := http.NewServeMux()
				mux.HandleFunc(domainsPath, func(w http.ResponseWriter, r *http.Request) {
					// Assert the method is what we expect
					require.Equal(t, http.MethodGet, r.Method)
					// Assert the path is what we expect
					require.Equal(
						t,
						fmt.Sprintf("%s%s/records/%s/TXT", domainsPath, testZone, testEntryName),
						r.URL.Path,
					)
					// Assert that the Authorization header is set correctly
					require.Equal(
						t,
						fmt.Sprintf("Bearer %s", testToken),
						r.Header.Get("Authorization"),
					)
					_, err := w.Write([]byte(`{
						"rrset_values": ["fakeValue", "anotherFakeValue"]
					}`))
					require.NoError(t, err)
				})
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				return srv.URL
			},
			assertions: func(t *testing.T, values []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"fakeValue", "anotherFakeValue"}, values)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			baseURL := testCase.setup(t)
			c := newClient(testToken)
			c.baseURL = baseURL
			values, err := c.getTxtRecordValues(testZone, testEntryName)
			testCase.assertions(t, values, err)
		})
	}
}

func TestCreateTxtRecord(t *testing.T) {
	var testValues = []string{"fakeValue", "anotherFakeValue"}

	testCases := []struct {
		name       string
		setup      func(*testing.T) string
		assertions func(*testing.T, error)
	}{
		{
			name: "unexpected status code",
			setup: func(t *testing.T) string {
				mux := http.NewServeMux()
				mux.HandleFunc(domainsPath, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				})
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				return srv.URL
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unexpected HTTP status")
				require.ErrorContains(t, err, strconv.Itoa(http.StatusBadRequest))
			},
		},
		{
			name: "success",
			setup: func(t *testing.T) string {
				mux := http.NewServeMux()
				mux.HandleFunc(domainsPath, func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()
					// Assert the method is what we expect
					require.Equal(t, http.MethodPost, r.Method)
					// Assert the path is what we expect
					require.Equal(
						t,
						fmt.Sprintf("%s%s/records", domainsPath, testZone),
						r.URL.Path,
					)
					// Assert that content type was set correctly
					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					// Assert that the Authorization header is set correctly
					require.Equal(
						t,
						fmt.Sprintf("Bearer %s", testToken),
						r.Header.Get("Authorization"),
					)
					// Assert that the body was set correctly
					bodyBytes, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					body := string(bodyBytes)
					for _, value := range testValues {
						require.Contains(t, body, value)
					}
					w.WriteHeader(http.StatusCreated)
				})
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				return srv.URL
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			baseURL := testCase.setup(t)
			c := newClient(testToken)
			c.baseURL = baseURL
			testCase.assertions(
				t,
				c.createTxtRecord(testZone, testEntryName, testValues),
			)
		})
	}
}

func TestUpdateTxtRecord(t *testing.T) {
	var testValues = []string{"fakeValue", "anotherFakeValue"}

	testCases := []struct {
		name       string
		setup      func(*testing.T) string
		assertions func(*testing.T, error)
	}{
		{
			name: "unexpected status code",
			setup: func(t *testing.T) string {
				mux := http.NewServeMux()
				mux.HandleFunc(domainsPath, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				})
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				return srv.URL
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unexpected HTTP status")
				require.ErrorContains(t, err, strconv.Itoa(http.StatusBadRequest))
			},
		},
		{
			name: "success",
			setup: func(t *testing.T) string {
				mux := http.NewServeMux()
				mux.HandleFunc(domainsPath, func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()
					// Assert the method is what we expect
					require.Equal(t, http.MethodPut, r.Method)
					// Assert the path is what we expect
					require.Equal(
						t,
						fmt.Sprintf("%s%s/records/%s/TXT", domainsPath, testZone, testEntryName),
						r.URL.Path,
					)
					// Assert that content type was set correctly
					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					// Assert that the Authorization header is set correctly
					require.Equal(
						t,
						fmt.Sprintf("Bearer %s", testToken),
						r.Header.Get("Authorization"),
					)
					// Assert that the body was set correctly
					bodyBytes, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					body := string(bodyBytes)
					for _, value := range testValues {
						require.Contains(t, body, value)
					}
					w.WriteHeader(http.StatusCreated)
				})
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				return srv.URL
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			baseURL := testCase.setup(t)
			c := newClient(testToken)
			c.baseURL = baseURL
			testCase.assertions(
				t,
				c.updateTxtRecord(testZone, testEntryName, testValues),
			)
		})
	}
}

func TestDeleteTxtRecord(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(*testing.T) string
		assertions func(*testing.T, error)
	}{
		{
			name: "unexpected status code",
			setup: func(t *testing.T) string {
				mux := http.NewServeMux()
				mux.HandleFunc(domainsPath, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				})
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				return srv.URL
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unexpected HTTP status")
				require.ErrorContains(t, err, strconv.Itoa(http.StatusBadRequest))
			},
		},
		{
			name: "success",
			setup: func(t *testing.T) string {
				mux := http.NewServeMux()
				mux.HandleFunc(domainsPath, func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()
					// Assert the method is what we expect
					require.Equal(t, http.MethodDelete, r.Method)
					// Assert the path is what we expect
					require.Equal(
						t,
						fmt.Sprintf("%s%s/records/%s/TXT", domainsPath, testZone, testEntryName),
						r.URL.Path,
					)
					// Assert that the Authorization header is set correctly
					require.Equal(
						t,
						fmt.Sprintf("Bearer %s", testToken),
						r.Header.Get("Authorization"),
					)
					w.WriteHeader(http.StatusOK)
				})
				srv := httptest.NewServer(mux)
				t.Cleanup(srv.Close)
				return srv.URL
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			baseURL := testCase.setup(t)
			c := newClient(testToken)
			c.baseURL = baseURL
			testCase.assertions(
				t,
				c.deleteTxtRecord(testZone, testEntryName),
			)
		})
	}
}
