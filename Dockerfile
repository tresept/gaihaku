FROM golang:1.24.5

# PostgreSQL
RUN apt-get update && apt-get install -y postgresql-client

WORKDIR /go/src

COPY . .