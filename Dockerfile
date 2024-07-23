FROM golang:1.22.5-alpine@sha256:0e90634e2a706a24f1176483f7315e019738d53a188b7a57c1f58426999bb0ee

COPY pihole-influx-exporter /pihole-influx-exporter

CMD ["/pihole-influx-exporter"]
