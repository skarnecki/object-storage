package main

import (
	"fmt"
	"github.com/docker/docker/client"
	"github.com/google/wire"
	"github.com/spacelift-io/homework-object-storage/config"
	"github.com/spacelift-io/homework-object-storage/handlers"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spacelift-io/homework-object-storage/backend"
	"os"
	"time"
)

const NetworkNameEnvironmentVariable = "PRIVATE_NETWORK_NAME"
const BucketNameEnvironmentVariable = "BUCKET_NAME"
const ObjectPathParameter = "{id:[a-zA-Z0-9]{1,32}}"
const ContainerName = "amazin-object-storage-node"
const MinioUserEnvVariable = "MINIO_ROOT_USER"
const MinioPwdEnvVariable = "MINIO_ROOT_PASSWORD"
const HttpPortEnvVariable = "HTTP_PORT"
const RefreshInterval = 5 * time.Second
const MaxPayloadSize = 8 << (10 * 2) //8 MB

//Use those environment variable for docker connection options
// DOCKER_HOST to set the url to the docker server.
// DOCKER_API_VERSION to set the version of the API to reach, leave empty for latest.
// DOCKER_CERT_PATH to load the TLS certificates from.
// DOCKER_TLS_VERIFY to enable or disable TLS verification, off by default.

var diContainer = wire.NewSet(client.NewClientWithOpts, backend.NewPool, handlers.NewObjectHandler, NewRouter)

func main() {
	log.SetLevel(log.DebugLevel)
	cfg := config.Config{
		os.Getenv(NetworkNameEnvironmentVariable),
		os.Getenv(BucketNameEnvironmentVariable),
		ContainerName,
		MinioUserEnvVariable,
		MinioPwdEnvVariable,
		MaxPayloadSize,
		RefreshInterval,
	}
	router, err := InitializeApp(cfg, client.FromEnv)

	if err != nil {
		log.Fatal(err)
	}

	port, err := strconv.Atoi(os.Getenv(HttpPortEnvVariable))
	if err != nil {
		log.Fatal("Wrong HTTP port number", err)
	}
	log.Printf("Serving HTTP on port %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}

func NewRouter(handler *handlers.ObjectHandler, config config.Config) *mux.Router {
	r := mux.NewRouter()
	r.Use(backend.MaxBytesMiddleware(config.MaxPayloadSize))
	r.HandleFunc("/object/"+ObjectPathParameter, handler.ReadObject).Methods("GET")
	r.HandleFunc("/object/"+ObjectPathParameter, handler.WriteObject).Methods("PUT")
	http.Handle("/", r)
	return r
}
