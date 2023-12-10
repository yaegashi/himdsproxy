package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

const (
	EnvListen           = "HIMDSPROXY_LISTEN"
	EnvIMDSEndpoint     = "IMDS_ENDPOINT"
	EnvIdentityEndpoint = "IDENTITY_ENDPOINT"
	DefaultListen       = "169.254.169.254:80"
	BasicRealmPrefix    = "Basic realm="
	HIMDSAPIVersion     = "2020-06-01"
)

type App struct {
	Listen              string
	IMDSEndpoint        string
	IdentityEndpoint    string
	IMDSEndpointURL     *url.URL
	IdentityEndpointURL *url.URL
	HttpClient          *http.Client
}

func (app *App) LogErrorf(w http.ResponseWriter, format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	log.Print(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (app *App) IdentityProxy(w http.ResponseWriter, r *http.Request) {
	// https://learn.microsoft.com/en-us/azure/azure-arc/servers/managed-identity-authentication#acquiring-an-access-token-using-rest-api
	log.Printf("himdsproxy: identity: request: %s", r.URL)
	ctx := r.Context()
	req1 := r.Clone(ctx)
	req1.URL.Scheme = app.IdentityEndpointURL.Scheme
	req1.URL.Host = app.IdentityEndpointURL.Host
	query := req1.URL.Query()
	query.Set("api-version", HIMDSAPIVersion)
	req1.URL.RawQuery = query.Encode()
	req1.RequestURI = ""
	log.Printf("himdsproxy: identity: target: %s", req1.URL)
	res, err := app.HttpClient.Do(req1)
	if err != nil {
		app.LogErrorf(w, "himdsproxy: identity: request1 error: %s", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		app.LogErrorf(w, "himdsproxy: identity: response1 error: %s", res.Status)
		return
	}
	a := res.Header.Get("WWW-Authenticate")
	if !strings.HasPrefix(a, BasicRealmPrefix) {
		app.LogErrorf(w, "himdsproxy: identity: response1 error: bad WWW-Authenticate header")
		return
	}
	a = strings.TrimPrefix(a, BasicRealmPrefix)
	a = strings.TrimSpace(a)
	a = strings.Trim(a, `"`)
	b, err := os.ReadFile(a)
	if err != nil {
		app.LogErrorf(w, "himdsproxy: identity: readfile error: %s", err)
		return
	}
	req2 := req1.Clone(ctx)
	req2.Header.Set("Authorization", "Basic "+string(b))
	res2, err := app.HttpClient.Do(req2)
	if err != nil {
		app.LogErrorf(w, "himdsproxy: identity: request2 error: %s", err)
		return
	}
	defer res2.Body.Close()
	for name, values := range res2.Header {
		w.Header()[name] = values
	}
	w.WriteHeader(res2.StatusCode)
	_, err = io.Copy(w, res2.Body)
	if err != nil {
		log.Printf("himdsproxy: identity: copy error: %s", err)
	}
}

func (app *App) Main(ctx context.Context) error {
	var err error
	app.IdentityEndpointURL, err = url.Parse(app.IdentityEndpoint)
	if err != nil {
		return err
	}
	app.IMDSEndpointURL, err = url.Parse(app.IMDSEndpoint)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc(app.IdentityEndpointURL.Path, app.IdentityProxy)
	mux.Handle("/", httputil.NewSingleHostReverseProxy(app.IMDSEndpointURL))
	log.Printf("himdsproxy: listening on %s", app.Listen)
	return http.ListenAndServe(app.Listen, mux)
}

func main() {
	app := &App{
		Listen:           os.Getenv(EnvListen),
		IMDSEndpoint:     os.Getenv(EnvIMDSEndpoint),
		IdentityEndpoint: os.Getenv(EnvIdentityEndpoint),
		HttpClient:       &http.Client{},
	}
	if app.IMDSEndpoint == "" || app.IdentityEndpoint == "" {
		log.Fatalf("himdsproxy: missing env vars: %q %q)", EnvIMDSEndpoint, EnvIdentityEndpoint)
	}
	if app.Listen == "" {
		app.Listen = DefaultListen
	}
	err := app.Main(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
