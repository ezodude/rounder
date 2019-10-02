package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

var app App

func init() {
	log.SetOutput(ioutil.Discard)
}

func newTestingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewServer(handler)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
		},
	}
	return client, s.Close
}

func TestMain(m *testing.M) {
	app = App{}
	app.Initialize()

	code := m.Run()
	os.Exit(code)
}

func TestHandlesHealthCheck(t *testing.T) {
	expected := `{"alive": true}`
	rr := ServeFakeHTTP(
		app.Router,
		"GET",
		"/health",
		[]string{},
	)
	assertStatus(t, rr.Code, http.StatusOK)
	assertEqual(t, rr.Body.String(), expected, "Content did not match.")
}

func TestHandlesIngestSubject(t *testing.T) {
	okJSON := filepath.Join("testdata", "ingestion-raw-success.json")
	okResponse, err := ioutil.ReadFile(okJSON)
	if err != nil {
		fmt.Printf("Cannot read testdata path[%s]\n", okJSON)
		t.FailNow()
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(okResponse)
	})
	client, teardown := newTestingHTTPClient(handler)
	defer teardown()

	tempStorage, err := ioutil.TempDir("", "taxround-ingest")
	if err != nil {
		fmt.Printf("Could not create a Temp dir at path[%s]\n", tempStorage)
		t.FailNow()
	}
	defer os.RemoveAll(tempStorage)

	app.HTTPClient = client
	app.IngestDir = tempStorage
	app.ProviderKey = "a-key"
	app.ProviderURL = "http://www.provider.com/api/v1/search?key=_KEY_&query=_SUBJECT_%20AND%20sourceCountry:%22United%20Kingdom%22&limit=100&format=json"

	expected := `{"ID":"ingestion_off_payrol_working","subject":"off-payrol working","ingested":true,"total":1}`
	rr := ServeFakeHTTP(
		app.Router,
		"POST",
		"/api/v0.1/ingest",
		[]string{"Content-Type: application/json"},
		`{ "subject": "off-payrol working" }`,
	)

	assertStatus(t, rr.Code, http.StatusOK)
	assertEqual(t, rr.Body.String(), expected, "Content did not match.")
}

func assertStatus(tb testing.TB, actual int, expected int, v ...interface{}) {
	if actual != expected {
		_, file, line, _ := runtime.Caller(1)
		msg := fmt.Sprintf("wrong status code: got %v want %v\n", actual, expected)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

func assertEqual(tb testing.TB, actual, expected, msg string, v ...interface{}) {
	if !strings.EqualFold(actual, expected) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

func ServeFakeHTTP(router *mux.Router, method, url string, headers []string, body ...string) *httptest.ResponseRecorder {
	var b string
	if len(body) > 0 {
		b = body[0]
	}

	req, err := http.NewRequest(method, url, strings.NewReader(b))
	if err != nil {
		panic(err)
	}

	for _, header := range headers {
		parts := strings.Split(header, ":")
		req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}
