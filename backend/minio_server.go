package backend

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	log "github.com/sirupsen/logrus"
	"io"
	"net/url"
	"time"
)

//go:generate mockgen -source=$GOFILE -destination=$PWD/mocks/${GOFILE} -package=mocks
type MinioClient interface {
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	PresignedGetObject(ctx context.Context, bucketName string, objectName string, expires time.Duration, reqParams url.Values) (u *url.URL, err error)
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) (err error)
	StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
}

type MinioServer struct {
	ip            string
	user          string
	password      string
	bucketName    string
	defaultExpiry time.Duration
	Client        MinioClient
}

func NewMinioServer(ip string, bucketName string, user string, password string) (*MinioServer, error) {
	server := &MinioServer{
		bucketName:    bucketName,
		defaultExpiry: 5 * time.Minute,
		ip:            ip,
		user:          user,
		password:      password,
	}

	if err := server.createClient(); err != nil {
		return nil, err
	}

	if err := server.initBucket(); err != nil {
		return nil, err
	}

	return server, nil
}

//Connecting to minio server and ensure bucket exist
func (s *MinioServer) createClient() error {
	minioClient, err := minio.New(
		fmt.Sprintf("%s:9000", s.ip),
		&minio.Options{
			Creds:  credentials.NewStaticV4(s.user, s.password, ""),
			Secure: false,
		})

	if err != nil {
		return err
	}
	client := MinioClient(minioClient)
	s.Client = client

	return nil
}

func (s *MinioServer) initBucket() error {
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
	log.Debug(uploadInfo)
	return nil
}
