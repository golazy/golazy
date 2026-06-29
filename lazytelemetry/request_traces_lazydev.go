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

type requestTraceFilter struct {
	Query    string
	Category string
}

func handleRequestTraces(w http.ResponseWriter, r *http.Request) {
	response := requestTracesResponse{Directory: requestCaptureDirectory}
	traces, errors := readRequestTraceSnapshots(requestTraceFilterFromRequest(r))
	response.Traces = traces
	response.Errors = errors

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(response)
}

func handleRequestTracesClear(w http.ResponseWriter, r *http.Request) {
	if err := clearRequestTraceSnapshots(); err != nil {
		http.Error(w, fmt.Sprintf("clear request traces: %v", err), http.StatusInternalServerError)
		return
	}
	handleRequestTraces(w, r)
}

func requestTraceFilterFromRequest(r *http.Request) requestTraceFilter {
	if r == nil {
		return requestTraceFilter{Category: "all"}
	}
	return requestTraceFilter{
		Query:    strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q"))),
		Category: normalizeRequestTraceCategory(r.URL.Query().Get("type")),
	}
}

func normalizeRequestTraceCategory(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "framework", "assets", "other":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "all"
	}
}

func readRequestTraceSnapshots(filter requestTraceFilter) ([]requestTraceSnapshot, []string) {
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
		if document.Category == "" {
			document.HandledBy, document.Category = requestHandledBy(document.Spans)
		}
		if !requestTraceMatches(document, filter) {
			continue
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

func requestTraceMatches(document requestCaptureDocument, filter requestTraceFilter) bool {
	if filter.Query != "" && !strings.Contains(strings.ToLower(document.Path), filter.Query) {
		return false
	}
	if filter.Category != "" && filter.Category != "all" && document.Category != filter.Category {
		return false
	}
	return true
}

func clearRequestTraceSnapshots() error {
	entries, err := os.ReadDir(requestCaptureDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".trace") && !strings.HasSuffix(name, ".spans") && !strings.HasSuffix(name, ".log.json") {
			continue
		}
		if err := os.Remove(filepath.Join(requestCaptureDirectory, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
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
