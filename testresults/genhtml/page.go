package main

import "html/template"

// page is the HTML template, styled after the Vizb benchmark viewer
// (https://vizb.goptics.org): light/dark themes, line & bar charts, an optional
// logarithmic Y axis, and per-series sorting. Chart data is injected as JSON so
// the layout adapts to any number of frameworks/rows.
var page = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en" data-theme="light">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Go Web Framework Benchmark</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
<style>
  :root[data-theme="light"] {
    --bg: #f7f8fa; --panel: #ffffff; --border: #e4e7ec; --shadow: 0 1px 3px rgba(16,24,40,.06);
    --text: #101828; --muted: #667085; --accent: #4f46e5; --badge: #f2f4f7; --grid: #eceff3;
  }
  :root[data-theme="dark"] {
    --bg: #0f1420; --panel: #1a2233; --border: #2a3448; --shadow: none;
    --text: #e6edf3; --muted: #8b98ac; --accent: #7c9cff; --badge: #232d40; --grid: #2a3448;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; background: var(--bg); color: var(--text);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    line-height: 1.5;
  }
  header { max-width: 1400px; margin: 0 auto; padding: 32px 24px 8px; text-align: center; }
  h1 { margin: 0 0 6px; font-size: 24px; }
  header p { margin: 0; color: var(--muted); font-size: 14px; }
  .updated {
    display: inline-block; margin-top: 12px; padding: 5px 12px; font-size: 12.5px;
    color: var(--muted); background: var(--badge); border: 1px solid var(--border); border-radius: 999px;
  }
  .controls {
    max-width: 1400px; margin: 0 auto; padding: 20px 24px 8px;
    display: flex; gap: 10px; align-items: center; flex-wrap: wrap; justify-content: center;
  }
  .group { display: inline-flex; border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
  .group button { border: 0; border-radius: 0; }
  .group button + button { border-left: 1px solid var(--border); }
  .controls .lbl { color: var(--muted); font-size: 13px; margin-right: 2px; }
  button {
    background: var(--panel); color: var(--text); border: 1px solid var(--border);
    border-radius: 8px; padding: 8px 14px; font-size: 13.5px; cursor: pointer;
  }
  button.active { background: var(--accent); color: #fff; }
  .grid {
    max-width: 1400px; margin: 0 auto; padding: 12px 24px 48px;
    display: grid; grid-template-columns: 1fr; gap: 24px;
  }
  .card { background: var(--panel); border: 1px solid var(--border); border-radius: 12px; padding: 18px 22px 12px; box-shadow: var(--shadow); }
  .card-head { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
  .card h2 { margin: 0; font-size: 17px; }
  .badges { display: flex; gap: 6px; }
  .badge { font-size: 11.5px; color: var(--muted); background: var(--badge); border: 1px solid var(--border); border-radius: 6px; padding: 2px 8px; }
  .card .sub { margin: 4px 0 12px; color: var(--muted); font-size: 12.5px; }
  .chart-wrap { position: relative; height: 460px; }
  footer { max-width: 1400px; margin: 0 auto; padding: 0 24px 40px; text-align: center; color: var(--muted); font-size: 12.5px; }
  a { color: var(--accent); }
</style>
</head>
<body>
<header>
  <h1>Go Web Framework Benchmark</h1>
  <p>Performance comparison of Go web frameworks</p>
  <div class="updated">Updated: <strong id="updated"></strong></div>
</header>

<div class="controls">
  <span class="lbl">Chart</span>
  <div class="group">
    <button id="btnBar">Bar</button>
    <button id="btnLine" class="active">Line</button>
  </div>
  <span class="lbl">Y scale</span>
  <div class="group">
    <button id="btnLinear" class="active">Linear</button>
    <button id="btnLog">Log</button>
  </div>
  <span class="lbl">Sort</span>
  <div class="group">
    <button id="btnSortNone" class="active">None</button>
    <button id="btnSortAsc">Asc</button>
    <button id="btnSortDesc">Desc</button>
  </div>
  <button id="btnTheme" title="Toggle theme">🌙</button>
</div>

<div class="grid" id="grid"></div>

<footer>
  Generated from the CSV files in <code>testresults/</code> by <code>go run ./genhtml</code>
  (no gnuplot required). Styled after <a href="https://vizb.goptics.org">Vizb</a>. Source:
  <a href="https://github.com/smallnest/go-web-framework-benchmark">go-web-framework-benchmark</a>.
</footer>

<script>
const FRAMEWORKS = {{ .Frameworks }};
const DATASETS = {{ .Datasets }};

// colorFor returns a stable, visually distinct color per framework index.
// A curated palette covers the first 10; beyond that we walk the hue circle by
// the golden angle so even 40+ frameworks stay distinguishable.
const BASE = ["#4f46e5","#f97066","#12b76a","#f79009","#9e77ed","#ee46bc","#06aed4","#fb6514","#66c61c","#e31b54"];
const colorCache = {};
function colorFor(i) {
  if (i < BASE.length) return BASE[i];
  if (colorCache[i]) return colorCache[i];
  const hue = (i * 137.508) % 360;                 // golden-angle spacing
  const sat = 62 + (i % 3) * 8;                    // vary sat/light so neighbors differ
  const light = 52 + (i % 2) * 8;
  return (colorCache[i] = "hsl(" + hue.toFixed(0) + " " + sat + "% " + light + "%)");
}

let chartType = "line";
let yScale = "linear";
let sort = "none";      // none | asc | desc
const charts = [];

// orderFor returns the framework indices in display order for the given dataset.
// Sorting ranks frameworks by their mean value across all rows.
function orderFor(ds) {
  const idx = ds.frameworks.map((_, i) => i);
  if (sort === "none") return idx;
  const mean = i => ds.rows.reduce((s, r) => s + r[i], 0) / ds.rows.length;
  return idx.sort((a, b) => sort === "asc" ? mean(a) - mean(b) : mean(b) - mean(a));
}

function seriesFor(ds) {
  return orderFor(ds).map((fi, pos) => ({
    label: ds.frameworks[fi],
    data: ds.rows.map(r => r[fi]),
    backgroundColor: colorFor(fi),
    borderColor: colorFor(fi),
    borderWidth: 2, tension: 0.25, pointRadius: 2.5,
  }));
}

function cssVar(n) { return getComputedStyle(document.documentElement).getPropertyValue(n).trim(); }

function buildCharts() {
  const grid = document.getElementById("grid");
  grid.innerHTML = "";
  charts.length = 0;
  Chart.defaults.color = cssVar("--muted");
  Chart.defaults.borderColor = cssVar("--grid");

  DATASETS.forEach(ds => {
    const card = document.createElement("div");
    card.className = "card";
    card.innerHTML =
      '<div class="card-head"><h2>' + ds.title + '</h2>' +
      '<div class="badges"><span class="badge">Series: ' + ds.frameworks.length + '</span></div></div>' +
      '<p class="sub">' + ds.sub + '</p><div class="chart-wrap"><canvas></canvas></div>';
    grid.appendChild(card);

    const ctx = card.querySelector("canvas");
    // With many frameworks an index tooltip would list dozens of rows, so above
    // a threshold switch to a single-series (nearest point) tooltip.
    const many = ds.frameworks.length > 12;
    charts.push(new Chart(ctx, {
      type: chartType,
      data: { labels: ds.labels, datasets: seriesFor(ds) },
      options: {
        responsive: true, maintainAspectRatio: false,
        interaction: many ? { mode: "nearest", intersect: false }
                          : { mode: "index", intersect: false },
        plugins: {
          legend: { position: "top", labels: { boxWidth: 8, usePointStyle: true, pointStyle: "circle", padding: 8, font: { size: 11 } } },
          tooltip: { callbacks: { label: c => c.dataset.label + ": " + c.parsed.y.toLocaleString() + " " + ds.unit } }
        },
        scales: {
          y: {
            type: yScale, beginAtZero: yScale === "linear",
            grid: { color: cssVar("--grid") },
            ticks: { callback: v => Number(v).toLocaleString() }
          },
          x: { grid: { display: false } }
        }
      }
    }));
  });
}

function bind(id, fn) { document.getElementById(id).onclick = fn; }
function activate(ids, on) { ids.forEach(i => document.getElementById(i).classList.toggle("active", i === on)); }

bind("btnBar",  () => { chartType = "bar";  activate(["btnBar","btnLine"], "btnBar");  buildCharts(); });
bind("btnLine", () => { chartType = "line"; activate(["btnBar","btnLine"], "btnLine"); buildCharts(); });
bind("btnLinear", () => { yScale = "linear";      activate(["btnLinear","btnLog"], "btnLinear"); buildCharts(); });
bind("btnLog",    () => { yScale = "logarithmic"; activate(["btnLinear","btnLog"], "btnLog");    buildCharts(); });
bind("btnSortNone", () => { sort = "none"; activate(["btnSortNone","btnSortAsc","btnSortDesc"], "btnSortNone"); buildCharts(); });
bind("btnSortAsc",  () => { sort = "asc";  activate(["btnSortNone","btnSortAsc","btnSortDesc"], "btnSortAsc");  buildCharts(); });
bind("btnSortDesc", () => { sort = "desc"; activate(["btnSortNone","btnSortAsc","btnSortDesc"], "btnSortDesc"); buildCharts(); });
bind("btnTheme", () => {
  const root = document.documentElement;
  const dark = root.getAttribute("data-theme") === "dark";
  root.setAttribute("data-theme", dark ? "light" : "dark");
  document.getElementById("btnTheme").textContent = dark ? "🌙" : "☀️";
  buildCharts();
});

document.getElementById("updated").textContent =
  new Date().toLocaleString(undefined, { year: "numeric", month: "short", day: "numeric", hour: "numeric", minute: "2-digit" });
buildCharts();
</script>
</body>
</html>
`))
