# Build report.html from persisted JSON artifacts (phase 2).

(import ./json_decode :as decode)

(defn- digit-byte?
  [b]
  (and (>= b (chr "0")) (<= b (chr "9"))))

(defn- month-file?
  [name]
  (and (= (length name) 12)
       (digit-byte? (get name 0))
       (digit-byte? (get name 1))
       (digit-byte? (get name 2))
       (digit-byte? (get name 3))
       (= (get name 4) (chr "-"))
       (digit-byte? (get name 5))
       (digit-byte? (get name 6))
       (= (string/slice name 7) ".json")))

(defn- escape-html
  "Escape unsafe HTML characters in dynamic text content."
  [s]
  (def t (if (string? s) s (string s)))
  (def buf @"")
  (each b t
    (cond
      (= b (chr "&")) (buffer/push-string buf "&amp;")
      (= b (chr "<")) (buffer/push-string buf "&lt;")
      (= b (chr ">")) (buffer/push-string buf "&gt;")
      (= b (chr "\"")) (buffer/push-string buf "&quot;")
      (= b (chr "'")) (buffer/push-string buf "&#39;")
      (buffer/push-byte buf b)))
  (string buf))

(defn- write-file
  "Write string to file atomically (write to temp, then rename)."
  [path content]
  (def tmp (string path ".tmp"))
  (spit tmp content)
  (os/rename tmp path))

(defn- load-json
  [path]
  (decode/decode (slurp path)))

(defn- maybe-load-json
  [path]
  (try
    (load-json path)
    ([err] nil)))

(defn- load-report-data
  "Load monthly JSON files (+ optional lifetime) from a directory."
  [json-dir]
  (def month-files (sort (filter month-file? (os/dir json-dir))))
  (when (empty? month-files)
    (error (string "No month JSON files found in " json-dir)))

  (def monthly-json @[])
  (def hourly-json @[])

  (each fname month-files
    (def doc (load-json (string json-dir "/" fname)))
    (def summary (get doc :summary @{}))
    (when (nil? (get summary :month nil))
      (put summary :month (string/slice fname 0 7)))
    (array/push monthly-json summary)
    (each h (get doc :hourly @[])
      (array/push hourly-json h)))

  (def monthly-sorted (sort-by |(or ($ :month) "") monthly-json))
  (def hourly-sorted (sort-by |(or ($ :hour) "") hourly-json))
  (def lifetime (maybe-load-json (string json-dir "/lifetime.json")))

  (def total-requests
    (if (and lifetime (not (nil? (get lifetime :total_requests nil))))
      (lifetime :total_requests)
      (sum (map |(or ($ :requests) 0) monthly-sorted))))

  (def total-uniques
    (if (and lifetime (not (nil? (get lifetime :total_unique_visitors_estimate nil))))
      (lifetime :total_unique_visitors_estimate)
      0))

  (def generated-at
    (if lifetime
      (get lifetime :generated_at (os/strftime "%Y-%m-%dT%H:%M:%SZ" (os/time)))
      (os/strftime "%Y-%m-%dT%H:%M:%SZ" (os/time))))

  (def log-files
    (if (and lifetime (or (array? (get lifetime :log_files_processed nil))
                          (tuple? (get lifetime :log_files_processed nil))))
      (lifetime :log_files_processed)
      @[]))

  @{:monthly-json monthly-sorted
    :hourly-json hourly-sorted
    :total-requests total-requests
    :total-uniques total-uniques
    :generated-at generated-at
    :log-files log-files})

(def DAYS-BEFORE-MONTH @[0 31 59 90 120 151 181 212 243 273 304 334])

(defn- round-int
  [x]
  (math/floor (+ x 0.5)))

(defn- leap-year?
  [y]
  (or (and (= (% y 4) 0) (not= (% y 100) 0))
      (= (% y 400) 0)))

(defn- parse-day-ymd
  [day]
  (when (and (string? day) (>= (length day) 10))
    (let [y (scan-number (string/slice day 0 4))
          m (scan-number (string/slice day 5 7))
          d (scan-number (string/slice day 8 10))]
      (when (and (number? y) (number? m) (number? d)
                 (>= m 1) (<= m 12)
                 (>= d 1) (<= d 31))
        @[y m d]))))

(defn- day-ordinal
  "Convert YYYY-MM-DD to a comparable day ordinal."
  [day]
  (when-let [ymd (parse-day-ymd day)]
    (let [y (get ymd 0)
          m (get ymd 1)
          d (get ymd 2)
          y1 (- y 1)]
      (+ (* 365 y1)
         (math/floor (/ y1 4))
         (- (math/floor (/ y1 100)))
         (math/floor (/ y1 400))
         (get DAYS-BEFORE-MONTH (- m 1) 0)
         (if (and (> m 2) (leap-year? y)) 1 0)
         d))))

(defn- suspicious-path?
  [path]
  (if (string? path)
    (let [p (string/ascii-lower path)]
      (or (string/find "wp-" p)
          (string/find "xmlrpc" p)
          (string/find ".git" p)
          (string/find ".env" p)
          (string/find "phpmyadmin" p)
          (string/find "wflogs" p)
          (string/find "cgi-bin" p)
          (string/find "debug.log" p)
          (string/find "vendor/" p)
          (string/find "admin" p)
          (string/find "login" p)
          (string/find ".php" p)))
    false))

(defn- top-pairs
  [tbl limit]
  (take limit (sort-by |(- (get $ 1)) (pairs tbl))))

(defn- status-summary
  [tbl]
  (if (empty? tbl)
    "-"
    (string/join
      (map (fn [[s c]] (string/format "%s: %d" s c))
           (sort-by |(- (get $ 1)) (pairs tbl)))
      ", ")))

(defn- pct
  [part whole]
  (if (> whole 0)
    (/ (* 100 part) whole)
    0))

(defn- delta-pct
  [current previous]
  (if (> previous 0)
    (/ (* 100 (- current previous)) previous)
    nil))

(defn- delta-class
  [current previous]
  (if (or (nil? previous) (<= previous 0))
    "neutral"
    (cond
      (> current previous) "up"
      (< current previous) "down"
      true "neutral")))

(defn- format-delta
  [current previous label]
  (if (nil? label)
    ""
    (if (and previous (> previous 0))
      (string/format
        "<span class=\"delta %s\">%+.1f%% vs prev %s</span>"
        (delta-class current previous)
        (delta-pct current previous)
        (escape-html label))
      (string/format "<span class=\"delta neutral\">No prior %s data</span>" (escape-html label)))))

(defn- filter-hours-window
  "Return hourly entries where day diff from latest is in [start, start+len)."
  [hourly-json latest-ord start len]
  (if (nil? latest-ord)
    (if (= start 0) hourly-json @[])
    (filter
      (fn [entry]
        (when (string? (entry :hour))
          (when-let [ord (day-ordinal (string/slice (entry :hour) 0 10))]
            (let [diff (- latest-ord ord)]
              (and (>= diff start) (< diff (+ start len)))))))
      hourly-json)))

(defn- aggregate-hourly
  [entries]
  (var requests 0)
  (var visitors 0)
  (var scans 0)
  (var bots 0)
  (def daily @{})
  (def daily-visitors @{})
  (def status-all @{})
  (def status-visitors @{})
  (def status-scans @{})
  (def status-bots @{})
  (def paths @{})
  (def referrers @{})
  (def referrers-visitors @{})
  (var has-referrer-by-type false)
  (def browsers @{})
  (def platforms @{})

  (each entry entries
    (def req (or (entry :requests) 0))
    (+= requests req)
    (+= visitors (or (entry :visitors) 0))
    (+= scans (or (entry :scans) 0))
    (+= bots (or (entry :bots) 0))

    (when (string? (entry :hour))
      (def day (string/slice (entry :hour) 0 10))
      (put daily day (+ (get daily day 0) req))
      (put daily-visitors day (+ (get daily-visitors day 0) (or (entry :visitors) 0))))

    (eachp [k v] (get entry :status_codes @{})
      (put status-all k (+ (get status-all k 0) v)))
    (eachp [k v] (get (get entry :status_by_type @{}) :visitors @{})
      (put status-visitors k (+ (get status-visitors k 0) v)))
    (eachp [k v] (get (get entry :status_by_type @{}) :scans @{})
      (put status-scans k (+ (get status-scans k 0) v)))
    (eachp [k v] (get (get entry :status_by_type @{}) :bots @{})
      (put status-bots k (+ (get status-bots k 0) v)))

    (each p (get entry :top_paths @[])
      (def path (p :path))
      (when (and (string? path) (not= path "other"))
        (put paths path (+ (get paths path 0) (or (p :requests) 0)))))

    (eachp [k v] (get entry :referrer_categories @{})
      (put referrers k (+ (get referrers k 0) v)))
    (def ref-by-type (get entry :referrer_categories_by_type nil))
    (when ref-by-type
      (set has-referrer-by-type true))
    (eachp [k v] (get ref-by-type :visitors @{})
      (put referrers-visitors k (+ (get referrers-visitors k 0) v)))
    (eachp [k v] (get entry :browsers @{})
      (put browsers k (+ (get browsers k 0) v)))
    (eachp [k v] (get entry :platforms @{})
      (put platforms k (+ (get platforms k 0) v))))

  (def sorted-days (sort (keys daily)))
  (def daily-pairs @[])
  (def daily-visitor-pairs @[])
  (var max-day "")
  (var max-count 0)
  (var max-visitor-day "")
  (var max-visitor-count 0)
  (each day sorted-days
    (def count (get daily day 0))
    (def vcount (get daily-visitors day 0))
    (array/push daily-pairs @[day count])
    (array/push daily-visitor-pairs @[day vcount])
    (when (> count max-count)
      (set max-count count)
      (set max-day day))
    (when (> vcount max-visitor-count)
      (set max-visitor-count vcount)
      (set max-visitor-day day)))

  @{:requests requests
    :visitors visitors
    :scans scans
    :bots bots
    :daily-pairs daily-pairs
    :max-day max-day
    :max-count max-count
    :daily-visitor-pairs daily-visitor-pairs
    :max-visitor-day max-visitor-day
    :max-visitor-count max-visitor-count
    :status-all status-all
    :status-visitors status-visitors
    :status-scans status-scans
    :status-bots status-bots
    :paths paths
    :referrers referrers
    :referrers-visitors referrers-visitors
    :has-referrer-by-type has-referrer-by-type
    :browsers browsers
    :platforms platforms})

(defn- estimate-window-uniques
  "Approximate unique visitors for a window by prorating monthly estimates."
  [entries monthly-json]
  (def month-requests @{})
  (each entry entries
    (when (string? (entry :hour))
      (def month (string/slice (entry :hour) 0 7))
      (put month-requests month (+ (get month-requests month 0) (or (entry :requests) 0)))))

  (var total 0)
  (each m monthly-json
    (def month (or (m :month) ""))
    (def req-month (or (m :requests) 0))
    (def req-window (get month-requests month 0))
    (def uniq-est (or (m :unique_visitors_estimate) (m :unique_visitors_exact) 0))
    (when (and (> req-month 0) (> req-window 0) (> uniq-est 0))
      (+= total (round-int (* uniq-est (/ req-window req-month))))))
  total)

(defn- split-path-groups
  [paths]
  (def human @[])
  (def suspicious @[])
  (each kv (pairs paths)
    (if (suspicious-path? (get kv 0))
      (array/push suspicious kv)
      (array/push human kv)))
  @{:human (take 12 (sort-by |(- (get $ 1)) human))
    :suspicious (take 12 (sort-by |(- (get $ 1)) suspicious))})

(defn- render-daily-chart
  [daily-pairs max-day max-count]
  (if (empty? daily-pairs)
    "<p class=\"meta\">No daily data available for this window.</p>\n"
    (let [width 760
          height 190
          pad 24
          baseline (- height pad)
          n (length daily-pairs)
          x-span (- width (* 2 pad))
          y-span (- height (* 2 pad))
          x-step (if (> n 1) (/ x-span (- n 1)) 0)
          y-max (if (> max-count 0) max-count 1)
          points @""
          area @""
          first-day (get (get daily-pairs 0) 0 "")
          last-day (get (get daily-pairs (- n 1)) 0 "")]
      (var idx 0)
      (var last-x 0)
      (each pair daily-pairs
        (def day (get pair 0 ""))
        (def count (get pair 1 0))
        (def x (+ pad (* idx x-step)))
        (def y (- baseline (* (/ count y-max) y-span)))
        (when (= idx 0)
          (buffer/push-string area (string/format "M %.1f %.1f L %.1f %.1f " x baseline x y)))
        (when (> idx 0)
          (buffer/push-string area (string/format "L %.1f %.1f " x y)))
        (set last-x x)
        (buffer/push-string points (string/format "%.1f,%.1f " x y))
        (++ idx))
      (buffer/push-string area (string/format "L %.1f %.1f Z" last-x baseline))
      (string/format
        "<p class=\"meta\">Peak day: <strong>%s</strong> (%d requests)</p>\n<div class=\"chart-wrap\"><div class=\"chart-y-axis\"><span>%d</span><span>0</span></div><svg class=\"chart-svg\" viewBox=\"0 0 %d %d\" preserveAspectRatio=\"none\"><path class=\"area\" d=\"%s\"></path><polyline class=\"line\" points=\"%s\"></polyline></svg><div class=\"chart-axis\"><span>%s</span><span>%s</span></div></div>\n"
        (escape-html max-day)
        max-count
        max-count
        width
        height
        (escape-html (string area))
        (escape-html (string points))
        (escape-html first-day)
        (escape-html last-day)))))

(defn build-html-report
  "Generate self-contained HTML report from JSON-derived monthly/hourly docs."
  [monthly-json hourly-json total-requests total-uniques generated-at log-files]
  (def day-seen @{})
  (def days @[])
  (each entry hourly-json
    (when (string? (entry :hour))
      (def day (string/slice (entry :hour) 0 10))
      (when (nil? (get day-seen day nil))
        (put day-seen day true)
        (array/push days day))))
  (def sorted-days (sort days))
  (def latest-day
    (if (empty? sorted-days)
      nil
      (get sorted-days (- (length sorted-days) 1))))
  (def latest-ord
    (if (nil? latest-day) nil (day-ordinal latest-day)))

  (def hours-all hourly-json)
  (def hours-30 (filter-hours-window hourly-json latest-ord 0 30))
  (def hours-7 (filter-hours-window hourly-json latest-ord 0 7))
  (def prev-30 (filter-hours-window hourly-json latest-ord 30 30))
  (def prev-7 (filter-hours-window hourly-json latest-ord 7 7))

  (def agg-all (aggregate-hourly hours-all))
  (def agg-30 (aggregate-hourly hours-30))
  (def agg-7 (aggregate-hourly hours-7))
  (def agg-prev-30 (aggregate-hourly prev-30))
  (def agg-prev-7 (aggregate-hourly prev-7))

  (def uniq-30 (estimate-window-uniques hours-30 monthly-json))
  (def uniq-7 (estimate-window-uniques hours-7 monthly-json))
  (def uniq-prev-30 (estimate-window-uniques prev-30 monthly-json))
  (def uniq-prev-7 (estimate-window-uniques prev-7 monthly-json))
  (def uniq-all
    (if (> total-uniques 0)
      total-uniques
      (estimate-window-uniques hours-all monthly-json)))

  (def summary-agg (if (> (agg-30 :requests) 0) agg-30 agg-all))
  (def summary-uniq (if (> (agg-30 :requests) 0) uniq-30 uniq-all))
  (def summary-label (if (> (agg-30 :requests) 0) "Last 30 days" "All time"))
  (def summary-human (pct (summary-agg :visitors) (summary-agg :requests)))
  (def current-month
    (if (and (string? generated-at) (>= (length generated-at) 7))
      (string/slice generated-at 0 7)
      (if (empty? monthly-json)
        ""
        (or ((get monthly-json (- (length monthly-json) 1)) :month) ""))))

  (defn- render-window-panel
    [buf id title agg prev-agg uniques prev-uniques prev-label default? show-delta?]
    (def requests (agg :requests))
    (def visitors (agg :visitors))
    (def scans (agg :scans))
    (def bots (agg :bots))
    (def human-share (pct visitors requests))
    (def scan-share (pct (+ scans bots) requests))
    (def seg-visitors (pct visitors requests))
    (def seg-scans (pct scans requests))
    (def seg-bots (pct bots requests))
    (def paths (split-path-groups (agg :paths)))
    (def human-paths (paths :human))
    (def suspicious-paths (paths :suspicious))
    (def scan-signatures (take 6 suspicious-paths))
    (def prev-requests (prev-agg :requests))
    (def prev-human-share (pct (prev-agg :visitors) (prev-agg :requests)))
    (def request-delta (if show-delta? (format-delta requests prev-requests prev-label) ""))
    (def unique-delta (if show-delta? (format-delta uniques prev-uniques prev-label) ""))
    (def visitor-share-delta (if show-delta? (format-delta human-share prev-human-share prev-label) ""))

    (buffer/push-string buf
      (string/format
        "<section class=\"window-panel%s\" data-window=\"%s\">\n<h2>%s</h2>\n<div class=\"cards\">\n"
        (if default? " active" "")
        (escape-html id)
        (escape-html title)))

    (buffer/push-string buf
      (string/format
        "<div class=\"card\"><div class=\"label\">Requests</div><div class=\"value\">%d</div>%s</div>\n"
        requests
        request-delta))
    (buffer/push-string buf
      (string/format
        "<div class=\"card\"><div class=\"label\">Approx Unique Visitors</div><div class=\"value\">%d</div>%s</div>\n"
        uniques
        unique-delta))
    (buffer/push-string buf
      (string/format
        "<div class=\"card\"><div class=\"label\">Visitor-Classified Traffic</div><div class=\"value\">%.1f%%</div>%s</div>\n"
        human-share
        visitor-share-delta))
    (buffer/push-string buf
      (string/format
        "<div class=\"card\"><div class=\"label\">Scan/Bot Share</div><div class=\"value\">%.1f%%</div></div>\n"
        scan-share))
    (buffer/push-string buf "</div>\n")

    (buffer/push-string buf "<h3>Traffic Quality</h3>\n")
    (buffer/push-string buf
      (string/format
        "<div class=\"stacked\"><span class=\"seg seg-visitors\" style=\"width:%.2f%%\"></span><span class=\"seg seg-scans\" style=\"width:%.2f%%\"></span><span class=\"seg seg-bots\" style=\"width:%.2f%%\"></span></div>\n"
        seg-visitors seg-scans seg-bots))
    (buffer/push-string buf
      (string/format
        "<p class=\"meta\">Visitor-classified %.1f%% • Scans %.1f%% • Bots %.1f%%</p>\n"
        seg-visitors seg-scans seg-bots))
    (buffer/push-string buf "<table>\n<tr><th>Type</th><th>Requests</th><th>%</th><th>Status Codes</th></tr>\n")
    (each [label count status-tbl]
          [["Visitor-classified" visitors (agg :status-visitors)]
           ["Scans" scans (agg :status-scans)]
           ["Bots" bots (agg :status-bots)]]
      (buffer/push-string buf
        (string/format
          "<tr><td>%s</td><td class=\"num\">%d</td><td class=\"num\">%.1f%%</td><td>%s</td></tr>\n"
          (escape-html label)
          count
          (pct count requests)
          (escape-html (status-summary status-tbl)))))
    (buffer/push-string buf
      (string/format
        "<tr><td><strong>Total</strong></td><td class=\"num\"><strong>%d</strong></td><td class=\"num\"><strong>100%%</strong></td><td>%s</td></tr>\n"
        requests
        (escape-html (status-summary (agg :status-all)))))
    (buffer/push-string buf "</table>\n")

    (buffer/push-string buf "<h3>Top Scan Signatures</h3>\n")
    (if (empty? scan-signatures)
      (buffer/push-string buf "<p class=\"meta\">No suspicious signatures in this window.</p>\n")
      (do
        (buffer/push-string buf "<ul class=\"scan-list\">\n")
        (each [path count] scan-signatures
          (buffer/push-string buf
            (string/format "<li><code>%s</code> <span class=\"num\">(%d)</span></li>\n"
              (escape-html path) count)))
        (buffer/push-string buf "</ul>\n")))

    (buffer/push-string buf "<h3>Daily Requests</h3>\n")
    (buffer/push-string buf
      (render-daily-chart (agg :daily-pairs) (agg :max-day) (agg :max-count)))
    (buffer/push-string buf "<h4>Daily Visitor-Classified Requests</h4>\n")
    (buffer/push-string buf
      (render-daily-chart
        (agg :daily-visitor-pairs)
        (agg :max-visitor-day)
        (agg :max-visitor-count)))

    (buffer/push-string buf "<h3>Top Paths</h3>\n<div class=\"split-grid\">\n")
    (buffer/push-string buf "<div><h4>Visitor-Classified Pages</h4>\n<table>\n<tr><th>Path</th><th>Requests</th></tr>\n")
    (if (empty? human-paths)
      (buffer/push-string buf "<tr><td colspan=\"2\" class=\"meta\">No visitor-classified paths in this window.</td></tr>\n")
      (each [path count] human-paths
        (buffer/push-string buf
          (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n"
            (escape-html path) count))))
    (buffer/push-string buf "</table></div>\n")

    (buffer/push-string buf "<div><h4>Scan/Bot Targets</h4>\n<table>\n<tr><th>Path</th><th>Requests</th></tr>\n")
    (if (empty? suspicious-paths)
      (buffer/push-string buf "<tr><td colspan=\"2\" class=\"meta\">No suspicious targets in this window.</td></tr>\n")
      (each [path count] suspicious-paths
        (buffer/push-string buf
          (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n"
            (escape-html path) count))))
    (buffer/push-string buf "</table></div>\n</div>\n")

    (buffer/push-string buf "</section>\n"))

  (def buf @"")
  (buffer/push-string buf
    ```
    <!DOCTYPE html>
    <html lang="en">
    <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Log Analysis Report</title>
    <style>
    :root {
      --bg: #fff; --fg: #1a1a1a; --muted: #666; --border: #e0e0e0;
      --accent: #2563eb; --bar-bg: #e5e7eb; --card-bg: #f9fafb;
      --ok: #059669; --warn: #dc2626; --neutral: #6b7280;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #111; --fg: #e5e5e5; --muted: #999; --border: #333;
        --accent: #60a5fa; --bar-bg: #1f2937; --card-bg: #1a1a2e;
        --ok: #34d399; --warn: #f87171; --neutral: #9ca3af;
      }
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: system-ui, sans-serif; background: var(--bg); color: var(--fg);
           max-width: 900px; margin: 0 auto; padding: 1rem; line-height: 1.5; }
    h1 { margin-bottom: 0.2rem; }
    h2 { margin: 1.5rem 0 0.75rem; border-bottom: 2px solid var(--accent); padding-bottom: 0.25rem; }
    h3 { margin: 1.25rem 0 0.5rem; color: var(--fg); }
    h4 { margin: 0.75rem 0 0.4rem; color: var(--muted); font-size: 0.95rem; }
    .meta { color: var(--muted); margin-bottom: 1rem; font-size: 0.9rem; }
    .summary { color: var(--fg); margin: 0.25rem 0 1rem; font-size: 0.98rem; }
    .window-tabs { display: flex; gap: 0.5rem; margin: 1rem 0; flex-wrap: wrap; }
    .tab-btn { border: 1px solid var(--border); background: var(--card-bg); color: var(--fg);
               border-radius: 999px; padding: 0.35rem 0.8rem; cursor: pointer; }
    .tab-btn.active { border-color: var(--accent); color: var(--accent); font-weight: 600; }
    .window-panel { display: none; }
    .window-panel.active { display: block; }
    .cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 1rem; margin: 1rem 0; }
    .card { background: var(--card-bg); border: 1px solid var(--border); border-radius: 8px; padding: 1rem; }
    .card .label { font-size: 0.8rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.05em; }
    .card .value { font-size: 1.4rem; font-weight: bold; margin-top: 0.2rem; }
    .delta { display: block; margin-top: 0.35rem; font-size: 0.78rem; color: var(--neutral); }
    .delta.up { color: var(--ok); }
    .delta.down { color: var(--warn); }
    .stacked { display: flex; width: 100%; height: 14px; background: var(--bar-bg); border-radius: 999px; overflow: hidden; margin-top: 0.35rem; }
    .seg { display: block; height: 100%; }
    .seg-visitors { background: #2563eb; }
    .seg-scans { background: #f59e0b; }
    .seg-bots { background: #dc2626; }
    .scan-list { margin: 0.3rem 0 0.7rem 1.1rem; }
    .scan-list li { margin: 0.2rem 0; }
    table { width: 100%; border-collapse: collapse; margin: 0.5rem 0; }
    th, td { text-align: left; padding: 0.4rem 0.75rem; border-bottom: 1px solid var(--border); }
    th { font-size: 0.8rem; color: var(--muted); text-transform: uppercase; }
    td.num { text-align: right; font-variant-numeric: tabular-nums; }
    .split-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 0.8rem; }
    .chart-wrap { border: 1px solid var(--border); border-radius: 8px; padding: 0.4rem 0.4rem 0.2rem; background: var(--card-bg); }
    .chart-svg { width: 100%; height: 180px; display: block; }
    .chart-svg .area { fill: rgba(37, 99, 235, 0.16); }
    .chart-svg .line { fill: none; stroke: var(--accent); stroke-width: 2; }
    .chart-y-axis { display: flex; justify-content: space-between; font-size: 0.78rem; color: var(--muted); padding: 0 0.2rem 0.2rem; }
    .chart-axis { display: flex; justify-content: space-between; font-size: 0.78rem; color: var(--muted); padding: 0 0.2rem 0.2rem; }
    .row-current td { background: rgba(37, 99, 235, 0.10); }
    footer { margin-top: 2rem; padding-top: 1rem; border-top: 1px solid var(--border);
             font-size: 0.8rem; color: var(--muted); }
    </style>
    </head>
    <body>
    ```)

  (buffer/push-string buf
    (string/format "<h1>Log Analysis Report</h1>\n<p class=\"meta\">Generated: %s</p>\n<p class=\"summary\">%s: <strong>%d</strong> requests, <strong>~%d</strong> unique visitors estimate, <strong>%.1f%%</strong> visitor-classified traffic.</p>\n<p class=\"meta\">\"Visitors\" means requests not classified as bots or vulnerability scans; this is heuristic, not verified-human identity.</p>\n"
      (escape-html generated-at)
      (escape-html summary-label)
      (summary-agg :requests)
      summary-uniq
      summary-human))

  (buffer/push-string buf
    "<div class=\"window-tabs\" role=\"tablist\" aria-label=\"Time window\">
<button class=\"tab-btn active\" data-window=\"30d\" type=\"button\">30d</button>
<button class=\"tab-btn\" data-window=\"7d\" type=\"button\">7d</button>
<button class=\"tab-btn\" data-window=\"all\" type=\"button\">All time</button>
</div>\n")

  (render-window-panel buf "30d" "Last 30 Days" agg-30 agg-prev-30 uniq-30 uniq-prev-30 "30d" true true)
  (render-window-panel buf "7d" "Last 7 Days" agg-7 agg-prev-7 uniq-7 uniq-prev-7 "7d" false true)
  (render-window-panel buf "all" "All Time" agg-all @{:requests 0 :visitors 0} uniq-all nil nil false false)

  (when (or (not (empty? (agg-all :referrers)))
            (not (empty? (agg-all :referrers-visitors))))
    (def sorted-refs-all (top-pairs (agg-all :referrers) 20))
    (def sorted-refs-visitors (top-pairs (agg-all :referrers-visitors) 20))
    (buffer/push-string buf "<h2>Referrer Categories</h2>\n<div class=\"split-grid\">\n")
    (buffer/push-string buf "<div><h4>All Traffic</h4>\n<table>\n<tr><th>Category</th><th>Requests</th></tr>\n")
    (if (empty? sorted-refs-all)
      (buffer/push-string buf "<tr><td colspan=\"2\" class=\"meta\">No referrer data available.</td></tr>\n")
      (each [cat count] sorted-refs-all
        (buffer/push-string buf
          (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" (escape-html cat) count))))
    (buffer/push-string buf "</table></div>\n")

    (buffer/push-string buf "<div><h4>Visitor-Classified</h4>\n<table>\n<tr><th>Category</th><th>Requests</th></tr>\n")
    (if (empty? sorted-refs-visitors)
      (buffer/push-string buf
        (if (agg-all :has-referrer-by-type)
          "<tr><td colspan=\"2\" class=\"meta\">No visitor-classified referrer categories.</td></tr>\n"
          "<tr><td colspan=\"2\" class=\"meta\">Visitor referrer split unavailable in this dataset.</td></tr>\n"))
      (each [cat count] sorted-refs-visitors
        (buffer/push-string buf
          (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" (escape-html cat) count))))
    (buffer/push-string buf "</table></div>\n</div>\n"))

  (buffer/push-string buf "<h2>Visitors (All Time)</h2>\n<p class=\"meta\">The sections below exclude bots and vulnerability scans. Exact monthly uniques are available while raw logs are retained; long-term totals rely on HyperLogLog estimates.</p>\n")
  (buffer/push-string buf "<h3>Monthly Summary</h3>\n<table>\n")
  (buffer/push-string buf "<tr><th>Month</th><th>All Requests</th><th>Unique Visitors</th><th>HLL Estimate</th></tr>\n")
  (each m monthly-json
    (def month (or (m :month) "?"))
    (def row-class (if (= month current-month) " class=\"row-current\"" ""))
    (buffer/push-string buf
      (string/format "<tr%s><td>%s</td><td class=\"num\">%d</td><td class=\"num\">%d</td><td class=\"num\">%d</td></tr>\n"
        row-class
        (escape-html month)
        (or (m :requests) 0)
        (or (m :unique_visitors_exact) 0)
        (or (m :unique_visitors_estimate) 0))))
  (buffer/push-string buf "</table>\n")

  (when (not (empty? (agg-all :browsers)))
    (def sorted-browsers (top-pairs (agg-all :browsers) 20))
    (buffer/push-string buf "<h3>Browsers</h3>\n<table>\n")
    (buffer/push-string buf "<tr><th>Browser</th><th>Requests</th></tr>\n")
    (each [browser count] sorted-browsers
      (buffer/push-string buf
        (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" (escape-html browser) count)))
    (buffer/push-string buf "</table>\n"))

  (when (not (empty? (agg-all :platforms)))
    (def sorted-platforms (top-pairs (agg-all :platforms) 20))
    (buffer/push-string buf "<h3>Platforms</h3>\n<table>\n")
    (buffer/push-string buf "<tr><th>Platform</th><th>Requests</th></tr>\n")
    (each [platform count] sorted-platforms
      (buffer/push-string buf
        (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" (escape-html platform) count)))
    (buffer/push-string buf "</table>\n"))

  (buffer/push-string buf
    "<script>
(() => {
  const buttons = Array.from(document.querySelectorAll('.tab-btn'));
  const panels = Array.from(document.querySelectorAll('.window-panel'));
  const show = (id) => {
    buttons.forEach((b) => b.classList.toggle('active', b.dataset.window === id));
    panels.forEach((p) => p.classList.toggle('active', p.dataset.window === id));
  };
  buttons.forEach((b) => b.addEventListener('click', () => show(b.dataset.window)));
  show('30d');
})();
</script>
<footer>Generated by log-analyzer</footer>\n</body>\n</html>")
  (string buf))

(defn generate-report-from-json-dir
  "Load persisted JSON from json-dir and write report HTML to report-path."
  [json-dir report-path]
  (def data (load-report-data json-dir))
  (def html
    (build-html-report
      (data :monthly-json)
      (data :hourly-json)
      (data :total-requests)
      (data :total-uniques)
      (data :generated-at)
      (data :log-files)))
  (write-file report-path html)
  @{:report-path report-path
    :month-count (length (data :monthly-json))
    :generated-at (data :generated-at)})
