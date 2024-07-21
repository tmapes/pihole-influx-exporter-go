package main

import (
	"errors"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"os/signal"
	"pihole-influx-exporter-go/pkg/config"
	"pihole-influx-exporter-go/pkg/influx"
	"pihole-influx-exporter-go/pkg/pihole"
	"strings"
	"syscall"
	"time"
)

func main() {
	app := cli.NewApp()

	app.Action = run
	app.Name = "pihole-influx-exporter"

	app.Flags = []cli.Flag{

		// PiHole Flags
		&cli.StringFlag{
			EnvVars:  []string{"PI_HOLE_TOKEN"},
			FilePath: "/run/secrets/pi_hole_token,/pi_hole_token",
			Name:     "pihole.token",
			Usage:    "set token used for making PiHole API requests",
			Required: true,
		},
		&cli.StringFlag{
			EnvVars:  []string{"PI_HOLE_BASE_URL"},
			FilePath: "/run/secrets/pi_hole_base_url,/pi_hole_base_url",
			Name:     "pihole.base_url",
			Usage:    "set base url for the PiHole to query for metrics",
			Value:    "http://127.0.0.1:8086",
		},

		// Influx Flags
		&cli.StringFlag{
			EnvVars:  []string{"INFLUX_BASE_URL"},
			FilePath: "/run/secrets/influx_base_url,/influx_base_url",
			Name:     "influx.base_url",
			Usage:    "set base url for Influx instance to write metrics into",
			Value:    "http://127.0.0.1:8086",
		},
		&cli.StringFlag{
			EnvVars:  []string{"INFLUX_BUCKET"},
			FilePath: "/run/secrets/influx_bucket,/influx_bucket",
			Name:     "influx.bucket",
			Usage:    "bucket to sent metrics into",
			Required: true,
		},
		&cli.BoolFlag{
			EnvVars:  []string{"INFLUX_ENABLE_GZIP"},
			FilePath: "/run/secrets/influx_enable_gzip,/influx_enable_gzip",
			Name:     "influx.gzip",
			Usage:    "enable compression of metrics before sending",
			Value:    false,
		},
		&cli.StringFlag{
			EnvVars:  []string{"INFLUX_ORG"},
			FilePath: "/run/secrets/influx_org,/influx_org",
			Name:     "influx.org",
			Usage:    "org to sent metrics into",
			Required: true,
		},
		&cli.StringFlag{
			EnvVars:  []string{"INFLUX_TOKEN"},
			FilePath: "/run/secrets/influx_token,/influx_token",
			Name:     "influx.token",
			Usage:    "token for influx authentication",
			Required: true,
		},
		&cli.StringFlag{
			EnvVars:  []string{"INFLUX_PI_HOLE_HOST"},
			FilePath: "/run/secrets/influx_pi_hole_host,/influx_pi_hole_host",
			Name:     "influx.pi_hole_host",
			Usage:    "custom tag to use for the pi_hole_host tag, defaults to PI_HOLE_BASE_URL",
			Value:    "",
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	appConfig := config.New(c)
	if !appConfig.Valid() {
		return errors.New("provided config was not valid")
	}
	done := make(chan struct{})
	go handleSignals(done)
	metricChan := queryPiHole(appConfig.PiHoleConfig, done)
	go processMetricChan(appConfig.InfluxConfig, metricChan, done)
	log.Print("Program started.")
	<-done
	close(metricChan)
	time.Sleep(1 * time.Second)
	return nil
}

func processMetricChan(influxConfig config.InfluxConfig, metricChan chan map[string]interface{}, done chan struct{}) {
	influxClient, writeAPI := influx.NewClient(influxConfig)
	go func() {
		for {
			select {
			case <-done:
				return
			case err := <-writeAPI.Errors():
				log.Println("influx write error:", err)
			}
		}
	}()
	for {
		select {
		case <-done:
			writeAPI.Flush()
			influxClient.Close()
			return
		case m := <-metricChan:
			go handleMetricMap(writeAPI, m)
		}
	}
}

func handleMetricMap(writeAPI api.WriteAPI, metricMap map[string]interface{}) {
	timestamp := time.Now()

	// create the root pi_hole metric
	piHoleMetric := influxdb2.NewPointWithMeasurement("pi_hole").
		SetTime(timestamp).
		AddField("ads_blocked_today", int(metricMap["ads_blocked_today"].(float64))).
		AddField("queries_cached", int(metricMap["queries_cached"].(float64))).
		AddField("queries_forwarded", int(metricMap["queries_forwarded"].(float64))).
		AddField("dns_queries_all_replies", int(metricMap["dns_queries_all_replies"].(float64))).
		AddField("dns_queries_all_types", int(metricMap["dns_queries_all_types"].(float64))).
		AddField("dns_queries_today", int(metricMap["dns_queries_today"].(float64))).
		AddField("domains_being_blocked", int(metricMap["domains_being_blocked"].(float64))).
		AddField("unique_clients", int(metricMap["unique_clients"].(float64))).
		AddField("unique_domains", int(metricMap["unique_domains"].(float64))).
		AddField("ads_percentage_today", metricMap["ads_percentage_today"].(float64))
	writeAPI.WritePoint(piHoleMetric)

	// create the "pi_hole_top_sources" metrics
	topSources := metricMap["top_sources"].(map[string]interface{})
	for topSource, count := range topSources {
		hostnameAndIpAddress := strings.Split(topSource, "|")
		topSourceMetric := influxdb2.NewPointWithMeasurement("pi_hole_top_sources").
			SetTime(timestamp).
			AddTag("host", hostnameAndIpAddress[0]).
			AddTag("ip_address", hostnameAndIpAddress[1]).
			AddField("count", int(count.(float64)))
		writeAPI.WritePoint(topSourceMetric)
	}

	// create the "pi_hole_query_types" metrics
	queryTypes := metricMap["querytypes"].(map[string]interface{})
	for queryType, percentage := range queryTypes {
		queryTypeMetric := influxdb2.NewPointWithMeasurement("pi_hole_query_types").
			SetTime(timestamp).
			AddTag("type", queryType).
			AddField("percentage", percentage.(float64))
		writeAPI.WritePoint(queryTypeMetric)
	}

	// create the "pi_hole_forward_destinations" metrics
	forwardDestinations := metricMap["forward_destinations"].(map[string]interface{})
	for forwardDestination, percentage := range forwardDestinations {
		hostnameAndIpAddress := strings.Split(forwardDestination, "|")
		forwardDestinationMetric := influxdb2.NewPointWithMeasurement("pi_hole_forward_destinations").
			SetTime(timestamp).
			AddTag("host", hostnameAndIpAddress[0]).
			AddTag("ip_address", hostnameAndIpAddress[1]).
			AddField("percentage", percentage.(float64))
		writeAPI.WritePoint(forwardDestinationMetric)
	}

	// create the "pi_hole_top_ads" metrics
	topAds := metricMap["top_ads"].(map[string]interface{})
	for topAd, count := range topAds {
		topAdMetric := influxdb2.NewPointWithMeasurement("pi_hole_top_ads").
			SetTime(timestamp).
			AddTag("host", topAd).
			AddField("count", int(count.(float64)))
		writeAPI.WritePoint(topAdMetric)
	}

	// create the "pi_hole_top_queries" metrics
	topQueries := metricMap["top_queries"].(map[string]interface{})
	for host, count := range topQueries {
		topQueryMetric := influxdb2.NewPointWithMeasurement("pi_hole_top_queries").
			SetTime(timestamp).
			AddTag("host", host).
			AddField("count", int(count.(float64)))
		writeAPI.WritePoint(topQueryMetric)
	}

	// create the "pi_hole_gravity" metric
	gravityMetric := influxdb2.NewPointWithMeasurement("pi_hole_gravity").
		SetTime(timestamp).
		AddField("updated", int(metricMap["gravity_last_updated"].(map[string]interface{})["absolute"].(float64)))
	writeAPI.WritePoint(gravityMetric)
}

func handleSignals(done chan struct{}) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	recv := <-signals
	log.Printf("%s received, shutting down.", recv.String())
	close(done)
}

func queryPiHole(piHoleConfig config.PiHoleConfig, done chan struct{}) chan map[string]interface{} {
	ticker := time.NewTicker(1 * time.Minute)
	piholeClient := pihole.New(piHoleConfig)
	c := make(chan map[string]interface{})
	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				if metrics, err := piholeClient.GetMetrics(); err == nil {
					c <- metrics
					log.Print("metrics collected and sent")
				}
			}
		}
	}()

	go func() {
		// get metrics once on startup
		if metrics, err := piholeClient.GetMetrics(); err == nil {
			c <- metrics
			log.Print("metrics collected and sent")
		}
	}()

	return c
}
