package provisioner

import (
	"encoding/json"
	"fmt"
	"github.com/AlekSi/zabbix-sender"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"strings"
	"time"
)

type Provisioner struct {
	Config ProvisionerConfig
}

type ProvisionerConfig struct {
	RulesUrl             string `yaml:"rulesUrl"`
	RulesPollingInterval int    `yaml:"rulesPollingTime"`

	ZabbixAddr             string `yaml:"zabbixAddr"`
	ZabbixDiscoveryRuleKey string `yaml:"zabbixDiscoveryRuleKey"`
}

type ZabbixDiscovery struct {
	Data []ZabbixDiscoveryEntry `json:"data"`
}

type ZabbixDiscoveryEntry struct {
	Name    string `json:"{#PROM_NAME}"`
	Summary string `json:"{#PROM_SUMMARY}"`
}

func New(cfg *ProvisionerConfig) *Provisioner {

	return &Provisioner{
		Config: *cfg,
	}

}

func ConfigFromFile(filename string) (cfg *ProvisionerConfig, err error) {
	log.Infof("loading configuration at '%s'", filename)
	configFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("can't open the config file: %s", err)
	}

	// Default values
	config := ProvisionerConfig{
		RulesUrl:               "https://127.0.0.1:9090/api/v1/rules",
		RulesPollingInterval:   3600,
		ZabbixAddr:             "127.0.0.1:10051",
		ZabbixDiscoveryRuleKey: "test",
	}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		return nil, fmt.Errorf("can't read the config file: %s", err)
	}

	log.Info("configuration loaded")

	return &config, nil
}

func (p *Provisioner) Start() {

	for {

		p.GetPrometheusRules()

		time.Sleep(time.Duration(p.Config.RulesPollingInterval) * time.Second)
	}
}

// Get Prometheus rules and create Zabbix discovery payload
func (p *Provisioner) GetPrometheusRules() {

	rules := GetRulesFromURL(p.Config.RulesUrl)
	host_rules := make(map[string][]ZabbixDiscoveryEntry)

	// Parse Prometheus rules and create corresponding discovery items
	for _, rule := range rules {

		if strings.ToLower(rule.Type) != "alerting" {
			continue
		}

		zabbixDiscoveryEntry := ZabbixDiscoveryEntry{
			Name: strings.ToLower(rule.Name),
		}

		for k, v := range rule.Annotations {
			switch k {
			case "zabbix_summary":
				zabbixDiscoveryEntry.Summary = v
			}
		}

		for k, v := range rule.Annotations {
			switch k {
			case "zabbix_host":
				host_rules[v] = append(host_rules[v], zabbixDiscoveryEntry)
			}
		}

	}
	var dataItems zabbix_sender.DataItems

	for k, v := range host_rules {
		json_data, _ := json.Marshal(ZabbixDiscovery{Data: v})

		zabbix_payload := zabbix_sender.DataItem{
			Hostname:  k,
			Value:     string(json_data),
			Timestamp: 0,
			Key:       p.Config.ZabbixDiscoveryRuleKey,
		}

		dataItems = append(dataItems, zabbix_payload)
	}

	dataItems.Marshal()

	addr, resolveErr := net.ResolveTCPAddr("tcp", p.Config.ZabbixAddr)

	if resolveErr != nil {
		log.Error(resolveErr)
	}
	res, sendErr := zabbix_sender.Send(addr, dataItems)

	if sendErr != nil {
		log.Error(sendErr)
	}

	log.Info("result:", res)
}
