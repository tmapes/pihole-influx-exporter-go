FROM golang:1.23.4-alpine@sha256:d37127f39271451047bcd91fc53ee014829603c96b91d02ff65ab3a7d1fb3c5e

COPY pihole-influx-exporter /pihole-influx-exporter

CMD ["/pihole-influx-exporter"]
