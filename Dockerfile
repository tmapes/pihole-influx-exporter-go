FROM golang:1.22.5-alpine@sha256:63be73fdea9899269e98a4ad8fdebbdba6819bd7d30eae97726739a548448541

COPY pihole-influx-exporter /pihole-influx-exporter

CMD ["/pihole-influx-exporter"]
