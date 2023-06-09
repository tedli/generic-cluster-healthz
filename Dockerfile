FROM golang:1.20.4-bullseye AS builder
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
COPY . /go/src/github.com/tedli/generic-cluster-healthz
WORKDIR /go/src/github.com/tedli/generic-cluster-healthz
RUN go build -v -mod=vendor -ldflags \
    "-w -s -X \"github.com/tedli/generic-cluster-healthz/pkg/version.CommitID=`git rev-parse HEAD`\" \
    -X \"github.com/tedli/generic-cluster-healthz/pkg/version.BuiltAt=`date -u +'%Y-%m-%dT%H:%M:%SZ'`\"" \
    -o probe.elf github.com/tedli/generic-cluster-healthz/cmd

FROM gcr.io/distroless/static-debian11:nonroot
COPY --from=builder /go/src/github.com/tedli/generic-cluster-healthz/probe.elf /usr/local/bin/probe
VOLUME ["/var/run/secrets/kubernetes.io/serviceaccount"]
EXPOSE 8080
ENTRYPOINT ["probe"]
