package ingest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	baseURL = "http://metabase.moreover.com/api/v10/searchArticles"
)

// Ingestion configures and performs a subject ingestion
type Ingestion struct {
	key         string
	subject     string
	path        string
	providerURL string
	httpClient  *http.Client
}

// Result is the outcome of an ingestion
type Result struct {
	ID, Subject string
	Ingested    bool
	Total       int
}

func (r *Result) String() string {
	return fmt.Sprintf(`%s::%s::%t::%d`, r.ID, r.Subject, r.Ingested, r.Total)
}

// New creates a Ingestion that can be configured
func New() *Ingestion {
	return &Ingestion{httpClient: &http.Client{}}
}

// Key configures the Ingestion key used for querying the downstream server
func (i *Ingestion) Key(k string) *Ingestion {
	i.key = k
	return i
}

// Subject configures the Ingestion subject used for querying the downstream server
func (i *Ingestion) Subject(s string) *Ingestion {
	i.subject = s
	return i
}

// Path configures the Ingestion directory path used for storing ingested data
func (i *Ingestion) Path(p string) *Ingestion {
	i.path = p
	return i
}

// ProviderURL configures the Ingestion data provider including placeholders for key + subject
// For now it assumes,
// - To be called using a GET request
// - The [_KEY_] and [_SUBJECT_] placeholders will be replaced with the configured key and subject
// For e.g https://provider.com/endpoint?key=_KEY_&query=_SUBJECT_
func (i *Ingestion) ProviderURL(url string) *Ingestion {
	i.providerURL = url
	return i
}

// HTTPClient configures the Ingestion httpClient used for creating http requests
func (i *Ingestion) HTTPClient(h *http.Client) *Ingestion {
	i.httpClient = h
	return i
}

// GenerateRepoID generates an ingestion repo based on the ingestion subject
func (i *Ingestion) GenerateRepoID() string {
	parts := regexp.MustCompile(`[-\s]`).Split(i.subject, -1)
	return fmt.Sprintf(`ingestion_%s`, strings.Join(parts, "_"))
}

// SubjectRepo is the full path where the article data should be stored
func (i *Ingestion) SubjectRepo() string {
	return fmt.Sprintf(`%s.json`, filepath.Join(i.path, i.GenerateRepoID()))
}

// Do runs an Ingestion storing ingested data at the configured path
func (i *Ingestion) Do() (*Result, error) {
	target := strings.Replace(i.providerURL, "_KEY_", i.key, 1)
	target = strings.Replace(target, "_SUBJECT_", i.subject, 1)

	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return nil, err
	}

	resp, err := i.httpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	res := struct {
		Status   string                   `json:"status"`
		Total    string                   `json:"totalResults"`
		Articles []map[string]interface{} `json:"articles"`
	}{}

	resData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resData, &res); err != nil {
		return nil, err
	}

	data := new(bytes.Buffer)
	enc := json.NewEncoder(data)
	enc.SetEscapeHTML(false)
	enc.Encode(res.Articles)

	if err := ioutil.WriteFile(i.SubjectRepo(), data.Bytes(), 0666); err != nil {
		return nil, err
	}

	total, _ := strconv.Atoi(res.Total)
	return &Result{
		ID:       i.GenerateRepoID(),
		Subject:  i.subject,
		Ingested: strings.ToLower(res.Status) == "success",
		Total:    total,
	}, nil
}
