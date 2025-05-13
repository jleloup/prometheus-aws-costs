########################
# Prometheus AWS Costs #
########################

FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
ARG TARGETARCH

RUN apk update && apk add --no-cache git

ENV USER=appuser
ENV UID=10001

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR $GOPATH/src/prometheus-aws-costs/app/

COPY go.mod ./
COPY go.sum ./

# RUN go mod download and verify
RUN go mod download \
    && go mod verify

COPY . ./

# Fetch dependencies.
# Using go get.
RUN go get -d -v . \
    # Build the binary.
    && CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-w -s" -tags="viper_bind_struct" -o  /go/bin/prometheus-aws-costs main.go \
    && chmod 755 /go/bin/prometheus-aws-costs \
    && chown ${USER}:${USER} /go/bin/prometheus-aws-costs

FROM scratch
# Import the user and group files from the builder.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
# Copy our static executable.
COPY --from=builder /go/bin/prometheus-aws-costs /go/bin/prometheus-aws-costs

# Use an unprivileged user.
USER ${UID}

ENTRYPOINT ["/go/bin/prometheus-aws-costs"]
