package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spacelift-io/homework-object-storage/backend"
	"github.com/spacelift-io/homework-object-storage/handlers"
	"net/http"
	"os"
	"strconv"
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

func main() {
	//TODO check for network name in env
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal("Can't connect to docker daemon.", err)
	}

	//Create minio server pool manager
	servers, err := backend.NewPool(
		context.Background(), cli,
		os.Getenv(NetworkNameEnvironmentVariable),
		os.Getenv(BucketNameEnvironmentVariable),
		ContainerName,
		MinioUserEnvVariable,
		MinioPwdEnvVariable,
	)
	if err != nil {
		log.Fatal("Can't connect to servers handler.", err)
	}

	//Schedule every `RefreshInterval` server list refresh
	go func() {
		for _ = range time.Tick(RefreshInterval) {
			err = servers.EndpointList(context.Background())
			if err != nil {
				log.Fatal("Can't connect to servers handler.", err)
			}
			log.Infof("Found %d servers servers", len(servers.Servers))
		}
	}()

	//Prepare and serve HTTP
	r := mux.NewRouter()
	r.Use(maxBytesMiddleware)
	r.HandleFunc("/object/"+ObjectPathParameter, handlers.NewReadObjectHandler(servers)).Methods("GET")
	r.HandleFunc("/object/"+ObjectPathParameter, handlers.NewWriteObjectHandler(servers)).Methods("PUT")
	http.Handle("/", r)

	port, err := strconv.Atoi(os.Getenv(HttpPortEnvVariable))
	if err != nil {
		log.Fatal("Wrong HTTP port number", err)
	}
	log.Printf("Serving HTTP on port %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}

//Middleware to check for
func maxBytesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Just in case if someone provides different content length
		r.Body = http.MaxBytesReader(w, r.Body, MaxPayloadSize)
		if r.ContentLength > MaxPayloadSize {
			http.Error(w, "File too large", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}
