# Owtt - Open Web Torrent Tracker

**NOTE:** This project is still in development and is not ready for production use.

Owtt is a simple, open source, web torrent tracker. It's written in go and was heavily inspired by another [similar project](https://github.com/OpenWebTorrent/openwebtorrent-tracker). Main difference is that Owtt is multi-threaded and has a few more features.

## Build

```bash
go build
```

## Usage

```bash
A go based webtorrent tracker

Usage:
  owtt [flags]

Flags:
  -a, --announce-interval int   Announce interval (default 15)
  -d, --debug                   Debug
  -h, --help                    help for owtt
  -m, --max-offers int          Max offers (default 20)
  -p, --port int                Port (default 8000)
  -l, --rate-limit int          Rate limit (default 10)
  -e, --rate-limit-expiry int   Rate limit expiration in seconds (default 2)
  -r, --reverse-proxy           Reverse proxy
  -c, --ssl-cert string         SSL certificate
  -k, --ssl-key string          SSL key
```

Stats are available at `/stats.json`.

### License

MIT