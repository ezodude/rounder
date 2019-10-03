package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ezodude/rounder/pkg/ingest"
	"github.com/gorilla/mux"
)

const (
	storage = "./repo"
)

// App struct inits and runs the application
type App struct {
	Router      *mux.Router
	HTTPClient  *http.Client
	IngestDir   string
	ProviderKey string
	DataEndpoint string
}

type ingestPayload struct {
	Subject string `json:"subject"`
}

// Initialize initialises all routes
func (a *App) Initialize() {
	log.Println("App initializing")

	a.HTTPClient = &http.Client{}
	a.IngestDir = storage
	a.Router = mux.NewRouter()
	a.Router.HandleFunc("/health", a.healthCheckHandler).Methods("GET")
	a.Router.HandleFunc("/api/v0.1/ingest", a.ingestHandler).
		Methods("POST").
		Headers("Content-Type", "application/json")
	// a.Router.HandleFunc("/api/v0.1/topics/digest", a.topicsDigestHandler).
	// 	Methods("POST").
	// 	Headers("Content-Type", "application/json")
	// a.Router.HandleFunc("/api/v0.1/sentiment/digest", a.sentimentDigestHandler).
	// 	Methods("POST").
	// 	Headers("Content-Type", "application/json")
	a.Router.HandleFunc("/api/v0.1/ingest", a.ingestHandler).
		Methods("POST").
		Headers("Content-Type", "application/json")
}

func (a *App) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handler healthCheckHandler request received")

	w.WriteHeader(http.StatusOK)
	log.Printf("%s status [%d]\n", r.RequestURI, http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"alive": true}`)
}

func (a *App) ingestHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handler ingestHandler request received")

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("%s status [%d]: %s\n", r.RequestURI, http.StatusInternalServerError, err.Error())
		return
	}
	defer r.Body.Close()

	var payload ingestPayload
	err = json.Unmarshal(data, &payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("%s status [%d]: %s\n", r.RequestURI, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("Handler ingestHandler payload:[%#v]", payload)

	result, err := ingest.New().
		HTTPClient(a.HTTPClient).
		Key(a.ProviderKey).
		Subject(payload.Subject).
		Path(a.IngestDir).
		DataEndpoint(a.DataEndpoint).
		Do()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("%s status [%d]: %s\n", r.RequestURI, http.StatusInternalServerError, err.Error())
		return
	}

	res, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("%s status [%d]: %s\n", r.RequestURI, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("%s status [%d]\n", r.RequestURI, http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

// Run will run the server in a non blocking goroutine
func (a *App) Run(port string) {
	addr := fmt.Sprintf(":%s", port)

	srv := &http.Server{
		Addr: addr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      a.Router,
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Printf("Running server on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)

	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	log.Println("shutting down")
	os.Exit(0)
}

func main() {
	providerKey, found := os.LookupEnv("PROVIDER_KEY")
	if !found {
		panic(fmt.Errorf("Data collection PROVIDER_KEY ENV is missing"))
	}

	dataEndpoint, found := os.LookupEnv("DATA_ENDPOINT")
	if !found {
		panic(fmt.Errorf("Data collection DATA_ENDPOINT ENV is missing"))
	}

	port, found := os.LookupEnv("ROUNDER_API_PORT")
	if !found {
		port = "8001"
	}

	a := App{}
	a.ProviderKey = providerKey
	a.DataEndpoint = dataEndpoint
	a.Initialize()
	a.Run(port)
}
