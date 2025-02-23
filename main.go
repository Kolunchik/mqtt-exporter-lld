package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type MetricData struct {
	Topic     string      `json:"topic,omitempty"`
	Type      string      `json:"type,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	Binary    string      `json:"binary,omitempty"`
	Timestamp int64       `json:"ts"`
	RFC3339   string      `json:"rfc3339,omitempty"`
}

type LLDData struct {
	Device string `json:"{#DEVICE}"`
	Name   string `json:"{#NAME},omitempty"`
	Macro  string `json:"{#MACRO},omitempty"`
	Id     string `json:"{#ID},omitempty"`
}

var commit = "unknown"

func getMetrics(url string, metrics map[string]MetricData) bool {
	res, err := http.Get(url)
	if err != nil {
		log.Print(err)
		return false
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode > 299 {
		log.Printf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
		return false
	}
	if err != nil {
		log.Print(err)
		return false
	}
	err = json.Unmarshal(body, &metrics)
	if err != nil {
		log.Print(err)
		return false
	}
	return true
}

func addDevice(lld map[string][]LLDData, key string, device LLDData) bool {
	for i := range lld[key] {
		if lld[key][i].Device == device.Device && lld[key][i].Id == device.Id {
			log.Printf("Device %s %s already exists in lld", key, device.Device)
			return false
		}
	}
	lld[key] = append(lld[key], device)
	return true
}

func lldResult(lld map[string][]LLDData, zh string, legacy bool) bool {
	for k, v := range lld {
		if legacy {
			for i := range v {
				if v[i].Id != "" {
					v[i].Device = v[i].Id
				}
			}
		}
		j, err := json.Marshal(v)
		if err != nil {
			log.Printf("JSON encode error: %s", err)
			return false
		}
		k = k + ".lld"
		fmt.Printf("%q %q %q\n", zh, k, j)
	}
	return true
}

func lldParse(metrics map[string]MetricData, lld map[string][]LLDData) bool {
	for k, _ := range metrics {
		var prefix, device, id string
		parsed := strings.SplitN(k, "/", 10)
		l := len(parsed)
		if l < 5 {
			log.Printf("Oh, %v < 5, skip %v", l, k)
			continue
		}
		if parsed[1] != "devices" && parsed[3] != "controls" {
			log.Printf("No /devices/*/controls/ found, skip %v", k)
			continue
		}
		if parsed[2] == "wb-w1" {
			prefix = parsed[2]
			device = parsed[4]
		} else {
			var ok bool
			prefix, id, ok = strings.Cut(parsed[2], "_")
			if ok {
				_, err := strconv.ParseUint(id, 10, 32)
				if err != nil {
					log.Printf("Invalid device id value: %s, skip", prefix)
					continue
				}
				device = parsed[2]
			} else {
				log.Printf("Not found device id: %s, skip", prefix)
				continue
			}
		}
		dev := LLDData{
			Device: device,
			Id:     id,
			Macro:  strings.ReplaceAll(strings.ToUpper("N_"+device), "-", "_"),
		}
		addDevice(lld, prefix, dev)
	}
	return true
}

func main() {
	var (
		metrics = make(map[string]MetricData)
		lld     = make(map[string][]LLDData)
		opts    struct {
			metrics_url string
			legacy      bool
			zh          string
		}
	)

	flag.StringVar(&opts.metrics_url, "metrics-url", "http://localhost:8080/v1/metrics", "url of mqtt-exporter metrics")
	flag.StringVar(&opts.zh, "zabbix-host", "-", "host name of zabbix host")
	flag.BoolVar(&opts.legacy, "legacy", false, "do not use this")
	flag.Parse()
	if !getMetrics(opts.metrics_url, metrics) {
		log.Fatalf("Can`t get metrics from %v", opts.metrics_url)
	}
	if !lldParse(metrics, lld) {
		log.Fatalf("Can`t parse metrics!")
	}
	if !lldResult(lld, opts.zh, opts.legacy) {
		log.Fatalf("Can`t show result :(")
	}
}
