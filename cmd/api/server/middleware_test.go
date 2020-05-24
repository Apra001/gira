package server

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/asankov/gira/internal/fixtures"
	"github.com/golang/mock/gomock"
)

var (
	token = "my_test_token"
)

func setupMiddlewareServer(a Authenticator) *Server {
	return &Server{
		Log:           log.New(os.Stdout, "", 0),
		Authenticator: a,
	}
}

func TestRequireLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	authenticator := fixtures.NewAuthenticatorMock(ctrl)
	srv := setupMiddlewareServer(authenticator)

	authenticator.EXPECT().
		DecodeToken(gomock.Eq(token)).
		Return(nil, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("x-auth-token", token)

	nextHandlerCalled := false
	h := srv.requireLogin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
	}))

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf(`Got ("%d") for StatusCode, expected ("%d")`, w.Code, http.StatusOK)
	}
	if !nextHandlerCalled {
		t.Errorf("Got false for nextHandlerCalled, expected true")
	}
}
func TestRequireLoginError(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(a *fixtures.AuthenticatorMock, r *http.Request)
	}{
		{
			name: "Authenticator error",
			setup: func(a *fixtures.AuthenticatorMock, r *http.Request) {
				a.EXPECT().
					DecodeToken(gomock.Eq(token)).
					Return(nil, errors.New("Authenticator error"))
				r.Header.Set("x-auth-token", token)
			},
		},
		{
			name: "Token not present",
			setup: func(a *fixtures.AuthenticatorMock, r *http.Request) {
				r.Header.Del("x-auth-token")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			authenticator := fixtures.NewAuthenticatorMock(ctrl)
			srv := Server{Authenticator: authenticator}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			testCase.setup(authenticator, r)

			nextHandlerCalled := false
			h := srv.requireLogin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextHandlerCalled = true
			}))

			h.ServeHTTP(w, r)

			if w.Code != http.StatusUnauthorized {
				t.Errorf(`Got ("%d") for StatusCode, expected ("%d")`, w.Code, http.StatusUnauthorized)
			}
			if nextHandlerCalled {
				t.Errorf("Got true for nextHandlerCalled, expected false")
			}
		})
	}
}