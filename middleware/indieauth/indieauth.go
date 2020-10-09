package indieauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Provider interface {
	Authorized() []string
	TokenEndpoint() string
}

type key int

const (
	User key = iota
)

func With(prov Provider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bearer := getBearer(r)
			if len(bearer) == 0 {
				serveJSON(w, http.StatusUnauthorized, oauthError{
					Error:       "unauthorized",
					Description: "missing authentication token",
				})
				return
			}

			token, err := getToken(r.Context(), prov, bearer)
			if err != nil {
				serveJSON(w, http.StatusUnauthorized, oauthError{
					Error:       "unauthorized",
					Description: err.Error(),
				})
				return
			}

			if token.StatusCode != http.StatusOK {
				serveJSON(w, http.StatusUnauthorized, oauthError{
					Error:       "unauthorized",
					Description: "failed to retrieve authentication information",
				})
				return
			}

			for _, allowed := range prov.Authorized() {
				err := compareHostnames(allowed, token.Me)
				if err == nil {
					ctx := context.WithValue(r.Context(), User, allowed)
					r = r.WithContext(ctx)
					next.ServeHTTP(w, r)
					return
				}
			}

			serveJSON(w, http.StatusForbidden, oauthError{
				Error:       "forbidden",
				Description: "user not allowed",
			})
		})
	}
}

func GetUser(ctx context.Context) string {
	v := ctx.Value(User)
	if u, ok := v.(string); ok {
		return u
	}
	return ""
}

type oauthError struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("error while serving json: %s", err)
	}
}

func getBearer(r *http.Request) string {
	token := r.Header.Get("Authorization")
	if len(token) == 0 {
		accessToken := r.URL.Query().Get("access_token")
		if len(accessToken) > 0 {
			token = "Bearer " + accessToken
		}
	}

	return token
}

type tokenResponse struct {
	Me               string `json:"me"`
	ClientID         string `json:"client_id"`
	Scope            string `json:"scope"`
	IssuedBy         string `json:"issued_by"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	StatusCode       int
}

func getToken(ctx context.Context, prov Provider, bearer string) (*tokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", prov.TokenEndpoint(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", bearer)
	req.Header.Add("Accept", "application/json")

	client := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	tokenRes := tokenResponse{
		StatusCode: resp.StatusCode,
	}
	err = json.NewDecoder(resp.Body).Decode(&tokenRes)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return &tokenRes, nil
}

func compareHostnames(a string, allowed string) error {
	h1, err := url.Parse(a)
	if err != nil {
		return err
	}

	if strings.EqualFold(h1.Hostname(), allowed) {
		return fmt.Errorf("hostnames do not match, %s is not %s", h1, allowed)
	}

	return nil
}
