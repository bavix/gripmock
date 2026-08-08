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
	renderScenarioChart(counts, outDir, "miss", "throughput-miss.svg",
		"Throughput vs Stub Count -- No Match",
		"Peak requests/sec when no stub can match, so every stub is examined")
	renderMatcherChart(counts[0], outDir)
	renderCPUChart(counts, outDir)
	renderStartupChart(counts[0], outDir)
	renderLatencyChart(counts[0], outDir)
	renderImageSizeChart(counts[len(counts)-1], outDir)

	log.Printf("wrote charts to %s", outDir)
}

func renderScalingCharts(counts []int, outDir string) {
	labels := make([]string, len(counts))
	metrics := map[string][3][]float64{}

	for _, key := range []string{"rps", "memory"} {
		metrics[key] = [3][]float64{make([]float64, len(counts)), make([]float64, len(counts)), make([]float64, len(counts))}
	}

	for i, count := range counts {
		dir := fmt.Sprintf("results-%d", count)
		labels[i] = humanCount(count)

		for k, engine := range engines {
			m := readJSON[meta](filepath.Join(dir, engine+"-meta.json"))

			metrics["rps"][k][i] = peakRPS(filepath.Join(dir, engine+"-equals-throughput.json"))
			metrics["memory"][k][i] = m.MemoryMB
		}
	}

	write(filepath.Join(outDir, "throughput-equals.svg"), renderChart(
		"Throughput vs Stub Count",
		"Peak requests/sec across the concurrency sweep. Higher is better",
		labels, newSeries(metrics["rps"][0], metrics["rps"][1], metrics["rps"][2]),
		func(v float64) string { return fmt.Sprintf("%.0f", v) },
	))

	write(filepath.Join(outDir, "memory-usage.svg"), renderChart(
		"Memory vs Stub Count",
		"Resident memory with the stub set loaded",
		labels, newSeries(metrics["memory"][0], metrics["memory"][1], metrics["memory"][2]),
		func(v float64) string { return fmt.Sprintf("%.0f MB", v) },
	))

}

func renderScenarioChart(counts []int, outDir, scenario, file, title, subtitle string) {
	labels := make([]string, len(counts))
	vals := [3][]float64{make([]float64, len(counts)), make([]float64, len(counts)), make([]float64, len(counts))}

	for i, count := range counts {
		dir := fmt.Sprintf("results-%d", count)
		labels[i] = humanCount(count)

		for k, engine := range engines {
			vals[k][i] = peakRPS(filepath.Join(dir, engine+"-"+scenario+"-throughput.json"))
		}
	}

	write(filepath.Join(outDir, file), renderChart(
		title, subtitle,
		labels, newSeries(vals[0], vals[1], vals[2]),
		func(v float64) string { return fmt.Sprintf("%.0f", v) },
	))
}

// renderCPUChart draws throughput per core. Raw consumption is not comparable
// on its own -- an engine serving far more requests is expected to consume
// more -- so the work done is divided out.
func renderCPUChart(counts []int, outDir string) {
	labels := make([]string, len(counts))
	perCore := [3][]float64{}

	for k := range engines {
		perCore[k] = make([]float64, len(counts))
	}

	for i, count := range counts {
		dir := fmt.Sprintf("results-%d", count)
		labels[i] = humanCount(count)

		for k, engine := range engines {
			u, ok := readJSONOpt[usage](filepath.Join(dir, engine+"-equals-usage.json"))
			if !ok {
				log.Printf("skipping the CPU chart: no usage samples for %s at %s stubs", engine, humanCount(count))

				return
			}

			if cores := u.CPUAvg / 100; cores > 0 {
				perCore[k][i] = peakRPS(filepath.Join(dir, engine+"-equals-throughput.json")) / cores
			}
		}
	}

	write(filepath.Join(outDir, "efficiency-cpu.svg"), renderChart(
		"CPU Efficiency",
		"Requests per second delivered per CPU core consumed. Higher is better",
		labels, newSeries(perCore[0], perCore[1], perCore[2]),
		func(v float64) string { return fmt.Sprintf("%.0f", v) },
	))
}

func renderMatcherChart(count int, outDir string) {
	dir := fmt.Sprintf("results-%d", count)
	scenarios := []string{"equals", "contains", "matches"}

	var vals [3][]float64

	for k, engine := range engines {
		vals[k] = make([]float64, len(scenarios))
		for i, scenario := range scenarios {
			vals[k][i] = peakRPS(filepath.Join(dir, engine+"-"+scenario+"-throughput.json"))
		}
	}

	write(filepath.Join(outDir, "matcher-kinds.svg"), renderChart(
		"Matcher Kinds",
		fmt.Sprintf("Peak requests/sec at %s stubs, by the matcher every stub is written with. Higher is better", humanCount(count)),
		[]string{"equals", "contains", "matches"},
		newSeries(vals[0], vals[1], vals[2]),
		func(v float64) string { return fmt.Sprintf("%.0f", v) },
	))
}

func renderLatencyChart(count int, outDir string) {
	dir := fmt.Sprintf("results-%d", count)
	labels := []string{"avg", "p50", "p95", "p99"}

	var vals [3][]float64

	for k, engine := range engines {
		r := readJSON[benchReport](filepath.Join(dir, engine+"-equals.json"))
		vals[k] = []float64{
			float64(r.Summary.AverageNs) / 1e6,
			r.latencyMs(50), r.latencyMs(95), r.latencyMs(99),
		}
	}

	bavixReport := readJSON[benchReport](filepath.Join(dir, "bavix-equals.json"))

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
	dir := "results-size"
	if _, err := os.Stat(filepath.Join(dir, "bavix-meta.json")); err != nil {
		dir = fmt.Sprintf("results-%d", count)
	}

	bavixMeta := readJSON[meta](filepath.Join(dir, "bavix-meta.json"))
	tkpdMeta := readJSON[meta](filepath.Join(dir, "tkpd-meta.json"))

	if !comparableSizes(bavixMeta.SizeMB, tkpdMeta.SizeMB) {
		log.Printf("skipping image-size.svg: no registry manifest for %q or %q (an unpublished tag has none, so compressed per-platform sizes cannot be read)",
			bavixMeta.Image, tkpdMeta.Image)

		return
	}

	write(filepath.Join(outDir, "image-size.svg"), renderChart(
		"Docker Image Size (Compressed)",
		fmt.Sprintf("%s vs %s, compressed layers from the registry manifest", bavixMeta.Image, tkpdMeta.Image),
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

// readJSONOpt returns the zero value when the file is absent, for artefacts a
// results directory may predate.
func readJSONOpt[T any](path string) (T, bool) {
	var v T

	data, err := os.ReadFile(path)
	if err != nil {
		return v, false
	}

	if err := json.Unmarshal(data, &v); err != nil {
		log.Fatalf("parse %s: %v", path, err)
	}

	return v, true
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

// axis maps a value to a fraction of the plot height. It is deliberately
// linear: a bar reads as length from zero, so proportions on the page match
// proportions in the data. A log axis would make a value a thousand times
// smaller look merely a third shorter.
type axis struct {
	max float64
}

// minVisible keeps a bar for a very small value from vanishing entirely, so
// its printed figure still has something to sit on.
const minVisible = 0.005

func newAxis(all []series) axis {
	maxVal := 0.0

	for _, s := range all {
		for _, v := range s.values {
			maxVal = max(maxVal, v)
		}
	}

	if maxVal <= 0 {
		return axis{max: 1}
	}

	return axis{max: maxVal * 1.2}
}

// fraction returns where value sits on the axis, in [0, 1].
func (a axis) fraction(value float64) float64 {
	if value <= 0 {
		return 0
	}

	return max(minVisible, value/a.max)
}

// ticks returns the gridline values from the top of the axis down.
func (a axis) ticks() []float64 {
	const steps = 5

	values := make([]float64, 0, steps+1)
	for i := range steps + 1 {
		values = append(values, a.max*float64(steps-i)/steps)
	}

	return values
}

func renderChart(title, subtitle string, categories []string, all []series, format func(float64) string) string {
	scale := newAxis(all)

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

	gridlines := scale.ticks()

	for i, value := range gridlines {
		y := plotTop + float64(i)*(plotHeight/float64(len(gridlines)-1))

		fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#1f2937" stroke-width="1"/>`+"\n", plotLeft, y, plotRight, y)
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="end" fill="#9ca3af" font-size="12" font-family="%s">%s</text>`+"\n",
			plotLeft-10, y+5, fontFamily, format(value))
	}

	drawBars(&b, scale, categories, all, format)

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

// drawBars encodes each value as length from the axis floor. That reading only
// holds on a linear axis, where the floor is zero.
func drawBars(b *strings.Builder, scale axis, categories []string, all []series, format func(float64) string) {
	slot := plotWidth / float64(len(categories))
	count := float64(len(all))
	width := min(barWidth, (slot-barGap*(count+1))/count)
	group := width*count + barGap*(count-1)
	margin := (slot - group) / 2

	for i, cat := range categories {
		slotStart := plotLeft + float64(i)*slot + margin

		for k, s := range all {
			x := slotStart + float64(k)*(width+barGap)
			h := scale.fraction(s.values[i]) * plotHeight
			y := plotBottom - h

			fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="url(#%s)" rx="8"/>`+"\n", x, y, width, h, s.id)
			fmt.Fprintf(b, `<text x="%.1f" y="%.1f" text-anchor="middle" fill="%s" font-size="11" font-family="%s">%s</text>`+"\n", x+width/2, y-8, s.value, fontFamily, format(s.values[i]))
		}

		fmt.Fprintf(b, `<text x="%.1f" y="498" text-anchor="middle" fill="#e5e7eb" font-size="13" font-family="%s">%s</text>`+"\n", slotStart+group/2, fontFamily, escape(cat))
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

		if _, err := os.Stat(filepath.Join(entry, "bavix-equals-throughput.json")); err != nil {
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
