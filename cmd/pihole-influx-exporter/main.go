package main

import (
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
	influxClient := influx.NewClient()
	tagPiHoleHost := determineHostTag()
	for {
		select {
		case <-done:
			return
		case m := <-metricChan:
			go handleMetricMap(influxClient, tagPiHoleHost, m)
		}
	}
}

func handleMetricMap(client *influx.Client, tagPiHoleHost string, metricMap map[string]interface{}) {
	metrics := make([]influx.Metric, 0)
	timestamp := time.Now().UnixNano()

	// create the root pi_hole metric
	piHoleMetric := influx.NewMetric("pi_hole", timestamp)
	_ = piHoleMetric.WithTag("pi_hole_host", tagPiHoleHost)
	_ = piHoleMetric.WithIntField("ads_blocked_today", int(metricMap["ads_blocked_today"].(float64)))
	_ = piHoleMetric.WithIntField("queries_cached", int(metricMap["queries_cached"].(float64)))
	_ = piHoleMetric.WithIntField("queries_forwarded", int(metricMap["queries_forwarded"].(float64)))
	_ = piHoleMetric.WithIntField("dns_queries_all_replies", int(metricMap["dns_queries_all_replies"].(float64)))
	_ = piHoleMetric.WithIntField("dns_queries_all_types", int(metricMap["dns_queries_all_types"].(float64)))
	_ = piHoleMetric.WithIntField("dns_queries_today", int(metricMap["dns_queries_today"].(float64)))
	_ = piHoleMetric.WithIntField("domains_being_blocked", int(metricMap["domains_being_blocked"].(float64)))
	_ = piHoleMetric.WithIntField("unique_clients", int(metricMap["unique_clients"].(float64)))
	_ = piHoleMetric.WithIntField("unique_domains", int(metricMap["unique_domains"].(float64)))
	_ = piHoleMetric.WithFloatField("ads_percentage_today", metricMap["ads_percentage_today"].(float64))
	metrics = append(metrics, piHoleMetric)

	// create the "pi_hole_top_sources" metrics
	topSources := metricMap["top_sources"].(map[string]interface{})
	for topSource, count := range topSources {
		hostnameAndIpAddress := strings.Split(topSource, "|")
		topSourceMetric := influx.NewMetric("pi_hole_top_sources", timestamp)
		_ = topSourceMetric.WithTag("pi_hole_host", tagPiHoleHost)
		_ = topSourceMetric.WithTag("host", hostnameAndIpAddress[0])
		_ = topSourceMetric.WithTag("ip_address", hostnameAndIpAddress[1])
		_ = topSourceMetric.WithIntField("count", int(count.(float64)))
		metrics = append(metrics, topSourceMetric)
	}

	// create the "pi_hole_query_types" metrics
	queryTypes := metricMap["querytypes"].(map[string]interface{})
	for queryType, percentage := range queryTypes {
		queryTypeMetric := influx.NewMetric("pi_hole_query_types", timestamp)
		_ = queryTypeMetric.WithTag("pi_hole_host", tagPiHoleHost)
		_ = queryTypeMetric.WithTag("type", queryType)
		_ = queryTypeMetric.WithFloatField("percentage", percentage.(float64))
		metrics = append(metrics, queryTypeMetric)
	}

	// create the "pi_hole_forward_destinations" metrics
	forwardDestinations := metricMap["forward_destinations"].(map[string]interface{})
	for forwardDestination, percentage := range forwardDestinations {
		hostnameAndIpAddress := strings.Split(forwardDestination, "|")
		forwardDestinationMetric := influx.NewMetric("pi_hole_forward_destinations", timestamp)
		_ = forwardDestinationMetric.WithTag("pi_hole_host", tagPiHoleHost)
		_ = forwardDestinationMetric.WithTag("host", hostnameAndIpAddress[0])
		_ = forwardDestinationMetric.WithTag("ip_address", hostnameAndIpAddress[1])
		_ = forwardDestinationMetric.WithFloatField("percentage", percentage.(float64))
		metrics = append(metrics, forwardDestinationMetric)
	}

	// create the "pi_hole_top_ads" metrics
	topAds := metricMap["top_ads"].(map[string]interface{})
	for topAd, count := range topAds {
		topAdMetric := influx.NewMetric("pi_hole_top_ads", timestamp)
		_ = topAdMetric.WithTag("pi_hole_host", tagPiHoleHost)
		_ = topAdMetric.WithTag("host", topAd)
		_ = topAdMetric.WithIntField("count", int(count.(float64)))
		metrics = append(metrics, topAdMetric)
	}

	// create the "pi_hole_top_queries" metrics
	topQueries := metricMap["top_queries"].(map[string]interface{})
	for host, count := range topQueries {
		topAdMetric := influx.NewMetric("pi_hole_top_queries", timestamp)
		_ = topAdMetric.WithTag("pi_hole_host", tagPiHoleHost)
		_ = topAdMetric.WithTag("host", host)
		_ = topAdMetric.WithIntField("count", int(count.(float64)))
		metrics = append(metrics, topAdMetric)
	}

	// create the "pi_hole_gravity" metric
	gravityMetric := influx.NewMetric("pi_hole_gravity", timestamp)
	_ = gravityMetric.WithTag("pi_hole_host", tagPiHoleHost)
	_ = gravityMetric.WithIntField("updated", int(metricMap["gravity_last_updated"].(map[string]interface{})["absolute"].(float64)))
	metrics = append(metrics, gravityMetric)

	if err := client.Send(metrics); err != nil {
		log.Println(err)
	}
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
