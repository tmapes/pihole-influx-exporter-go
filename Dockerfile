FROM golang:1.23.0-alpine@sha256:ac8b5667d9e4b3800c905ccd11b27a0f7dcb2b40b6ad0aca269eab225ed5584e

COPY pihole-influx-exporter /pihole-influx-exporter

CMD ["/pihole-influx-exporter"]
