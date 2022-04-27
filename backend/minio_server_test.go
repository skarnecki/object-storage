package backend

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/spacelift-io/homework-object-storage/backend/mocks"
	"strings"
	"testing"
)

const MockedBucket = "mockedbucket"

func TestCreateClient_NoClient(t *testing.T) {
	server, err := NewMinioServer("stub", "bucketname", "user", "password")
	if server != nil {
		t.Errorf("Returned server object")
	}
	if err == nil {
		t.Errorf("Returned no error when creating client")
	}
	if !strings.Contains(err.Error(), "no such host") {
		t.Errorf("Got %s instead of \"no such host\"", err.Error())
	}
}

func TestCreateClient_InitBucketWhenExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	server := mockedServer(ctrl, true, nil, nil, false)
	defer ctrl.Finish()
	err := server.initBucket()
	if err != nil {
		t.Error("Bucket check error", err)
	}
}

func TestCreateClient_InitBucketWhenDoesNotExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	server := mockedServer(ctrl, false, nil, nil, true)
	defer ctrl.Finish()

	err := server.initBucket()
	if err != nil {
		t.Error("Bucket check error", err)
	}
}

func TestCreateClient_InitBucketWhenDoesNotExistsMakeBucketError(t *testing.T) {
	ctrl := gomock.NewController(t)
	server := mockedServer(ctrl, false, nil, fmt.Errorf("some-error"), true)
	defer ctrl.Finish()

	err := server.initBucket()
	if err.Error() != "some-error" {
		t.Error("Bucket check error", err)
	}
}

func TestCreateClient_InitBucketWhenError(t *testing.T) {
	ctrl := gomock.NewController(t)
	server := mockedServer(ctrl, false, fmt.Errorf("someerror"), nil, false)
	defer ctrl.Finish()
	err := server.initBucket()
	if err.Error() != "someerror" {
		t.Error("Bucket check error", err)
	}
}

func mockedServer(ctrl *gomock.Controller, exists bool, resultErr error, makeErr error, withMake bool) *MinioServer {
	m := mocks.NewMockMinioClient(ctrl)
	m.EXPECT().
		BucketExists(gomock.Any(), gomock.Eq(MockedBucket)).
		Return(exists, resultErr)

	if withMake {
		m.EXPECT().
			MakeBucket(gomock.Any(), gomock.Eq(MockedBucket), gomock.Any()).
			Return(makeErr)

	}

	server := &MinioServer{
		ip:            "",
		user:          "",
		password:      "",
		bucketName:    MockedBucket,
		defaultExpiry: 0,
		Client:        m,
	}
	return server
}
