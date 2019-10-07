package ingest_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/ezodude/rounder/internal/ingest"
)

var okRaw = "testdata/ingestion-raw-success.json"
var key = "api-key"
var subject = "off-payroll"
var dataEndpoint = `http://www.provider.com/api/v1/search?key=_KEY_&query=_SUBJECT_%20AND%20sourceCountry:%22United%20Kingdom%22&limit=100&format=json`
var expectedUrl = `http://www.provider.com/api/v1/search?key=api-key&query=off-payroll%20AND%20sourceCountry:%22United%20Kingdom%22&limit=100&format=json`

func TestIngestionResult(t *testing.T) {
	expectedResult := `ingestion_off_payroll::off-payroll::true::1`

	httpClient, teardown := newTestingHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(mustReadFile(t, okRaw))

		if r.Method != "GET" {
			t.Fatalf("Expected a GET request but got [%s]", r.Method)
		}

		url := fmt.Sprintf("http://%s%s", r.Host, r.RequestURI)
		if url != expectedUrl {
			t.Fatalf("Expected request URL equals [%s] but got [%s]", expectedUrl, url)
		}
	}))
	defer teardown()

	path, err := ioutil.TempDir("", "rounder-ingest")
	if err != nil {
		fmt.Printf("Could not create a Temp dir at path[%s]\n", path)
		t.FailNow()
	}
	defer os.RemoveAll(path)

	actual, err := ingest.New().
		HTTPClient(httpClient).
		Key(key).
		Subject(subject).
		Path(path).
		DataEndpoint(dataEndpoint).
		Do()

	if err != nil {
		t.Fatalf("Did not expect error [%s]", err)
	}

	if actual.String() != expectedResult {
		t.Fatalf("Expected ingest result to equal [%s] but got [%s]", expectedResult, actual)
	}
}

func TestIngestionStoresArticles(t *testing.T) {
	expectedArticles := "testdata/ingestion-success.json"

	httpClient, teardown := newTestingHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(mustReadFile(t, okRaw))

		if r.Method != "GET" {
			t.Fatalf("Expected a GET request but got [%s]", r.Method)
		}

		url := fmt.Sprintf("http://%s%s", r.Host, r.RequestURI)
		if url != expectedUrl {
			t.Fatalf("Expected request URL equals [%s] but got [%s]", expectedUrl, url)
		}
	}))
	defer teardown()

	path, err := ioutil.TempDir("", "rounder-ingest")
	if err != nil {
		fmt.Printf("Could not create a Temp dir at path[%s]\n", path)
		t.FailNow()
	}
	defer os.RemoveAll(path)

	data, err := ingest.New().
		HTTPClient(httpClient).
		Key(key).
		Subject(subject).
		Path(path).
		DataEndpoint(dataEndpoint).
		Do()

	if err != nil {
		t.Fatalf("Did not expect error [%s]", err)
	}

	actualFilename := fmt.Sprintf(`%s.json`, filepath.Join(path, data.ID))
	assertFiles(t, expectedArticles, actualFilename)
}

func mustReadFile(tb testing.TB, filename string, v ...interface{}) []byte {
	result, err := ioutil.ReadFile(filename)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		msg := fmt.Sprintf("Cannot read filename[%s]\n", filename)
		tb.Fatalf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
	}
	return result
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

func assertFiles(tb testing.TB, expectedFile, actualFile string) {
	eContent := mustReadFile(tb, expectedFile)
	aContent := mustReadFile(tb, actualFile)

	msg := fmt.Sprintf("Expected content equals [%s] \n but got [%s]\n", string(eContent), string(aContent))
	assertBytes(tb, eContent, aContent, msg)
}

func assertBytes(tb testing.TB, expected []byte, actual []byte, msg string, v ...interface{}) {
	e := strings.Split(string(expected), "")
	a := strings.Split(string(actual), "")

	sort.Strings(e)
	sort.Strings(a)

	condition := reflect.DeepEqual(e, a)

	if !condition {
		_, file, line, _ := runtime.Caller(1)
		tb.Fatalf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
	}
}
