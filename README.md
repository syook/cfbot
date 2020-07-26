# cfbot

## Overview [![PkgGoDev](https://pkg.go.dev/badge/github.com/x/syook/cfbot)](https://pkg.go.dev/github.com/x/syook/cfbot) [![Go Report Card](https://goreportcard.com/badge/github.com/syook/cfbot)](https://goreportcard.com/report/github.com/syook/cfbot)

CFbot is a CLI application for Cloudflare that helps you automate getting certificates from Cloudflare.

## Install

```
go get github.com/syook/cfbot
```

## Example

```
sudo cfbot --init --auth <cloudflare CA token> --hostnames "*.example.com,example.com"  --validity 7
```

### The above command does the following.

- Initializes all the necessary folder paths.
- Fetches the first set of certificates from cloudflare
- Saves the current config for further use
- Adds a cron which runs twice a day and gets new certificates if the existing ones are about expire in 48 hours.

## PS

The service needs sudo permissions to access the /etc directory and also to add the cron job under /etc/cron.d

## License

Apache 2.0.
