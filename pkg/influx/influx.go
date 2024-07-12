package influx

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"os"
	"strings"
	"time"
)

func NewClient() (influxdb2.Client, api.WriteAPI) {
	var influxUrl string
	var token string
	var org string
	var bucket string
	var gzipCompression = true
	var ok bool
	if token, ok = os.LookupEnv("INFLUX_TOKEN"); !ok {
		panic("INFLUX_TOKEN not set!")
	}
	if org, ok = os.LookupEnv("INFLUX_ORG"); !ok {
		panic("INFLUX_ORG not set!")
	}
	if bucket, ok = os.LookupEnv("INFLUX_BUCKET"); !ok {
		panic("INFLUX_BUCKET not set!")
	}
	if influxUrl, ok = os.LookupEnv("INFLUX_URL"); !ok {
		influxUrl = "http://localhost:8086"
	}
	if gzipCompressionValue, ok := os.LookupEnv("INFLUX_ENABLE_GZIP"); ok {
		gzipCompression = strings.EqualFold("true", gzipCompressionValue)
	}
	options := influxdb2.DefaultOptions().
		SetUseGZip(gzipCompression).
		AddDefaultTag("pi_hole_host", determineHostTag()).
		SetFlushInterval(uint((5 * time.Second).Milliseconds()))

	client := influxdb2.NewClientWithOptions(influxUrl, token, options)
	return client, client.WriteAPI(org, bucket)
}

func determineHostTag() string {
	if hostTag, set := os.LookupEnv("PI_HOLE_HOST_TAG"); set {
		return hostTag
	}
	piHoleHost, set := os.LookupEnv("PI_HOLE_HOST")
	if !set {
		return "http://localhost:8086"
	}
	return piHoleHost
}
