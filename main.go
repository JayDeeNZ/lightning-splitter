package main

import (
	"context"
	"flag"
	"lightning-splitter/config"
	"lightning-splitter/lnd"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
)

var (
	configPath *string
	lndClient  *lnd.Client
	validate   *validator.Validate
)

func init() {
	configPath = flag.String("config", "config/config.yaml", "configuration file")
}

func main() {
	flag.Parse()

	validate = validator.New()

	// Load the configuration file
	config.LoadConfig(*configPath)

	lndClient = lnd.New()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	lndClient.Connect(ctx)
	lndClient.PrintInfo(ctx)

	router := mux.NewRouter()
	apiRouter := router.PathPrefix("/api").Subrouter()

	// set default response headers
	apiRouter.Use(defaultResponseHeaders)

	apiRouter.HandleFunc("/nodeinfo", GetNodeInfo).Methods(http.MethodGet)
	apiRouter.HandleFunc("/register", RegisterPayee).Methods(http.MethodPost)

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", router))
}

func defaultResponseHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
