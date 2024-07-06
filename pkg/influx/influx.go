package influx

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Client struct {
	c               client
	once            sync.Once
	gzipCompression bool
	bucket          string
	host            string
	org             string
	token           string
}

type client interface {
	Send(points []Metric) error
}

func NewClient() *Client {
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
	return &Client{
		host:            influxUrl,
		bucket:          bucket,
		org:             org,
		token:           token,
		gzipCompression: gzipCompression,
	}
}

func (c *Client) buildOnce() {
	urlBuilder, err := url.Parse(fmt.Sprintf("%s/api/v2/write", c.host))
	if err != nil {
		panic(err)
	}
	q := make(url.Values)
	q.Set("org", c.org)
	q.Set("bucket", c.bucket)
	q.Set("precision", "ns")
	urlBuilder.RawQuery = q.Encode()

	authorizationValue := fmt.Sprintf("Token %s", c.token)
	httpClient := http.Client{Timeout: 5 * time.Second}
	influxUrl := urlBuilder.String()
	if c.gzipCompression {
		println("GZIP Compression Enabled for Influx output")
		c.c = gzipCompressedClient{
			authorizationValue: authorizationValue,
			httpClient:         httpClient,
			url:                influxUrl,
		}
	} else {
		c.c = uncompressedClient{
			authorizationValue: authorizationValue,
			httpClient:         httpClient,
			url:                influxUrl,
		}
	}
}

func (c *Client) Send(points []Metric) error {
	c.once.Do(c.buildOnce)
	return c.c.Send(points)
}

// uncompressedClient sends the metrics to Influx without any compression,
// network bandwidth is not spared in this implementation.
type uncompressedClient struct {
	httpClient         http.Client
	authorizationValue string
	url                string
}

func (c uncompressedClient) Send(points []Metric) error {
	strs := make([]string, len(points))
	for i, v := range points {
		strs[i] = v.String()
	}
	request, err := http.NewRequest(http.MethodPost, c.url, strings.NewReader(strings.Join(strs, "\n")))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain; charset=utf-8")
	request.Header.Set("Authorization", c.authorizationValue)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return errors.Join(errors.New("failed to post metrics"), err)
	}
	defer handleBodyClose(response.Body)
	if response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("received %s from Influx instead of a 204\n%s", response.Status, body)
	}
	return nil
}

// gzipCompressedClient sends the metrics to Influx after gzip compressing the metric lines.
type gzipCompressedClient struct {
	httpClient         http.Client
	authorizationValue string
	url                string
}

func (g gzipCompressedClient) Send(points []Metric) error {
	strs := make([]string, len(points))
	for i, v := range points {
		strs[i] = v.String()
	}

	dataReader := strings.NewReader(strings.Join(strs, "\n"))
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := dataReader.WriteTo(gz); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, g.url, bytes.NewReader(b.Bytes()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain; charset=utf-8")
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Authorization", g.authorizationValue)
	response, err := g.httpClient.Do(request)
	if err != nil {
		return errors.Join(errors.New("failed to post metrics"), err)
	}
	defer handleBodyClose(response.Body)
	if response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("received %s from Influx instead of a 204\n%s", response.Status, body)
	}
	return nil
}

func handleBodyClose(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Printf("failed to close closer %s\n", err.Error())
	}
}
