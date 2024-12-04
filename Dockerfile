FROM golang:1.23.4-alpine@sha256:29c74ca0344a4da5fbf0003f31812d47b72db3551820d6d3642937d247cba5bf

COPY pihole-influx-exporter /pihole-influx-exporter

CMD ["/pihole-influx-exporter"]
