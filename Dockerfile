FROM golang:1.23.4-alpine@sha256:04ec5618ca64098b8325e064aa1de2d3efbbd022a3ac5554d49d5ece99d41ad5

COPY pihole-influx-exporter /pihole-influx-exporter

CMD ["/pihole-influx-exporter"]
