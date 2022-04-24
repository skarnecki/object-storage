package backend

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"net/url"
	"strings"
	"time"
)

type  MinioServer struct{
	ip       string
	user     string
	password string
	bucketName string
	defaultExpiry time.Duration
	Client   *minio.Client
}

func NewMinioServer(details types.ContainerJSON, ip string, bucketName string) (*MinioServer, error) {
	server := &MinioServer{
		bucketName: bucketName,
		ip: ip,
	}

	for _, config := range details.Config.Env {
		if strings.HasPrefix(config, fmt.Sprintf("%s=", MinioUserEnvVariable)) {
			server.user = ""
		} else if strings.HasPrefix(config, fmt.Sprintf("%s=", MinioPwdEnvVariable)) {
			server.password = ""
		}
	}

	if err := server.createClient(); err != nil {
		return nil, err
	}

	return server, nil
}

func (s MinioServer) createClient() error {
	minioClient, err := minio.New(s.ip, &minio.Options{
		Creds:  credentials.NewStaticV4(s.user, s.password, ""),
		Secure: false,
	})
	if err != nil {
		return err
	}
	s.Client = minioClient
	return nil
}

func (s MinioServer) PresignedGetObject(ctx context.Context, id string) (*url.URL, error) {
	return s.Client.PresignedGetObject(ctx, s.bucketName, id, s.defaultExpiry, nil)
}




