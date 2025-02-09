package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetData(t *testing.T) {
	// Создаем мок-сервер
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]MetricData{
			"devices/wb-w1/controls/28-000000000001": {Topic: "test", Value: 25.5, Timestamp: 1234567890},
		})
	}))
	defer ts.Close()

	// Вызываем функцию getData с URL мок-сервера
	if !getData(ts.URL) {
		t.Errorf("getData() returned false, expected true")
	}

	// Проверяем, что данные были правильно загружены
	if len(metrics) == 0 {
		t.Errorf("metrics map is empty, expected at least one entry")
	}
}

func TestAddLLD(t *testing.T) {
	key := "testKey"
	device := LLDData{Device: "testDevice"}

	// Добавляем устройство
	if !addLLD(key, device) {
		t.Errorf("addLLD() returned false, expected true")
	}

	// Проверяем, что устройство было добавлено
	if len(lld[key]) != 1 || lld[key][0].Device != device.Device {
		t.Errorf("device was not added correctly")
	}

	// Пытаемся добавить то же устройство снова
	if addLLD(key, device) {
		t.Errorf("addLLD() returned true for duplicate device, expected false")
	}
}

func TestMainLogic(t *testing.T) {
	// Создаем мок-сервер
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]MetricData{
			"/devices/wb-w1/controls/28-000000000001": {Topic: "test3", Value: 25.5, Timestamp: 1234567890},
			"/devices/wb-w2/controls/28-000000000001": {Topic: "test5", Value: 30.4, Timestamp: 1234567891},
			"/devices/msu24hit_5/controls/0000000001": {Topic: "test8", Value: 30.2, Timestamp: 1234567891},
			"/devices/msu24hit_6/controls/0000000001": {Topic: "test69", Value: 30.1, Timestamp: 1234567891},
			"/devices/msu24hit_6/controls/0000000002": {Topic: "test9", Value: 30.1, Timestamp: 1234567891},
			"/invalid/key/format": {Topic: "test6", Value: 25.5, Timestamp: 1234567890},
			"///": {Topic: "test68", Value: 2.5, Timestamp: 1234567890},
			"/devices/msu24hit_dda/controls/0000000001": {Topic: "test7", Value: 34.0, Timestamp: 1234567891},
		})
	}))
	defer ts.Close()

	// Запускаем основную логику
	flag.Set("http-url", ts.URL)
	main()

	// Проверяем, что LLD данные были правильно сформированы
	if len(lld["wb-w1"]) != 1 || lld["wb-w1"][0].Device != "28-000000000001" {
		t.Errorf("LLD data for wb-w1 was not generated correctly")
	}

	if len(lld["msu24hit"]) != 2 || lld["msu24hit"][0].Device != "msu24hit_5" || lld["msu24hit"][1].Device != "msu24hit_6" {
		t.Errorf("LLD data for msu24hit was not generated correctly")
	}

	if len(lld) != 3 {
		t.Errorf("LLD data should have 3 keys")
	}
}

func TestInvalidURL(t *testing.T) {
	invalidURL := "http://invalid.url"
	if getData(invalidURL) {
		t.Errorf("getData() returned true for invalid URL, expected false")
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

	// Вызываем функцию getData с URL мок-сервера
	if getData(ts.URL) {
		t.Errorf("getData() returned true for invalid JSON, expected false")
	}
}

func TestHTTPError(t *testing.T) {
	// Создаем мок-сервер, который возвращает ошибку
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	// Вызываем функцию getData с URL мок-сервера
	if getData(ts.URL) {
		t.Errorf("getData() returned true for HTTP error, expected false")
	}
}
