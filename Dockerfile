FROM golang:1.15
WORKDIR /mnt/homework
COPY . .
ENV MINIO_VERSION=RELEASE.2022-04-16T04-26-02Z
ENV PRIVATE_NETWORK_NAME=amazin-object-storage
ENV MINIO_API_SERVER_PORT=9000
ENV BUCKET_NAME=somebucket
RUN export $(cat .env | xargs)
ENTRYPOINT ["go", "run", "main.go"]
