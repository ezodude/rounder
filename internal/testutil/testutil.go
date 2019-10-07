package testutil

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

	"github.com/gorilla/mux"
)

func NewClientWithServerChecks(tb testing.TB, fname, method, url string, v ...interface{}) (*http.Client, func()) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(MustReadFile(tb, fname))

		if r.Method != method {
			msg := fmt.Sprintf("Expected [%s] request but got [%s]", method, r.Method)

			_, file, line, _ := runtime.Caller(1)
			tb.Fatalf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		}

		reqURL := fmt.Sprintf("http://%s%s", r.Host, r.RequestURI)
		if reqURL != url {
			msg := fmt.Sprintf("Expected request URL equals [%s] but got [%s]", url, reqURL)

			_, file, line, _ := runtime.Caller(1)
			tb.Fatalf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		}
	})

	return NewTestingHTTPClient(handler)
}

func NewTestingHTTPClient(handler http.Handler) (*http.Client, func()) {
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

func MustCreateTempDir(tb testing.TB, dir, prefix string, v ...interface{}) (string, func()) {
	result, err := ioutil.TempDir("", "rounder-ingest")
	if err != nil {
		msg := fmt.Sprintf("Could not create a Temp dir at path[%s]", result)

		_, file, line, _ := runtime.Caller(1)
		tb.Fatalf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
	}
	return result, func() { os.RemoveAll(result) }
}

func MustReadFile(tb testing.TB, filename string, v ...interface{}) []byte {
	result, err := ioutil.ReadFile(filename)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		msg := fmt.Sprintf("Cannot read filename[%s]\n", filename)
		tb.Fatalf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
	}
	return result
}

func AssertFiles(tb testing.TB, expectedFile, actualFile string) {
	eContent := MustReadFile(tb, expectedFile)
	aContent := MustReadFile(tb, actualFile)

	msg := fmt.Sprintf("Expected content equals [%s] \n but got [%s]\n", string(eContent), string(aContent))
	AssertBytes(tb, eContent, aContent, msg)
}

func AssertBytes(tb testing.TB, expected []byte, actual []byte, msg string, v ...interface{}) {
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

func AssertStatus(tb testing.TB, actual int, expected int, v ...interface{}) {
	if actual != expected {
		_, file, line, _ := runtime.Caller(1)
		msg := fmt.Sprintf("wrong status code: got %v want %v\n", actual, expected)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

func AssertEqual(tb testing.TB, actual, expected, msg string, v ...interface{}) {
	if !strings.EqualFold(actual, expected) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}
