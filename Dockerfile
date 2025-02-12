# Stage 1: Build the go-imapgrab binary
FROM golang:1.22 AS builder

# Set the working directory
WORKDIR /build

# Clone the go-imapgrab repository
COPY . .
#RUN git clone https://github.com/razziel89/go-imapgrab.git .

# Build the go-imapgrab binary
#ENV CGO_ENABLED=0
RUN make build

# Stage 2: Create a minimal runtime image
#FROM debian:bookworm-slim
#RUN apt-get update && apt-get install -y \
#  ca-certificates \
#  && rm -rf /var/lib/apt/lists/*

# Copy the built binary and ca-certificates from the builder stage
FROM scratch
COPY --from=builder /build/cli/go-imapgrab /go-imapgrab
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set up volume for mail storage
VOLUME ["/maildir"]

# Set defulat UID and GID
USER 1000:1000

# Use the built go binary as entrypoint
ENTRYPOINT ["/go-imapgrab"]

# Set the default command to display help
CMD ["--help"]
