package backend

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	log "github.com/sirupsen/logrus"
	"io"
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
		defaultExpiry: 5 * time.Minute,
		ip: ip,
	}

	userVar := fmt.Sprintf("%s=", MinioUserEnvVariable)
	pwdVar := fmt.Sprintf("%s=", MinioPwdEnvVariable)
	for _, config := range details.Config.Env {
		if strings.HasPrefix(config, userVar) {
			server.user = strings.Replace(config, userVar, "", 1)
		} else if strings.HasPrefix(config, pwdVar) {
			server.password = strings.Replace(config, pwdVar, "", 1)
		}
	}

	if err := server.createClient(); err != nil {
		return nil, err
	}

	return server, nil
}

func (s *MinioServer) createClient() error {
	minioClient, err := minio.New(fmt.Sprintf("%s:9000", s.ip), &minio.Options{
		Creds:  credentials.NewStaticV4(s.user, s.password, ""),
		Secure: false,
	})
	if err != nil {
		return err
	}
	s.Client = minioClient
	exists, err := s.Client.BucketExists(context.Background(), s.bucketName)
	if err != nil {
		return err
	}
	//Try to create bucket since it does not exists
	if !exists {
		log.Warnf("Bucket `%s` does not exists - creating", s.bucketName)
		err := s.Client.MakeBucket(context.Background(), s.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *MinioServer) PresignedGetObject(ctx context.Context, id string) (*url.URL, error) {
	return s.Client.PresignedGetObject(ctx, s.bucketName, id, s.defaultExpiry, nil)
}

func (s *MinioServer) PutObject(ctx context.Context, id string, reader io.Reader, size int64) error {
	uploadInfo, err := s.Client.PutObject(ctx, s.bucketName, id, reader, size, minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	log.Infof("Uploaded file `%s`", id)
	log.Info(uploadInfo)
	return nil
}




