FROM golang:1.15
WORKDIR /mnt/homework
#TODO build binary and use it instead of copying source code
COPY . .
ENV MINIO_VERSION=RELEASE.2022-04-16T04-26-02Z
ENV PRIVATE_NETWORK_NAME=amazin-object-storage
ENV MINIO_API_SERVER_PORT=9000
ENV BUCKET_NAME=somebucket
ENV HTTP_PORT=3000
RUN GOARCH=amd64 GOOS=linux go build .
RUN chmod +x homework-object-storage
