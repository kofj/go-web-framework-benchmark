// Command genhtml reads the benchmark CSV files in this directory and writes an
// interactive index.html (Chart.js). It replaces the need for gnuplot when you
// only want a web view of the results. Run it from the testresults directory:
//
//	go run ./genhtml
//
// It is also invoked automatically at the end of plot.sh.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strconv"
	"strings"
)

// dataset describes one chart: where its data comes from and how to label it.
type dataset struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Sub   string `json:"sub"`
	Unit  string `json:"unit"`
	File  string `json:"-"`

	Labels     []string    `json:"labels"`     // x-axis category per CSV row
	Rows       [][]float64 `json:"rows"`        // row-major values, columns = frameworks
	Frameworks []string    `json:"frameworks"` // CSV header (minus the leading blank cell)
}

// charts lists every CSV to render, in display order.
var charts = []dataset{
	{ID: "processtime", Title: "Processing time", Sub: "requests / second (higher is better)", Unit: "req/s", File: "processtime.csv"},
	{ID: "processtime_latency", Title: "Processing time — Latency", Sub: "millisecond (lower is better)", Unit: "ms", File: "processtime_latency.csv"},
	{ID: "processtime_alloc", Title: "Processing time — Allocations", Sub: "MB (lower is better)", Unit: "MB", File: "processtime_alloc.csv"},
	{ID: "processtime_pipeline", Title: "Processing time — Pipelining", Sub: "requests / second (higher is better)", Unit: "req/s", File: "processtime-pipeline.csv"},
	{ID: "concurrency", Title: "Concurrency (30ms)", Sub: "requests / second (higher is better)", Unit: "req/s", File: "concurrency.csv"},
	{ID: "concurrency_latency", Title: "Concurrency — Latency", Sub: "millisecond (lower is better)", Unit: "ms", File: "concurrency_latency.csv"},
	{ID: "concurrency_alloc", Title: "Concurrency — Allocations", Sub: "MB (lower is better)", Unit: "MB", File: "concurrency_alloc.csv"},
	{ID: "concurrency_pipeline", Title: "Concurrency — Pipelining", Sub: "requests / second (higher is better)", Unit: "req/s", File: "concurrency-pipeline.csv"},
	{ID: "cpubound_concurrency", Title: "CPU-bound — Concurrency", Sub: "requests / second (higher is better)", Unit: "req/s", File: "cpubound-concurrency.csv"},
	{ID: "cpubound", Title: "CPU-bound", Sub: "requests / second (higher is better)", Unit: "req/s", File: "cpubound.csv"},
}

func main() {
	var built []dataset
	var frameworks []string
	for _, ds := range charts {
		if err := loadCSV(&ds); err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", ds.File, err)
			continue
		}
		if len(frameworks) == 0 {
			frameworks = ds.Frameworks
		}
		built = append(built, ds)
	}
	if len(built) == 0 {
		fmt.Fprintln(os.Stderr, "no CSV data found; nothing to generate")
		os.Exit(1)
	}

	dataJSON, err := json.MarshalIndent(built, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fwJSON, _ := json.Marshal(frameworks)

	var buf bytes.Buffer
	if err := page.Execute(&buf, map[string]any{
		"Datasets":   template.JS(dataJSON),
		"Frameworks": template.JS(fwJSON),
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile("index.html", buf.Bytes(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote index.html (%d charts, %d frameworks)\n", len(built), len(frameworks))
}

// loadCSV parses a benchmark CSV whose first row is a header (blank cell +
// framework names) and whose first column is the x-axis label per row.
func loadCSV(ds *dataset) error {
	f, err := os.Open(ds.File)
	if err != nil {
		return err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return err
	}
	if len(records) < 2 {
		return fmt.Errorf("expected header + at least one data row")
	}

	ds.Frameworks = trim(records[0][1:])
	for _, rec := range records[1:] {
		if len(rec) < 2 {
			continue
		}
		ds.Labels = append(ds.Labels, strings.TrimSpace(rec[0]))
		row := make([]float64, 0, len(rec)-1)
		for _, cell := range rec[1:] {
			v, _ := strconv.ParseFloat(strings.TrimSpace(cell), 64)
			row = append(row, v)
		}
		ds.Rows = append(ds.Rows, row)
	}
	return nil
}

func trim(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.TrimSpace(s)
	}
	return out
}
