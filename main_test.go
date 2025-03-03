package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

var (
	metrics = make(map[string]MetricData)
	lld     = make(map[string][]LLDData)
)

func TestInvalidURL(t *testing.T) {
	if getMetrics("http://invalid.url", metrics) {
		t.Errorf("getMetrics() returned true for invalid URL, expected false")
	}
}

func TestLooongAnswer(t *testing.T) {
	// Создаем мок-сервер, который доооолго возвращает валидный JSON
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-r.Context().Done():
			return
		case <-timer.C:
			json.NewEncoder(w).Encode(map[string]MetricData{
				"devices/wb-w1/controls/28-0200000000001":   {Topic: "test", Value: 25.5, Timestamp: 1234567890},
				"/devices/wb-w1/controls/28-000000000001":   {Topic: "test3", Value: 25.5, Timestamp: 1234567890},
				"/devices/wb-w2/controls/28-000000000001":   {Topic: "test5", Value: 30.4, Timestamp: 1234567891},
				"/devices/msu24hit_5/controls/0000000001":   {Topic: "test8", Value: 30.2, Timestamp: 1234567891},
				"/devices/wb-mcm16_1/controls/0005000001":   {Topic: "test-8", Value: 30.3, Timestamp: 1234567891},
				"/devices/msu24hit_6/controls/0000000001":   {Topic: "test69", Value: 30.1, Timestamp: 1234567891},
				"/devices/msu24hit_6/controls/0000000002":   {Topic: "test9", Value: 30.1, Timestamp: 1234567891},
				"/devices/msu24hit_6_7/controls/00000002":   {Topic: "test29", Value: 35.1, Timestamp: 1234567891},
				"/invalid/key/format":                       {Topic: "test6", Value: 25.5, Timestamp: 1234567890},
				"///":                                       {Topic: "test68", Value: 2.5, Timestamp: 1234567890},
				"/devices/msu24hit_dda/controls/0000000001": {Topic: "test7", Value: 34.0, Timestamp: 1234567891},
			})
		}
	}))
	defer ts.Close()

	// Вызываем функцию getMetrics с URL мок-сервера
	if getMetrics(ts.URL, metrics) {
		t.Errorf("getMetrics() returned true for invalid JSON, expected false")
	}
}

func TestInvalidJSON(t *testing.T) {
	// Создаем мок-сервер, который возвращает невалидный JSON
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer ts.Close()

	// Вызываем функцию getMetrics с URL мок-сервера
	if getMetrics(ts.URL, metrics) {
		t.Errorf("getMetrics() returned true for invalid JSON, expected false")
	}
}

func TestHTTPError(t *testing.T) {
	// Создаем мок-сервер, который возвращает ошибку
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	// Вызываем функцию getMetrics с URL мок-сервера
	if getMetrics(ts.URL, metrics) {
		t.Errorf("getMetrics() returned true for HTTP error, expected false")
	}
}

func TestGetMetrics(t *testing.T) {
	// Создаем мок-сервер
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]MetricData{
			"devices/wb-w1/controls/28-0200000000001":   {Topic: "test", Value: 25.5, Timestamp: 1234567890},
			"/devices/wb-w1/controls/28-000000000001":   {Topic: "test3", Value: 25.5, Timestamp: 1234567890},
			"/devices/wb-w2/controls/28-000000000001":   {Topic: "test5", Value: 30.4, Timestamp: 1234567891},
			"/devices/msu24hit_5/controls/0000000001":   {Topic: "test8", Value: 30.2, Timestamp: 1234567891},
			"/devices/wb-mcm16_1/controls/0005000001":   {Topic: "test-8", Value: 30.3, Timestamp: 1234567891},
			"/devices/msu24hit_6/controls/0000000001":   {Topic: "test69", Value: 30.1, Timestamp: 1234567891},
			"/devices/msu24hit_6/controls/0000000002":   {Topic: "test9", Value: 30.1, Timestamp: 1234567891},
			"/devices/msu24hit_6_7/controls/00000002":   {Topic: "test29", Value: 35.1, Timestamp: 1234567891},
			"/invalid/key/format":                       {Topic: "test6", Value: 25.5, Timestamp: 1234567890},
			"///":                                       {Topic: "test68", Value: 2.5, Timestamp: 1234567890},
			"/devices/msu24hit_dda/controls/0000000001": {Topic: "test7", Value: 34.0, Timestamp: 1234567891},
			"/devices/wb-gpio/controls/EXT1_DR14":       {Topic: "test88", Value: 0, Timestamp: 1234567891},
			"/devices/wb-gpio/controls/EXT1_DR14/meta":  {Topic: "test888", Value: "{\"order\":14,\"readonly\":true,\"type\":\"switch\"}", Timestamp: 1234567891},
		})
	}))
	defer ts.Close()

	// Вызываем функцию getMetrics с URL мок-сервера
	if !getMetrics(ts.URL, metrics) {
		t.Errorf("getMetrics() returned false, expected true")
	}

	// Проверяем, что данные были правильно загружены
	if len(metrics) == 0 {
		t.Errorf("metrics map is empty, expected at least one entry")
	}
}

func TestAddDevice(t *testing.T) {
	key := "testKey"
	device := LLDData{Device: "testDevice"}

	// Добавляем устройство
	if !addDevice(lld, key, device) {
		t.Errorf("addDevice() returned false, expected true")
	}

	// Проверяем, что устройство было добавлено
	if len(lld[key]) != 1 || lld[key][0].Device != device.Device {
		t.Errorf("device was not added correctly")
	}

	// Пытаемся добавить то же устройство снова
	if addDevice(lld, key, device) {
		t.Errorf("addDevice() returned true for duplicate device, expected false")
	}
}

func TestMainLogicLLD(t *testing.T) {
	if !lldParse(metrics, lld) {
		t.Errorf("lldParse() returned false, expected true")
	}

	// Проверяем, что LLD данные были правильно сформированы
	if len(lld["wb-w1"]) != 1 || lld["wb-w1"][0].Device != "28-000000000001" {
		t.Errorf("LLD data for wb-w1 was not generated correctly")
	}

	if len(lld["wb-w1"][0].Id) > 0 {
		t.Errorf("LLD data for wb-w1 was not generated correctly")
	}

	if len(lld["msu24hit"]) == 2 {
		f := 0
		if slices.ContainsFunc(lld["msu24hit"], func(n LLDData) bool {
			return n.Device == "msu24hit_5" && n.Id == "5" && n.Macro == "N_MSU24HIT_5"
		}) {
			f++
		}
		if slices.ContainsFunc(lld["msu24hit"], func(n LLDData) bool {
			return n.Device == "msu24hit_6" && n.Id == "6" && n.Macro == "N_MSU24HIT_6"
		}) {
			f++
		}
		if f != 2 {
			t.Errorf("LLD data for msu24hit was not generated correctly")
		}
	} else {
		t.Errorf("LLD data for msu24hit was not generated correctly")
	}

	if !slices.ContainsFunc(lld["wb-mcm16"], func(n LLDData) bool {
		return n.Device == "wb-mcm16_1" && n.Id == "1" && n.Macro == "N_WB_MCM16_1"
	}) {
		t.Errorf("LLD data for wb-mcm16_1 was not generated correctly")
	}

	if len(lld) != 4 {
		t.Errorf("LLD data should have 4 keys")
	}

	if !lldResult(lld, "figa", false) {
		t.Errorf("lldResult() returned false, expected true")
	}
}

func TestMainLogicLLDLegacy(t *testing.T) {
	if !lldResult(lld, "legacy", true) {
		t.Errorf("lldParse() returned false, expected true")
	}

	// Проверяем, что LLD данные были правильно сформированы
	if len(lld["wb-w1"]) != 1 || lld["wb-w1"][0].Device != "28-000000000001" {
		t.Errorf("LLD data for wb-w1 was not generated correctly")
	}

	if len(lld["wb-w1"][0].Id) > 0 {
		t.Errorf("LLD data for wb-w1 was not generated correctly")
	}

	if len(lld["msu24hit"]) == 2 {
		f := 0
		if slices.ContainsFunc(lld["msu24hit"], func(n LLDData) bool {
			return n.Device == "5" && n.Id == "5" && n.Macro == "N_MSU24HIT_5"
		}) {
			f++
		}
		if slices.ContainsFunc(lld["msu24hit"], func(n LLDData) bool {
			return n.Device == "6" && n.Id == "6" && n.Macro == "N_MSU24HIT_6"
		}) {
			f++
		}
		if f != 2 {
			t.Errorf("LLD data for msu24hit was not generated correctly")
		}
	} else {
		t.Errorf("LLD data for msu24hit was not generated correctly")
	}

	if !slices.ContainsFunc(lld["wb-mcm16"], func(n LLDData) bool {
		return n.Device == "1" && n.Id == "1" && n.Macro == "N_WB_MCM16_1"
	}) {
		t.Errorf("LLD data for wb-mcm16_1 was not generated correctly")
	}

	if len(lld) != 4 {
		t.Errorf("LLD data should have 4 keys")
	}
}
