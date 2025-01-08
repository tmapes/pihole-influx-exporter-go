FROM golang:1.23.4-alpine@sha256:94b4686346804a8cfc3d2c9069bbdf4bfb21ac8bdc6d99e23f529e295db1cd81

COPY pihole-influx-exporter /pihole-influx-exporter

CMD ["/pihole-influx-exporter"]
