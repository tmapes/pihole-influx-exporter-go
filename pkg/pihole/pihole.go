package pihole

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	url        string
	httpClient http.Client
}

func New() *Client {
	token, ok := os.LookupEnv("PI_HOLE_API_TOKEN")
	if !ok {
		panic("PI_HOLE_API_TOKEN not set!")
	}
	val, ok := os.LookupEnv("PI_HOLE_HOST")
	if !ok {
		val = "http://localhost:8086"
	}

	clientUrl, err := url.Parse(fmt.Sprintf("%s/admin/api.php", val))
	if err != nil {
		_ = fmt.Errorf("failed to parse %s, %s", val, err)
		return nil
	}
	query := clientUrl.Query()
	query.Set("auth", token)
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
	return respMap, err
}
