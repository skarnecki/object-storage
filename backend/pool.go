package backend

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	"github.com/spacelift-io/homework-object-storage/config"
	"hash/fnv"
	"io"
	"sort"
	"strings"
	"time"
)

//go:generate mockgen -source=$GOFILE -destination=$PWD/mocks/${GOFILE} -package=mocks
type DockerClient interface {
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
}

type Pool struct {
	BucketName           string
	Servers              map[string]*MinioServer // map[serverIp]MinioServer"
	cli                  *client.Client
	networkName          string
	containerName        string
	minioUserEnvVariable string
	minioPwdEnvVariable  string
}

func NewPool(cli *client.Client, config config.Config) (*Pool, error) {
	networkName, err := verifyNetwork(context.Background(), cli, config.NetworkName)
	if err != nil {
		return nil, err
	}
	serversMap := make(map[string]*MinioServer)
	pool := &Pool{
		BucketName:           config.BucketName,
		cli:                  cli,
		networkName:          networkName,
		containerName:        config.ContainerName,
		minioUserEnvVariable: config.MinioUser,
		minioPwdEnvVariable:  config.MinioPwd,
		Servers:              serversMap,
	}

	autoRefresh(context.Background(), config.RefreshInterval, pool)
	return pool, pool.EndpointList(context.Background())
}

func (b Pool) EndpointList(ctx context.Context) error {
	containers, err := b.getContainerList(ctx)
	if err != nil {
		return err
	}

	newServerIps := make(map[string]bool, len(containers))
	for _, container := range containers {
		ip, err := b.appendServers(container)
		if err != nil {
			return err
		}
		newServerIps[ip] = true
	}

	b.clearRemoved(newServerIps)

	if len(b.Servers) < 1 {
		return fmt.Errorf("no running Minio containers found")
	}

	return nil
}

func (b Pool) FindServerIPById(id string) string {
	h := fnv.New32a()
	h.Write([]byte(id))

	numberOfServers := len(b.Servers)
	log.Debugf("Hash calculated: %d, Servers available: %d", h.Sum32(), numberOfServers)
	serverNumber := h.Sum32() % uint32(numberOfServers)
	log.Debugf("Server no %d", serverNumber)


	keys := b.getServerIps()
	sort.Strings(keys)
	hashedServerIp := keys[serverNumber]

	log.Debugf("Hash pointed to `%s`", hashedServerIp)
	return hashedServerIp
}

func (b Pool) getServerIps() []string {
	keys := make([]string, 0, len(b.Servers))
	for k := range b.Servers {
		keys = append(keys, k)
	}
	return keys
}

func (b Pool) FindObjectServer(ctx context.Context, id string) (*MinioServer, error) {
	hashedServerIp := b.FindServerIPById(id)

	//Attempt to look for object in server pointed by hash
	err, done := b.checkForObject(ctx, id, b.Servers[hashedServerIp])
	if done {
		return b.Servers[hashedServerIp], err
	}

	log.Debugf("Id: %s not found on %s, attempting to find on other servers", id, hashedServerIp)
	//Hashed server does not have the object, look for other
	for idx, server := range b.Servers {
		if idx == hashedServerIp {
			//Server that we already checked
			continue
		}
		err, done := b.checkForObject(ctx, id, server)
		if done {
			log.Debugf("Id: %s found on %s", id, idx)
			return server, err
		}
	}

	//No servers containing object, no errors = 404
	return nil, nil
}

func (b Pool) checkForObject(ctx context.Context, id string, server *MinioServer) (error, bool) {
	objDetails, err := server.Client.StatObject(ctx, b.BucketName, id, minio.StatObjectOptions{})

	if err != nil && !checkIfObjectNotExistError(err) {
		return err, true
	}

	if objDetails.ETag != "" {
		return nil, true
	}
	return nil, false
}

func (b Pool) PutObject(ctx context.Context, id string, reader io.Reader, objectSize int64) error {
	serverIp := b.FindServerIPById(id)

	if err := b.Servers[serverIp].PutObject(ctx, id, reader, objectSize); err != nil {
		return err
	}
	return nil
}

func (b Pool) getContainerList(ctx context.Context) ([]types.Container, error) {
	//Filters for running, healthy containers with name amazin-object-storage-node-[0-9]
	containerFilters := filters.NewArgs(
		filters.KeyValuePair{Key: "name", Value: b.containerName},
		filters.KeyValuePair{Key: "status", Value: "running"},
		//containerFilters.KeyValuePair{Key: "health", Value: "healthy"}, //FIXME add health checks
	)
	containers, err := b.cli.ContainerList(ctx, types.ContainerListOptions{Limit: 10, Filters: containerFilters})
	if err != nil {
		return nil, err
	}
	return containers, nil
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


func (b Pool) appendServers(container types.Container) (string, error) {
	details, err := b.cli.ContainerInspect(context.Background(), container.ID)
	if err != nil {
		return "", err
	}

	ip := details.NetworkSettings.Networks[b.networkName].IPAddress
	minio, err := b.createServer(details, ip, b.BucketName)
	if err != nil {
		//If one of the servers are miss configured we should log it and try others
		log.Warn(err)
		return "", nil
	}
	b.Servers[ip] = minio
	return ip, nil
}

func (b Pool) clearRemoved(newIps map[string]bool) {
	for ip := range b.Servers {
		if _, ok := newIps[ip]; !ok {
			log.Warnf("Removing %s since offline", ip)
			delete(b.Servers, ip)
		}
	}
}

func checkIfObjectNotExistError(err error) bool {
	errResp := minio.ToErrorResponse(err)
	return "NoSuchKey" == errResp.Code
}

//Check if provided network name is present
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

func autoRefresh(ctx context.Context, RefreshInterval time.Duration, servers *Pool) {
	//Schedule every `RefreshInterval` server list refresh
	go func() {
		for _ = range time.Tick(RefreshInterval) {
			err := servers.EndpointList(ctx)
			if err != nil {
				log.Fatal("Can't connect to servers handler.", err)
			}
			log.Debugf("Found %d servers servers", len(servers.Servers))
		}
	}()
}