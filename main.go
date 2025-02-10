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
}

var (
	metrics = make(map[string]MetricData)
	lld     = make(map[string][]LLDData)
	httpURL string
)

func getData(httpURL string) bool {
	res, err := http.Get(httpURL)
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

func addLLD(key string, device LLDData) bool {
	for i := range lld[key] {
		if lld[key][i].Device == device.Device {
			log.Printf("Device %s %s already exists in lld", key, device.Device)
			return false
		}
	}
	lld[key] = append(lld[key], device)
	return true
}

func init() {
	flag.StringVar(&httpURL, "http-url", "http://localhost:8080/v1/metrics", "url of mqtt-exporter metrics")
}

func main() {
	zs := flag.String("zabbix-sender", "zabbix_sender --config /etc/zabbix/zabbix_agentd.conf --verbose", "zabbix_sender command")
	zh := flag.String("zabbix-host", "", "host name the item belongs to")
	flag.Parse()

	if getData(httpURL) {
		log.Printf("Gotcha")
	}

	for key, _ := range metrics {
		parsed := strings.SplitN(key, "/", 10)
		l := len(parsed)
		if l > 2 {
			if parsed[1] == "devices" && parsed[3] == "controls" {
				if parsed[2] == "wb-w1" && l > 4 {
					device := LLDData{
						Device: parsed[4],
					}
					addLLD(parsed[2], device)
				} else {
					if l == 5 {
						longname := parsed[2]
						name, index, ok := strings.Cut(parsed[2], "_")
						if ok {
							_, err := strconv.ParseUint(index, 10, 32)
							if err != nil {
								log.Printf("Invalid device index value: %s", index)
								continue
							}
							device := LLDData{
								Device: longname,
							}
							addLLD(name, device)
						}
					}
				}
			}
		}
	}

	if len(*zh) > 0 {
		*zh = fmt.Sprintf("--host %q", *zh)
	}

	for k, v := range lld {
		j, err := json.Marshal(v)
		if err != nil {
			log.Printf("JSON encode error: %s", err)
			continue
		}
		k = k + ".lld"
		fmt.Printf("%s %s --key %q --value %q\n", *zs, *zh, k, j)
	}
}
