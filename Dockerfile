FROM golang:1.15-alpine3.13 AS builder

ARG DRONE_TAG=""
ARG DRONE_COMMIT=""

WORKDIR /go/src/drone-admit-members
COPY . /go/src/drone-admit-members

RUN go get -d -v ./... \
  && CGO_ENABLED=0 GOOS=linux go build \
     -a -ldflags "-X main.version=${DRONE_TAG} -X main.rev=${DRONE_COMMIT}" \
     -o /go/bin/drone-admit-members


FROM gcr.io/distroless/base-debian10

COPY --from=builder /go/bin/drone-admit-members /
WORKDIR /

EXPOSE 3000

ENTRYPOINT ["/drone-admit-members"]
