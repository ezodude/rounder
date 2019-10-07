package ingest_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ezodude/rounder/internal/ingest"
	"github.com/ezodude/rounder/internal/testutil"
)

var okRaw = "testdata/ingestion-raw-success.json"
var key = "api-key"
var subject = "off-payroll"
var dataEndpoint = `http://www.provider.com/api/v1/search?key=_KEY_&query=_SUBJECT_%20AND%20sourceCountry:%22United%20Kingdom%22&limit=100&format=json`
var expectedUrl = `http://www.provider.com/api/v1/search?key=api-key&query=off-payroll%20AND%20sourceCountry:%22United%20Kingdom%22&limit=100&format=json`

func TestIngestionResult(t *testing.T) {
	expectedResult := `ingestion_off_payroll::off-payroll::true::1`

	httpClient, srvrteardown := testutil.NewClientWithServerChecks(t, okRaw, "GET", expectedUrl)
	defer srvrteardown()

	dir, remove := testutil.MustCreateTempDir(t, "", "rounder-ingest")
	defer remove()

	actual, err := ingest.New().
		HTTPClient(httpClient).
		Key(key).
		Subject(subject).
		Path(dir).
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

	httpClient, srvrteardown := testutil.NewClientWithServerChecks(t, okRaw, "GET", expectedUrl)
	defer srvrteardown()

	dir, remove := testutil.MustCreateTempDir(t, "", "rounder-ingest")
	defer remove()

	data, err := ingest.New().
		HTTPClient(httpClient).
		Key(key).
		Subject(subject).
		Path(dir).
		DataEndpoint(dataEndpoint).
		Do()

	if err != nil {
		t.Fatalf("Did not expect error [%s]", err)
	}

	actualFilename := fmt.Sprintf(`%s.json`, filepath.Join(dir, data.ID))
	testutil.AssertFiles(t, expectedArticles, actualFilename)
}
