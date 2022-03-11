# General

This is `go-imapgrab`, a re-implementation of the amazing [`imapgrab`][imapgrab]
in plain Golang.
It is a command line application that can be used to make a local backup of your
IMAP mailboxes.
See [below](#how-to-use) for how to use.

This software is in beta state but has been used by the main author to backup
quite a few mailboxes already.
Development started in early 2022.
Contributions are very welcome (see below)!

## Notable features

- download IMAP mailboxes to a local directory following the
  [`maildir`][maildir] format
- static binary without any additional dependencies
- optional support for system keyring to store credentials securely
- maildir output fully compatible to the original [`imapgrab`][impagrab] (please
  open an issue in this repository if you notice incompatibilities)

## Currently absent features

- output in other formats such as `mbox`
- password specification via command line argument (use environment variable or
  keyring for now)
- more than two verbosity levels
- `version` and `about` commands
- disabling SSL for connections
- change user for download (cf. [`imapgrab`][imapgrab]'s `--localuser` flag) to
  be runnable as `root` (use `sudo` instead to change the user for one
  invocation)
- restoration of a local backup to a server
- view downloaded emails (use `mutt -f <PATH>` for that)
- local removal of emails that have been removed remotely

# Why a re-implementation

The main author had been using the original [`imapgrab`][imapgrab] successfully
for quite a while to backup mailboxes.
However, the original is now abandoned and implemented in the deprecated Python
version 2.
While there are re-implementations in Python 3, none of them appeared quite
complete at the time this project started.

In addition to the above, [`imapgrab`][imapgrab] uses the [`getmail`][getmail]
executable to download emails.
That executable is also written in Python 2 and lacked a complete
re-implementation in Python 3at the time this project started.

Furthermore, the author had started learning Golang not too long before this
project started.
Golang is a language that provides static binaries while supporting
cross-compilation natively.
One advantage of such a setup is that it is very easy to provide executables for
systems that would run [`imapgrab`][imapgrab] or [`getmail`][getmail] only with
difficulty.

Still, this project would not have been possible without the amazing tools
[`imapgrab`][imapgrab] and [`getmail`][getmail], which provided a very solid
basis for `go-imapgrap`.
Thank you!

# How to use

The repo does not yet have binary distributions (but they are planned).
Thus, for now, the first step is to clone this repository and compile the CLI
yourself.
See [installation](#installation) below for details.

Once you have the executable, run `./imapgrab --help` to see whether it works.
Please open an issue in this repository if you experience problems!

Then, list the folders in your mailbox and download the ones you wish to backup.

## List folders

Usually, the first step is to list the folders available in your mailbox.
To do so, run:

```bash
IGRAB_PASSWORD=<PASSWORD> ./imapgrab list -u <USERNAME> -s <SERVER> -p <PORT>
```

The specification of the port it optional, it defaults to 993.
Refer to the documentation of your email provider for the username, server, and
port.
For Gmail, you can:

- use an [application-specific password][gmail-app-password] for `<PASSWORD>`
- provide your email address as `<USERNAME>`
- use `imap.gmail.com` as `<SERVER>`
- leave out the port since Gmail uses the default one

The password needs to be specified only the first time via an environment
variable.
Ever call after the first one will use the system's keyring if you do not
provide a password.
To disable the keyring, for example if you experience problems, add the
`--no-keyring` flag.
You will need to provide your password via the environment variable in that
case.

Once you see your list of folders, decide which ones you want to download and
proceed with the `download` command (see below).

To see the full specification for the `list` command, run:

```bash
./imapgrab list --help
```

## Download

The second step is to download the folders you want.
For example, to download all folders apart from Gmail-specific ones and the
`Drafts` directory, you can run:

```bash
./imapgrab download -u <USERNAME> -s <SERVER> -p <PORT> \
    -f _ALL_ -f -_Gmail_ -f -Drafts --path <PATH>
```

For the first run for a mailbox, specify for `<PATH>` a non-existing or empty
directory.
This is where you will download all folders for this mailbox to.
A non-existing directory will be created first, including all parents.

The above command will result in one directory per folder in `<PATH>` in
addition to one meta data file per folder that must not be modified or updates
won't work.
For every run after the first, specify the very same `<PATH>` if you want to
download only missing emails.

As you can see in the above command, you can provide multiple folder
specifications via the `-f` or `--folder` flag.
They are evaluated in order.
A folder specification is either

- a literal folder name such as `Drafts` in the above example
- the literal string `_ALL_` to specify all folders
- the literal string `_Gmail_` to specify all Gmail-specific folders

A folder specification can optionally start with a minus sign (`-`), in which
case it negates the specification.
Thus, `-Drafts` in the above example deselects that folder, while `_ALL_`
selects all folders first.

These folder specification have been taken from [`imapgrab`][imapgrab].
In contrast, though, folders are not separated by commas but the `--folder` flag
is provides several times instead.

To see the full specification for the `download` command, run:

```bash
./imapgrab download --help
```

# Installation

Currently, you need to build the binary yourself.
First, install a [Golang toolchain](https://go.dev/doc/install).
Then, run the following in a terminal:

```bash
git clone https://github.com/razziel89/go-imapgrab
cd go-imapgrab/cli
go build -o imapgrab .
```

Downloadable binary distributions will follow.

# How to contribute

If you have found a bug and want to fix it, please simply go ahead and fork the
repository, fix the bug, and open a pull request to this repository!
Bug fixes are always welcome.

In all other cases, please open an issue on GitHub first to discuss the
contribution.
The feature you would like to introduce might already be in development.

# Licence

[GPLv3](./LICENCE)

If you want to use this piece of software under a different, more permissive
open-source licence, please contact me.
I am very open to discussing this point.

[imapgrab]: https://sourceforge.net/p/imapgrab/wiki/Home/ "imapgrab website"
[maildir]: https://cr.yp.to/proto/maildir.html "maildir format"
[getmail]: https://pyropus.ca./software/getmail/ "getmail website"
[gmail-app-password]: https://support.google.com/accounts/answer/185833?hl=en "application-specific passwords"
