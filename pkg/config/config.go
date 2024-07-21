package config

import (
	"github.com/urfave/cli/v2"
)

type AppConfig struct {
	PiHoleConfig
	InfluxConfig
}

type InfluxConfig struct {
	BaseUrl               string
	Bucket                string
	EnableGzipCompression bool
	HostTag               string
	Org                   string
	Token                 string
}

type PiHoleConfig struct {
	BaseUrl string
	Token   string
}

func New(c *cli.Context) AppConfig {

	piHoleHostTag := c.String("influx.pi_hole_host")
	if piHoleHostTag == "" {
		piHoleHostTag = c.String("pihole.base_url")
	}

	return AppConfig{
		PiHoleConfig{
			BaseUrl: c.String("pihole.base_url"),
			Token:   c.String("pihole.token"),
		},
		InfluxConfig{
			BaseUrl:               c.String("influx.base_url"),
			Bucket:                c.String("influx.bucket"),
			EnableGzipCompression: c.Bool("influx.gzip"),
			HostTag:               piHoleHostTag,
			Org:                   c.String("influx.org"),
			Token:                 c.String("influx.token"),
		},
	}

}

func (c AppConfig) Valid() bool {
	return c.PiHoleConfig.valid() && c.InfluxConfig.valid()
}

func (i PiHoleConfig) valid() bool {
	return i.BaseUrl != "" && i.Token != ""
}

func (i InfluxConfig) valid() bool {
	return i.BaseUrl != "" && i.Token != "" && i.HostTag != ""
}
