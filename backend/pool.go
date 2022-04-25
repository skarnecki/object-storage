package backend

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	"hash/fnv"
	"io"
	"sort"
	"strings"
)

type Pool struct {
	cli         *client.Client
	networkName string
	BucketName string
	containerName string
	minioUserEnvVariable string
	minioPwdEnvVariable string
	Servers map[string]*MinioServer
}

func NewPool(ctx context.Context, cli *client.Client, networkName string, bucketName string, containerName string,
	minioUserEnvVariable string,
	minioPwdEnvVariable string) (*Pool, error) {
	networkName, err := verifyNetwork(ctx, cli, networkName)
	if err != nil {
		return nil, err
	}
	serversMap := make(map[string]*MinioServer)
	return &Pool{
		BucketName: bucketName,
		cli: cli,
		networkName: networkName,
		containerName: containerName,
		minioUserEnvVariable: minioUserEnvVariable,
		minioPwdEnvVariable: minioPwdEnvVariable,
		Servers: serversMap,
	}, nil
}


func (b Pool) EndpointList(ctx context.Context) error {
	//Filters for running, healthy containers with name amazin-object-storage-node-[0-9]
	containerFilters := filters.NewArgs(
		filters.KeyValuePair{Key: "name", Value: b.containerName},
		filters.KeyValuePair{Key: "status", Value: "running"},
		//containerFilters.KeyValuePair{Key: "health", Value: "healthy"}, //FIXME add health checks
	)
	containers, err := b.cli.ContainerList(ctx, types.ContainerListOptions{Limit: 10, Filters: containerFilters})
	if err != nil {
		return err
	}

	for _, container := range containers {
		details, err := b.cli.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			return err
		}

		ip := details.NetworkSettings.Networks[b.networkName].IPAddress
		minio, err := b.createServer(details, ip, b.BucketName)
		if err != nil {
			//If one of the servers are miss configured we should log it and try others
			log.Warn(err)
			continue
		}
		b.Servers[ip] = minio
	}

	if len(b.Servers) < 1 {
		return fmt.Errorf("no running Minio containers found")
	}

	b.sortServers()

	return nil
}

func (b Pool) createServer(details types.ContainerJSON, ip string, bucketName string) (*MinioServer, error) {
	userVar := fmt.Sprintf("%s=", b.minioUserEnvVariable)
	pwdVar := fmt.Sprintf("%s=", b.minioPwdEnvVariable)
	user := ""
	password := ""
	for _, config := range details.Config.Env {
		if strings.HasPrefix(config, userVar) {
			user = strings.Replace(config, userVar, "", 1)
		} else if strings.HasPrefix(config, pwdVar) {
			password = strings.Replace(config, pwdVar, "", 1)
		}
	}
	return NewMinioServer(ip, bucketName, user, password)
}

func (b Pool) FindServerIPById(id string) string {
	h := fnv.New32a()
	h.Write([]byte(id))

	numberOfServers := len(b.Servers)
	log.Infof("Hash calculated: %d, Servers available: %d", h.Sum32(), numberOfServers)
	serverNumber := h.Sum32() % uint32(numberOfServers)
	log.Infof("Server no %d", serverNumber)
	keys := make([]string, 0, len(b.Servers))

	for k := range b.Servers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	log.Infof("Keys %d", keys)
	hashedServerIp := keys[serverNumber]

	log.Infof("Hash pointed to `%s`", hashedServerIp)
	return hashedServerIp
}

func (b Pool) FindObjectServer(ctx context.Context, id string) (*MinioServer, error) {
	hashedServerIp := b.FindServerIPById(id)
	objDetails, err := b.Servers[hashedServerIp].Client.StatObject(ctx, b.BucketName, id, minio.StatObjectOptions{})

	//TODO check if 404 is an error
	if err != nil && !checkIfObjectNotExistError(err) {
		return nil, err
	}

	if objDetails.ETag != "" {
		return b.Servers[hashedServerIp], nil
	}

	//Hashed server does not have the object, look for other
	for idx, server := range b.Servers {
		if idx == hashedServerIp {
			//Server that we already checked
			continue
		}
		_, err := server.Client.StatObject(ctx, b.BucketName, id, minio.StatObjectOptions{})
		if err != nil && !checkIfObjectNotExistError(err) {
			return nil, err
		}
		if objDetails.ETag != "" {
			return server, nil
		}
	}

	//No servers containing object, no errors = 404
	return nil, nil
}

func (b Pool) PutObject(ctx context.Context, id string, reader io.Reader, objectSize int64) error {
	serverIp := b.FindServerIPById(id)
	err := b.Servers[serverIp].PutObject(ctx, id, reader, objectSize)
	if err != nil {
		return err
	}
	return nil
}

func (b *Pool) sortServers() {
	newOrder :=  make(map[string]*MinioServer)
	keys := make([]string, 0, len(b.Servers))

	for k := range b.Servers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		newOrder[k] = b.Servers[k]
	}

	b.Servers = newOrder
}

func checkIfObjectNotExistError(err error) bool {
	errResp := minio.ToErrorResponse(err)
	return "NoSuchKey" == errResp.Code
}


func verifyNetwork(ctx context.Context, cli *client.Client, networkName string) (string, error) {
	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{Filters: filters.NewArgs(filters.KeyValuePair{"name", networkName})})
	if err != nil {
		return "", err
	}

	//We should find exactly one network
	if len(networks) != 1 {
		return "", fmt.Errorf("no shared network found Minio: %s", networkName)
	}
	return networks[0].Name, nil
}
