FROM golang:1.23.4-alpine@sha256:13aaa4b92fd4dc81683816b4b62041442e9f685deeb848897ce78c5e2fb03af7

COPY pihole-influx-exporter /pihole-influx-exporter

CMD ["/pihole-influx-exporter"]
