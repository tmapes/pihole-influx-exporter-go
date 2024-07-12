package main

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"log"
	"os"
	"os/signal"
	"pihole-influx-exporter-go/pkg/influx"
	"pihole-influx-exporter-go/pkg/pihole"
	"strings"
	"syscall"
	"time"
)

func main() {
	done := make(chan struct{})
	go handleSignals(done)
	metricChan := queryPiHole(done)
	go processMetricChan(metricChan, done)
	log.Print("Program started.")
	<-done
	close(metricChan)
	time.Sleep(1 * time.Second)
	os.Exit(0)
}

func processMetricChan(metricChan chan map[string]interface{}, done chan struct{}) {
	influxClient, writeAPI := influx.NewV2Client()
	tagPiHoleHost := determineHostTag()
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
			go handleMetricMap(writeAPI, tagPiHoleHost, m)
		}
	}
}

func handleMetricMap(writeAPI api.WriteAPI, tagPiHoleHost string, metricMap map[string]interface{}) {
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
			AddTag("pi_hole_host", tagPiHoleHost).
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
			AddTag("pi_hole_host", tagPiHoleHost).
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
			AddTag("pi_hole_host", tagPiHoleHost).
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
			AddTag("pi_hole_host", tagPiHoleHost).
			AddTag("host", topAd).
			AddField("count", int(count.(float64)))
		writeAPI.WritePoint(topAdMetric)
	}

	// create the "pi_hole_top_queries" metrics
	topQueries := metricMap["top_queries"].(map[string]interface{})
	for host, count := range topQueries {
		topQueryMetric := influxdb2.NewPointWithMeasurement("pi_hole_top_queries").
			SetTime(timestamp).
			AddTag("pi_hole_host", tagPiHoleHost).
			AddTag("host", host).
			AddField("count", int(count.(float64)))
		writeAPI.WritePoint(topQueryMetric)
	}

	// create the "pi_hole_gravity" metric
	gravityMetric := influxdb2.NewPointWithMeasurement("pi_hole_gravity").
		SetTime(timestamp).
		AddTag("pi_hole_host", tagPiHoleHost).
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

func queryPiHole(done chan struct{}) chan map[string]interface{} {
	ticker := time.NewTicker(1 * time.Minute)
	piholeClient := pihole.New()
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
