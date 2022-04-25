package main

import (
	"context"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spacelift-io/homework-object-storage/backend"
	"github.com/spacelift-io/homework-object-storage/handlers"
	"net/http"
	"os"
)

const NetworkNameEnvironmentVariable = "PRIVATE_NETWORK_NAME"
const BucketNameEnvironmentVariable = "BUCKET_NAME"
const ObjectPathParameter = "{id:[a-zA-Z0-9]{1,32}}"
const ContainerName = "amazin-object-storage-node"
const MinioUserEnvVariable = "MINIO_ROOT_USER"
const MinioPwdEnvVariable = "MINIO_ROOT_PASSWORD"

func main() {
	//TODO check for network name in env
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal("Can't connect to docker daemon.", err)
	}
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

	//FIXME
	err = servers.EndpointList(context.Background())

	if err != nil {
		log.Fatal("Can't connect to servers handler.", err)
	}
	log.Printf("Found %d servers servers", len(servers.Servers))

	r := mux.NewRouter()
	r.HandleFunc("/object/" + ObjectPathParameter, handlers.NewReadObjectHandler(servers)).Methods("GET")
	r.HandleFunc("/object/" + ObjectPathParameter, handlers.NewWriteObjectHandler(servers)).Methods("PUT")
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":3000", r))
}
