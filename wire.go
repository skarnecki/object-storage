//+build wireinject

package main

import (
	"github.com/docker/docker/client"
	"github.com/google/wire"
	"github.com/gorilla/mux"
	"github.com/spacelift-io/homework-object-storage/config"
)

func InitializeApp(config config.Config, ops ...client.Opt) (*mux.Router, error) {
	wire.Build(diContainer)
	return &mux.Router{}, nil
}
