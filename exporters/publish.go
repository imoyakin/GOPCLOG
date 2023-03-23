package exporter

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/doteich/OPC-UA-Logger/exporters/http_exporter"
	"github.com/doteich/OPC-UA-Logger/exporters/metrics_exporter"
	"github.com/doteich/OPC-UA-Logger/exporters/mongodb"
	"github.com/doteich/OPC-UA-Logger/exporters/websockets"
	"github.com/doteich/OPC-UA-Logger/setup"
)

type Exporters struct {
	Rest       bool
	Prometheus bool
	Websockets bool
	MongoDB    bool
}

var EnabledExporters Exporters
var namespace string

//var PubConfig setup.Config

func InitExporters(config *setup.Config) {

	//PubConfig = *config

	namespace = strings.Replace(config.LoggerConfig.Name, " ", "", -1)

	go metrics_exporter.ExposeMetrics(namespace)

	if config.ExporterConfig.Rest.Enabled {
		http_exporter.InitRoutes(config.ExporterConfig.Rest.URL, config.ExporterConfig.Rest.AuthType, config.ExporterConfig.Rest.Username, config.ExporterConfig.Rest.Password)
		EnabledExporters.Rest = true
	}

	if config.ExporterConfig.Prometheus.Enabled {
		EnabledExporters.Prometheus = true

	}

	if config.ExporterConfig.Websockets.Enabled {
		go websockets.InitWebsockets()
		EnabledExporters.Websockets = true
	}

	if config.ExporterConfig.MongoDB.Enabled {
		mongodb.CreateConnection(namespace, config.ExporterConfig.MongoDB.Username, config.ExporterConfig.MongoDB.Password, config.ExporterConfig.MongoDB.URL, config.ExporterConfig.MongoDB.Port)
		EnabledExporters.MongoDB = true
	}

}

func PublishData(nodeId string, iface interface{}, timestamp time.Time) {

	var dataType string
	var metricsValue float64

	switch v := iface.(type) {
	case int:
		dataType = "int"
		metricsValue = float64(v)
	case int8:
		dataType = "int8"
		metricsValue = float64(v)
	case int16:
		dataType = "int16"
		metricsValue = float64(v)
	case int32:
		dataType = "int32"
		metricsValue = float64(v)
	case uint8:
		dataType = "uint8"
		metricsValue = float64(v)
	case uint16:
		dataType = "uint16"
		metricsValue = float64(v)
	case uint32:
		dataType = "uint32"
		metricsValue = float64(v)
	case float32:
		dataType = "float32"
		metricsValue = float64(v)
	case float64:
		dataType = "float64"
		metricsValue = v
	case string:
		dataType = "string"
	case bool:
		dataType = "bool"
	}

	node, err := findNodeDetails(nodeId)

	if err != nil {
		fmt.Println(err)
		return
	}

	if EnabledExporters.Rest {
		http_exporter.PostLoggedData(node.NodeId, node.NodeName, iface, timestamp, setup.PubConfig.LoggerConfig.Name, setup.PubConfig.ClientConfig.Url, dataType)
	}

	if EnabledExporters.Prometheus && (dataType != "bool" && dataType != "string") {

		metrics_exporter.SetMetricsValue(node.MetricsType, nodeId, node.NodeName, metricsValue)
	}

	if EnabledExporters.Websockets {

		websockets.BroadcastToWebsocket(node.NodeId, node.NodeName, iface, timestamp, setup.PubConfig.LoggerConfig.Name, setup.PubConfig.ClientConfig.Url, dataType)
	}

	if EnabledExporters.MongoDB {
		mongodb.WriteData(node.NodeId, node.NodeName, iface, timestamp, setup.PubConfig.LoggerConfig.Name, setup.PubConfig.ClientConfig.Url, dataType, namespace)
	}

}

func findNodeDetails(nodeId string) (setup.NodeObject, error) {
	for _, node := range setup.PubConfig.Nodes {
		if nodeId == node.NodeId {
			return node, nil
		}
	}
	return setup.NodeObject{}, errors.New("node not found")
}
