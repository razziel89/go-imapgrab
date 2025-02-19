# Docker Compose

The included `docker-compose.yml` file provides an example of how to download
emails from an IMAP server to a local directory using `go-imapgrab` with docker.
It is configurable using environment variables which can be provided via a
`.env` file in the same directory as your `docker-compose.yml`.
An example `.env` is provided as `.env.example`.

Note that you should create the local folder for your maildir files before
running `docker compose`, otherwise the docker daemon (which runs as `root`)
will create the folder, and it will be therefore be owned by `root`.
By default, the `docker-compose.yml` executes `go-imapgrab` with UID and GID
both set to `1000`, but these can be set via `IGRAB_UID` and/or `IGRAB_GID`
variables.
This user must have write access to the local folder used for downloads.

### Build

```sh
docker compose build
```

### Run

```sh
docker compose run --rm go-imapgrab
```

Or add the `--build` option to run and build with a single command:

```sh
docker compose run --build --rm go-imapgrab
```
