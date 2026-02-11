FROM golang:1.26-bookworm AS builder
RUN \
  apt-get update && \
  DEBIAN_FRONTEND=noninteractive apt-get install -y make ca-certificates
WORKDIR /app
COPY cli/go.* ./cli/
COPY core/go.* ./core/
RUN cd /app/cli && go mod download && cd /app/core && go mod download
COPY Makefile ./
COPY cli/*.go cli/Makefile ./cli/
COPY core/*.go core/Makefile ./core/
RUN make build

FROM scratch
LABEL org.opencontainers.image.source=https://github.com/razziel89/go-imapgrab
LABEL org.opencontainers.image.description="A re-implementation of the amazing imapgrab in plain Golang. "
LABEL org.opencontainers.image.licenses=GPLv3
COPY --from=builder /app/cli/go-imapgrab /go-imapgrab
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
VOLUME ["/maildir"]
USER 1000:1000
ENTRYPOINT ["/go-imapgrab"]
