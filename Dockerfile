FROM golang:1.22 AS builder

WORKDIR /build
COPY . .
ENV CGO_ENABLED=0
RUN make build

# Copy the built binary and ca-certificates from the builder stage
FROM scratch
COPY --from=builder /build/cli/go-imapgrab /go-imapgrab
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set up volume for mail storage
VOLUME ["/maildir"]

# Set default UID and GID
USER 1000:1000

# Use the built go binary as entrypoint
ENTRYPOINT ["/go-imapgrab"]

# Set the default command to display help
CMD ["--help"]
