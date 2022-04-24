package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

func main() {
	const ContainerName = "amazin-object-storage-node"
	
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	//Filters for running, healthy containers with name amazin-object-storage-node-[0-9]
	filters := filters.NewArgs(
		filters.KeyValuePair{Key: "name", Value: ContainerName},
		filters.KeyValuePair{Key: "status", Value: "running"},
		//filters.KeyValuePair{Key: "health", Value: "healthy"}, //FIXME add healthchecks
		)
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{Limit: 10, Filters: filters})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		details, err := cli.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s\n", details.NetworkSettings)
	}
}
