package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var engines = [3]string{"bavix", "native", "tkpd"}

const (
	canvasW = 1100
	canvasH = 560

	plotLeft   = 90.0
	plotRight  = 1060.0
	plotTop    = 90.0
	plotBottom = 470.0
	plotWidth  = plotRight - plotLeft
	plotHeight = plotBottom - plotTop

	barWidth = 56.0
	barGap   = 12.0

	fontFamily = "-apple-system,BlinkMacSystemFont,Segoe UI,Helvetica,Arial,sans-serif"
)

func runChart() {
	outDir := "../docs/public/bench"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", outDir, err)
	}

	counts := sweepDirs()
	if len(counts) == 0 {
		log.Fatalf("no results-<count> directories found; run the bench first")
	}

	renderScalingCharts(counts, outDir)
	renderMissChart(counts, outDir)
	renderStartupChart(counts[0], outDir)
	renderLatencyChart(latencyDetailCount(counts), outDir)
	renderImageSizeChart(counts[len(counts)-1], outDir)

	log.Printf("wrote charts to %s", outDir)
}

func renderScalingCharts(counts []int, outDir string) {
	labels := make([]string, len(counts))
	metrics := map[string][3][]float64{}

	for _, key := range []string{"rps", "p99", "memory"} {
		metrics[key] = [3][]float64{make([]float64, len(counts)), make([]float64, len(counts)), make([]float64, len(counts))}
	}

	for i, count := range counts {
		dir := fmt.Sprintf("results-%d", count)
		labels[i] = humanCount(count)

		for k, engine := range engines {
			m := readJSON[meta](filepath.Join(dir, engine+"-meta.json"))
			r := readJSON[benchReport](filepath.Join(dir, engine+"-hit.json"))

			metrics["rps"][k][i] = peakRPS(filepath.Join(dir, engine+"-hit-throughput.json"))
			metrics["p99"][k][i] = r.latencyMs(99)
			metrics["memory"][k][i] = m.MemoryMB
		}
	}

	write(filepath.Join(outDir, "throughput-rps.svg"), renderChart(
		"Throughput vs Stub Count",
		"Peak requests/sec across the concurrency sweep. Higher is better",
		labels, newSeries(metrics["rps"][0], metrics["rps"][1], metrics["rps"][2]),
		func(v float64) string { return fmt.Sprintf("%.0f", v) },
	))

	write(filepath.Join(outDir, "latency-p99.svg"), renderChart(
		"p99 Latency vs Stub Count",
		"Tail latency at the highest concurrency level. Lower is better",
		labels, newSeries(metrics["p99"][0], metrics["p99"][1], metrics["p99"][2]),
		func(v float64) string { return fmt.Sprintf("%.1f ms", v) },
	))

	write(filepath.Join(outDir, "memory-usage.svg"), renderChart(
		"Memory vs Stub Count",
		"Resident memory with the stub set loaded",
		labels, newSeries(metrics["memory"][0], metrics["memory"][1], metrics["memory"][2]),
		func(v float64) string { return fmt.Sprintf("%.0f MB", v) },
	))

}

func renderMissChart(counts []int, outDir string) {
	labels := make([]string, len(counts))
	vals := [3][]float64{make([]float64, len(counts)), make([]float64, len(counts)), make([]float64, len(counts))}

	for i, count := range counts {
		dir := fmt.Sprintf("results-%d", count)
		labels[i] = humanCount(count)

		for k, engine := range engines {
			vals[k][i] = peakRPS(filepath.Join(dir, engine+"-miss-throughput.json"))
		}
	}

	write(filepath.Join(outDir, "throughput-miss.svg"), renderChart(
		"Throughput vs Stub Count -- No Match",
		"Peak requests/sec when no stub can match, so every stub is examined",
		labels, newSeries(vals[0], vals[1], vals[2]),
		func(v float64) string { return fmt.Sprintf("%.0f", v) },
	))
}

func latencyDetailCount(counts []int) int {
	const preferred = 1000

	best := counts[0]
	for _, count := range counts {
		if abs(count-preferred) < abs(best-preferred) {
			best = count
		}
	}

	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}

	return n
}

func renderLatencyChart(count int, outDir string) {
	dir := fmt.Sprintf("results-%d", count)
	labels := []string{"avg", "p50", "p95", "p99"}

	var vals [3][]float64

	for k, engine := range engines {
		r := readJSON[benchReport](filepath.Join(dir, engine+"-hit.json"))
		vals[k] = []float64{
			float64(r.Summary.AverageNs) / 1e6,
			r.latencyMs(50), r.latencyMs(95), r.latencyMs(99),
		}
	}

	bavixReport := readJSON[benchReport](filepath.Join(dir, "bavix-hit.json"))

	write(filepath.Join(outDir, "latency-percentiles.svg"), renderChart(
		"Latency Distribution",
		fmt.Sprintf("At %s stubs, concurrency=%s. Lower is better",
			humanCount(count), bavixReport.OptionsResolved["concurrency"].Value),
		labels, newSeries(vals[0], vals[1], vals[2]),
		func(v float64) string { return fmt.Sprintf("%.2f ms", v) },
	))
}

func renderStartupChart(count int, outDir string) {
	dir := fmt.Sprintf("results-%d", count)
	var vals [3][]float64

	var runs int

	for k, engine := range engines {
		m := readJSON[meta](filepath.Join(dir, engine+"-meta.json"))
		vals[k] = []float64{m.Startup.Min, m.Startup.Avg, m.Startup.Max}
		runs = m.Startup.Runs
	}

	write(filepath.Join(outDir, "startup-ready.svg"), renderChart(
		"Startup Time",
		fmt.Sprintf("Until the service answers gRPC reflection, one stub loaded, over %d runs", runs),
		[]string{"min", "avg", "max"},
		newSeries(vals[0], vals[1], vals[2]),
		func(v float64) string { return fmt.Sprintf("%.2f s", v) },
	))
}

func renderImageSizeChart(count int, outDir string) {
	dir := fmt.Sprintf("results-%d", count)
	bavixMeta := readJSON[meta](filepath.Join(dir, "bavix-meta.json"))
	tkpdMeta := readJSON[meta](filepath.Join(dir, "tkpd-meta.json"))

	if !comparableSizes(bavixMeta.SizeMB, tkpdMeta.SizeMB) {
		log.Printf("skipping image-size.svg: no registry manifest for %q or %q (an unpublished tag has none, so compressed per-platform sizes cannot be read)",
			bavixMeta.Image, tkpdMeta.Image)

		return
	}

	write(filepath.Join(outDir, "image-size.svg"), renderChart(
		"Docker Image Size (Compressed)",
		"Smaller image means faster pull and less CI overhead",
		[]string{"linux/amd64", "linux/arm64"},
		[]series{
			{id: "bavix", label: "bavix/gripmock", from: "#34d399", to: "#059669", value: "#86efac", legend: "#d1fae5",
				values: []float64{bavixMeta.SizeMB["amd64"], bavixMeta.SizeMB["arm64"]}},
			{id: "tkpd", label: "tkpd/gripmock", from: "#f59e0b", to: "#d97706", value: "#fcd34d", legend: "#fef3c7",
				values: []float64{tkpdMeta.SizeMB["amd64"], tkpdMeta.SizeMB["arm64"]}},
		},
		func(v float64) string { return fmt.Sprintf("%.2f MB", v) },
	))
}

func comparableSizes(a, b map[string]float64) bool {
	for _, sizes := range []map[string]float64{a, b} {
		if sizes["amd64"] == 0 || sizes["arm64"] == 0 {
			return false
		}
	}

	return true
}

func readJSON[T any](path string) T {
	var v T
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, &v); err != nil {
		log.Fatalf("parse %s: %v", path, err)
	}
	return v
}

func write(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}
}

type series struct {
	id     string
	label  string
	from   string
	to     string
	value  string
	legend string
	values []float64
}

func renderChart(title, subtitle string, categories []string, all []series, format func(float64) string) string {
	maxVal := 0.0

	for _, s := range all {
		for _, v := range s.values {
			maxVal = max(maxVal, v)
		}
	}

	if maxVal == 0 {
		maxVal = 1
	}

	scaleMax := maxVal * 1.2

	var b strings.Builder

	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, canvasW, canvasH, canvasW, canvasH)
	b.WriteString("\n<defs>\n")
	b.WriteString(`<linearGradient id="bg" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#0b1220"/><stop offset="100%" stop-color="#111827"/></linearGradient>` + "\n")

	for _, s := range all {
		fmt.Fprintf(&b, `<linearGradient id="%s" x1="0" y1="0" x2="0" y2="1"><stop offset="0%%" stop-color="%s"/><stop offset="100%%" stop-color="%s"/></linearGradient>`+"\n", s.id, s.from, s.to)
	}

	b.WriteString("</defs>\n")
	fmt.Fprintf(&b, `<rect x="0" y="0" width="%d" height="%d" fill="url(#bg)" rx="16"/>`+"\n", canvasW, canvasH)
	fmt.Fprintf(&b, `<text x="90" y="44" fill="#f9fafb" font-size="28" font-family="%s" font-weight="700">%s</text>`+"\n", fontFamily, escape(title))
	fmt.Fprintf(&b, `<text x="90" y="70" fill="#9ca3af" font-size="16" font-family="%s">%s</text>`+"\n", fontFamily, escape(subtitle))

	for i := 0; i <= 5; i++ {
		y := plotTop + float64(i)*(plotHeight/5)
		val := scaleMax * float64(5-i) / 5

		fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#1f2937" stroke-width="1"/>`+"\n", plotLeft, y, plotRight, y)
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="end" fill="#9ca3af" font-size="12" font-family="%s">%.1f</text>`+"\n", plotLeft-10, y+5, fontFamily, val)
	}

	slot := plotWidth / float64(len(categories))
	count := float64(len(all))
	width := min(barWidth, (slot-barGap*(count+1))/count)
	group := width*count + barGap*(count-1)
	margin := (slot - group) / 2

	for i, cat := range categories {
		slotStart := plotLeft + float64(i)*slot + margin

		for k, s := range all {
			x := slotStart + float64(k)*(width+barGap)
			h := (s.values[i] / scaleMax) * plotHeight
			y := plotBottom - h

			fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="url(#%s)" rx="8"/>`+"\n", x, y, width, h, s.id)
			fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="middle" fill="%s" font-size="11" font-family="%s">%s</text>`+"\n", x+width/2, y-8, s.value, fontFamily, format(s.values[i]))
		}

		fmt.Fprintf(&b, `<text x="%.1f" y="498" text-anchor="middle" fill="#e5e7eb" font-size="13" font-family="%s">%s</text>`+"\n", slotStart+group/2, fontFamily, escape(cat))
	}

	for k, s := range all {
		x := 90 + float64(k)*220

		fmt.Fprintf(&b, `<rect x="%.0f" y="514" width="14" height="14" fill="url(#%s)" rx="3"/>`+"\n", x, s.id)
		fmt.Fprintf(&b, `<text x="%.0f" y="526" fill="%s" font-size="13" font-family="%s">%s</text>`+"\n", x+22, s.legend, fontFamily, escape(s.label))
	}

	b.WriteString("</svg>\n")

	return b.String()
}

func newSeries(bavix, native, tkpd []float64) []series {
	return []series{
		{id: "bavix", label: "bavix/gripmock (Docker)", from: "#34d399", to: "#059669", value: "#86efac", legend: "#d1fae5", values: bavix},
		{id: "native", label: "bavix/gripmock (native)", from: "#60a5fa", to: "#2563eb", value: "#bfdbfe", legend: "#dbeafe", values: native},
		{id: "tkpd", label: "tkpd/gripmock (Docker)", from: "#f59e0b", to: "#d97706", value: "#fcd34d", legend: "#fef3c7", values: tkpd},
	}
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "gen" {
		if len(os.Args) != 5 {
			log.Fatalf("usage: %s gen <count> <stubs.json> <names.csv>", os.Args[0])
		}

		var count int
		if _, err := fmt.Sscanf(os.Args[2], "%d", &count); err != nil || count < 1 {
			log.Fatalf("invalid stub count %q", os.Args[2])
		}

		if err := generateDataset(count, os.Args[3], os.Args[4]); err != nil {
			log.Fatalf("generate dataset: %v", err)
		}

		return
	}

	runChart()
}

func sweepDirs() []int {
	entries, err := filepath.Glob("results-*")
	if err != nil {
		return nil
	}

	var counts []int

	for _, entry := range entries {
		var n int
		if _, err := fmt.Sscanf(entry, "results-%d", &n); err != nil {
			continue
		}

		if _, err := os.Stat(filepath.Join(entry, "bavix-hit-throughput.json")); err != nil {
			continue
		}

		counts = append(counts, n)
	}

	sort.Ints(counts)

	return counts
}

func peakRPS(path string) float64 {
	var best float64

	for _, point := range readJSON[[]throughputPoint](path) {
		best = max(best, point.RPS)
	}

	return best
}

func humanCount(n int) string {
	switch {
	case n >= 1_000_000 && n%1_000_000 == 0:
		return fmt.Sprintf("%dM", n/1_000_000)
	case n >= 1000 && n%1000 == 0:
		return fmt.Sprintf("%dk", n/1000)
	default:
		return fmt.Sprint(n)
	}
}
