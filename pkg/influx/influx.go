package influx

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"log"
	"pihole-influx-exporter-go/pkg/config"
	"time"
)

func NewClient(c config.InfluxConfig) (influxdb2.Client, api.WriteAPI) {
	options := influxdb2.DefaultOptions().
		SetUseGZip(c.EnableGzipCompression).
		AddDefaultTag("pi_hole_host", c.HostTag).
		SetFlushInterval(uint((5 * time.Second).Milliseconds())).
		SetApplicationName("pihole-influx-exporter")

	client := influxdb2.NewClientWithOptions(c.BaseUrl, c.Token, options)
	log.Printf("Influx Config | BaseUrl: '%s', Org: '%s', Bucket: '%s'", c.BaseUrl, c.Org, c.Bucket)
	return client, client.WriteAPI(c.Org, c.Bucket)
}
