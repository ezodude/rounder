package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ezodude/rounder/internal/testutil"
)

var app App

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestMain(m *testing.M) {
	app = App{}
	app.Initialize()

	code := m.Run()
	os.Exit(code)
}

func TestHandlesHealthCheck(t *testing.T) {
	expected := `{"alive": true}`
	rr := testutil.ServeFakeHTTP(
		app.Router,
		"GET",
		"/health",
		[]string{},
	)
	testutil.AssertStatus(t, rr.Code, http.StatusOK)
	testutil.AssertEqual(t, rr.Body.String(), expected, "Content did not match.")
}

func TestHandlesIngestSubject(t *testing.T) {
	okJSON := filepath.Join("testdata", "ingestion-raw-success.json")
	okResponse, err := ioutil.ReadFile(okJSON)
	if err != nil {
		fmt.Printf("Cannot read testdata path[%s]\n", okJSON)
		t.FailNow()
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(okResponse) })
	client, teardown := testutil.NewTestingHTTPClient(handler)
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
	app.DataEndpoint = "http://www.provider.com/api/v1/search?key=_KEY_&query=_SUBJECT_%20AND%20sourceCountry:%22United%20Kingdom%22&limit=100&format=json"

	expected := `{"ID":"ingestion_off_payrol_working","subject":"off-payrol working","ingested":true,"total":1}`
	rr := testutil.ServeFakeHTTP(
		app.Router,
		"POST",
		"/api/v0.1/ingest",
		[]string{"Content-Type: application/json"},
		`{ "subject": "off-payrol working" }`,
	)

	testutil.AssertStatus(t, rr.Code, http.StatusOK)
	testutil.AssertEqual(t, rr.Body.String(), expected, "Content did not match.")
}
