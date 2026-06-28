//go:build lazydev

package lazytelemetry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxRequestTraceSnapshots = 50

type requestTracesResponse struct {
	Directory string                 `json:"directory"`
	Traces    []requestTraceSnapshot `json:"traces"`
	Errors    []string               `json:"errors,omitempty"`
}

type requestTraceSnapshot struct {
	requestCaptureDocument
	Logs []map[string]interface{} `json:"logs,omitempty"`
}

func handleRequestTraces(w http.ResponseWriter, _ *http.Request) {
	response := requestTracesResponse{Directory: requestCaptureDirectory}
	traces, errors := readRequestTraceSnapshots()
	response.Traces = traces
	response.Errors = errors

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(response)
}

func readRequestTraceSnapshots() ([]requestTraceSnapshot, []string) {
	entries, err := os.ReadDir(requestCaptureDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{fmt.Sprintf("read trace directory: %v", err)}
	}

	snapshots := make([]requestTraceSnapshot, 0)
	errors := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".spans") {
			continue
		}
		path := filepath.Join(requestCaptureDirectory, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("read %s: %v", path, err))
			continue
		}
		var document requestCaptureDocument
		if err := json.Unmarshal(data, &document); err != nil {
			errors = append(errors, fmt.Sprintf("parse %s: %v", path, err))
			continue
		}
		if strings.TrimSpace(document.RequestID) == "" {
			document.RequestID = strings.TrimSuffix(entry.Name(), ".spans")
		}
		logs, err := readRequestTraceLogs(filepath.Join(requestCaptureDirectory, strings.TrimSuffix(entry.Name(), ".spans")+".log.json"))
		if err != nil {
			errors = append(errors, err.Error())
		}
		snapshots = append(snapshots, requestTraceSnapshot{
			requestCaptureDocument: document,
			Logs:                   logs,
		})
	}

	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].StartedAt.After(snapshots[j].StartedAt)
	})
	if len(snapshots) > maxRequestTraceSnapshots {
		snapshots = snapshots[:maxRequestTraceSnapshots]
	}
	return snapshots, errors
}

func readRequestTraceLogs(path string) ([]map[string]interface{}, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %v", path, err)
	}
	defer file.Close()

	var logs []map[string]interface{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return logs, fmt.Errorf("parse %s: %v", path, err)
		}
		logs = append(logs, record)
	}
	if err := scanner.Err(); err != nil {
		return logs, fmt.Errorf("read %s: %v", path, err)
	}
	return logs, nil
}
