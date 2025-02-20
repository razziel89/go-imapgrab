# Usage Examples

This page showcases some potential real-word uses of `go-imapgrab`.

## Docker Compose

It can be desirable to run `go-imapgrab` via docker compose.
Below is an example `compose.yaml` that shows how to download emails from an
IMAP server to a local directory using `go-imapgrab`.
You can create one such file for each mailbox you wish to download.
All values that are enclosed in double quotes `"` in the below example have to
be adjusted for each mailbox.
You could use [environment variable interpolation] to create a single
`compose.yaml` file that is then parametrised via environment variables.
For simplicity, all values apart from the password are hard-coded in the below
example `compose.yaml` file, though.

Note that you must create the local target folder for your maildir before
running `docker compose`, otherwise it will be created as owned by the system's
super user `root`, which would make it inaccessible to any other user.
The example below executes `go-imapgrab` with user ID and group ID both set to
`1000`, but those values have to be adjusted.
The user referred to by the specified user ID and group ID must have write
access to the local folder used for downloads.
You can find out your own user ID by running `id -u`.
You can find out your own group ID by running `id -g`.
It is recommended to replace both user ID and group ID by the value of your own
unprivileged user account.

The below example would download emails from a server reachable as
`imap.example.com` for a user `john@example.com`.
The maildir and the files within it will be owned by user ID 1000 and group ID
1000, which is the first default user on many Linux distributions.
The password for the mailbox is located in a file `igrab_password.txt` located
in the same directory as the `compose.yaml` file.
All folders will be downloaded as indicated by the `_ALL_` folder specification.
Emails will be downloaded to a directory called `maildir` that is located in the
same directory as the `compose.yaml` file.

Once you have created a suitable `compose.yaml` file and put the password in a
file `igrab_password.txt` next to it, you can trigger the download of your
emails by running `docker compose up`.

```yaml
---
services:
  go-imapgrab:
    image: ghcr.io/razziel89/go-imapgrab:latest
    volumes:
      - "./maildir:/maildir"
    environment:
      IGRAB_PASSWORD: /run/secrets/IGRAB_PASSWORD
    secrets:
      - IGRAB_PASSWORD
    command:
      - download
      - --no-keyring
      - --server
      - "imap.example.com"
      - --user
      - "john@example.com"
      - --folder
      - "_ALL_"
      - --path
      - /maildir
      - --verbose
    user: "1000:1000"
    restart: no
secrets:
  IGRAB_PASSWORD:
    file: "igrab_password.txt"
```

[environment variable interpolation]: https://docs.docker.com/compose/how-tos/environment-variables/variable-interpolation/
