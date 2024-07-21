package pihole

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"pihole-influx-exporter-go/pkg/config"
	"time"
)

type Client struct {
	url        string
	httpClient http.Client
}

func New(c config.PiHoleConfig) *Client {
	clientUrl, err := url.Parse(fmt.Sprintf("%s/admin/api.php", c.BaseUrl))
	if err != nil {
		_ = fmt.Errorf("failed to parse %s, %s", c.BaseUrl, err)
		return nil
	}
	log.Printf("PiHole Config | BaseUrl: '%s'", c.BaseUrl)

	query := clientUrl.Query()
	query.Set("auth", c.Token)
	query.Set("summaryRaw", "")
	query.Set("overTimeData", "")
	query.Set("topItems", "")
	query.Set("recentItems", "")
	query.Set("getQueryTypes", "")
	query.Set("getForwardDestinations", "")
	query.Set("getQuerySources", "")
	query.Set("jsonForceObject", "")
	clientUrl.RawQuery = query.Encode()

	return &Client{
		url: clientUrl.String(),
		httpClient: http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (client *Client) GetMetrics() (map[string]interface{}, error) {
	request, err := http.NewRequest(http.MethodGet, client.url, nil)
	if err != nil {
		return make(map[string]interface{}), nil
	}
	l, err := client.httpClient.Do(request)
	if err != nil {
		return map[string]interface{}{}, errors.New("failed to fetch metrics")
	}
	if l.StatusCode != http.StatusOK {
		return map[string]interface{}{}, fmt.Errorf("%s", l.Status)
	}
	defer l.Body.Close()
	bytes, _ := io.ReadAll(l.Body)
	respMap := make(map[string]interface{})
	err = json.Unmarshal(bytes, &respMap)
	if len(respMap) == 0 {
		err = errors.New("pi hole response had no entries")
		log.Print(err)
		return nil, err
	}
	return respMap, err
}
