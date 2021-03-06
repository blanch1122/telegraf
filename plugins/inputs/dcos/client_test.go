package dcos

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

const (
	privateKey = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQCwlGyzVp9cqtwiNCgCnaR0kilPZhr4xFBcnXxvQ8/uzOHaWKxj
XWR38cKR3gPh5+4iSmzMdo3HDJM5ks6imXGnp+LPOA5iNewnpLNs7UxA2arwKH/6
4qIaAXAtf5jE46wZIMgc2EW9wGL3dxC0JY8EXPpBFB/3J8gADkorFR8lwwIDAQAB
AoGBAJaFHxfMmjHK77U0UnrQWFSKFy64cftmlL4t/Nl3q7L68PdIKULWZIMeEWZ4
I0UZiFOwr4em83oejQ1ByGSwekEuiWaKUI85IaHfcbt+ogp9hY/XbOEo56OPQUAd
bEZv1JqJOqta9Ug1/E1P9LjEEyZ5F5ubx7813rxAE31qKtKJAkEA1zaMlCWIr+Rj
hGvzv5rlHH3wbOB4kQFXO4nqj3J/ttzR5QiJW24STMDcbNngFlVcDVju56LrNTiD
dPh9qvl7nwJBANILguR4u33OMksEZTYB7nQZSurqXsq6382zH7pTl29ANQTROHaM
PKC8dnDWq8RGTqKuvWblIzzGIKqIMovZo10CQC96T0UXirITFolOL3XjvAuvFO1Q
EAkdXJs77805m0dCK+P1IChVfiAEpBw3bKJArpAbQIlFfdI953JUp5SieU0CQEub
BSSEKMjh/cxu6peEHnb/262vayuCFKkQPu1sxWewLuVrAe36EKCy9dcsDmv5+rgo
Odjdxc9Madm4aKlaT6kCQQCpAgeblDrrxTrNQ+Typzo37PlnQrvI+0EceAUuJ72G
P0a+YZUeHNRqT2pPN9lMTAZGGi3CtcF2XScbLNEBeXge
-----END RSA PRIVATE KEY-----`
)

func TestLogin(t *testing.T) {
	var tests = []struct {
		name          string
		responseCode  int
		responseBody  string
		expectedError error
		expectedToken string
	}{
		{
			name:          "Login successful",
			responseCode:  200,
			responseBody:  `{"token": "XXX.YYY.ZZZ"}`,
			expectedError: nil,
			expectedToken: "XXX.YYY.ZZZ",
		},
		{
			name:          "Unauthorized Error",
			responseCode:  http.StatusUnauthorized,
			responseBody:  `{"title": "x", "description": "y"}`,
			expectedError: &APIError{http.StatusUnauthorized, "x", "y"},
			expectedToken: "",
		},
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKey))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				fmt.Fprintln(w, tt.responseBody)
			})
			ts := httptest.NewServer(handler)
			u, err := url.Parse(ts.URL)
			require.NoError(t, err)

			ctx := context.Background()
			sa := &ServiceAccount{
				AccountID:  "telegraf",
				PrivateKey: key,
			}
			client := NewClusterClient(u, defaultResponseTimeout, 1, nil)
			auth, err := client.Login(ctx, sa)

			require.Equal(t, tt.expectedError, err)

			if tt.expectedToken != "" {
				require.Equal(t, tt.expectedToken, auth.Text)
			} else {
				require.Nil(t, auth)
			}

			ts.Close()
		})
	}
}

func TestGetSummary(t *testing.T) {
	var tests = []struct {
		name          string
		responseCode  int
		responseBody  string
		expectedValue *Summary
		expectedError error
	}{
		{
			name:          "No nodes",
			responseCode:  200,
			responseBody:  `{"cluster": "a", "slaves": []}`,
			expectedValue: &Summary{Cluster: "a", Slaves: []Slave{}},
			expectedError: nil,
		},
		{
			name:          "Unauthorized Error",
			responseCode:  http.StatusUnauthorized,
			responseBody:  `<html></html>`,
			expectedValue: nil,
			expectedError: &APIError{StatusCode: http.StatusUnauthorized, Title: "401 Unauthorized"},
		},
		{
			name:         "Has nodes",
			responseCode: 200,
			responseBody: `{"cluster": "a", "slaves": [{"id": "a"}, {"id": "b"}]}`,
			expectedValue: &Summary{
				Cluster: "a",
				Slaves: []Slave{
					Slave{ID: "a"},
					Slave{ID: "b"},
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// check the path
				w.WriteHeader(tt.responseCode)
				fmt.Fprintln(w, tt.responseBody)
			})
			ts := httptest.NewServer(handler)
			u, err := url.Parse(ts.URL)
			require.NoError(t, err)

			ctx := context.Background()
			client := NewClusterClient(u, defaultResponseTimeout, 1, nil)
			summary, err := client.GetSummary(ctx)

			require.Equal(t, tt.expectedError, err)
			require.Equal(t, tt.expectedValue, summary)

			ts.Close()
		})
	}

}

func TestGetNodeMetrics(t *testing.T) {
	var tests = []struct {
		name          string
		responseCode  int
		responseBody  string
		expectedValue *Metrics
		expectedError error
	}{
		{
			name:          "Empty Body",
			responseCode:  200,
			responseBody:  `{}`,
			expectedValue: &Metrics{},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// check the path
				w.WriteHeader(tt.responseCode)
				fmt.Fprintln(w, tt.responseBody)
			})
			ts := httptest.NewServer(handler)
			u, err := url.Parse(ts.URL)
			require.NoError(t, err)

			ctx := context.Background()
			client := NewClusterClient(u, defaultResponseTimeout, 1, nil)
			m, err := client.GetNodeMetrics(ctx, "foo")

			require.Equal(t, tt.expectedError, err)
			require.Equal(t, tt.expectedValue, m)

			ts.Close()
		})
	}

}

func TestGetContainerMetrics(t *testing.T) {
	var tests = []struct {
		name          string
		responseCode  int
		responseBody  string
		expectedValue *Metrics
		expectedError error
	}{
		{
			name:          "204 No Contents",
			responseCode:  204,
			responseBody:  ``,
			expectedValue: &Metrics{},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// check the path
				w.WriteHeader(tt.responseCode)
				fmt.Fprintln(w, tt.responseBody)
			})
			ts := httptest.NewServer(handler)
			u, err := url.Parse(ts.URL)
			require.NoError(t, err)

			ctx := context.Background()
			client := NewClusterClient(u, defaultResponseTimeout, 1, nil)
			m, err := client.GetContainerMetrics(ctx, "foo", "bar")

			require.Equal(t, tt.expectedError, err)
			require.Equal(t, tt.expectedValue, m)

			ts.Close()
		})
	}

}
