package provisioner

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

type PrometheusRule struct {
	Name        string            `json:"name"`
	Query       string            `json:"query"`
	Duration    int               `json:"duration"`
	Labels      map[string]string `json:"severity"`
	Annotations map[string]string `json:"annotations"`
	//Health string `json:"health"`
	Type string `json:"type"`
}

type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		Groups []struct {
			Name     string           `json:"name"`
			Rules    []PrometheusRule `json:"rules"`
			Interval int              `json:"interval"`
		} `json:"groups"`
	} `json:"data"`
}

func GetRulesFromURL(url string) []PrometheusRule {
	promClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "prom-rules-scraper")
	req.Close = true

	res, getErr := promClient.Do(req)
	if res != nil {
		defer res.Body.Close()
	}

	if getErr != nil {
		log.Fatal(getErr)
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	promResponse := PrometheusResponse{}

	jsonErr := json.Unmarshal(body, &promResponse)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	rules := []PrometheusRule{}

	for _, group := range promResponse.Data.Groups {
		for _, rule := range group.Rules {
			rules = append(rules, rule)
		}
	}

	return rules
}
