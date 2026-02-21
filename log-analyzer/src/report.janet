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

(defn build-html-report
  "Generate self-contained HTML report from JSON-derived monthly/hourly docs."
  [monthly-json hourly-json total-requests total-uniques generated-at log-files]
  # Aggregate top paths across all hours
  (def all-paths @{})
  (each entry hourly-json
    (each p (get entry :top_paths @[])
      (def path (p :path))
      (when (and (string? path) (not= path "other"))
        (put all-paths path (+ (get all-paths path 0) (or (p :requests) 0))))))
  (def top-paths-all (take 20 (sort-by |(- (get $ 1)) (pairs all-paths))))

  # Aggregate browsers/platforms/referrers across all hours
  (def all-browsers @{})
  (def all-platforms @{})
  (def all-referrers @{})
  (each entry hourly-json
    (eachp [k v] (get entry :browsers @{})
      (put all-browsers k (+ (get all-browsers k 0) v)))
    (eachp [k v] (get entry :platforms @{})
      (put all-platforms k (+ (get all-platforms k 0) v)))
    (eachp [k v] (get entry :referrer_categories @{})
      (put all-referrers k (+ (get all-referrers k 0) v))))

  # Aggregate traffic type totals and status codes per type
  (var total-bots 0)
  (var total-scans 0)
  (var total-visitors 0)
  (def all-status @{})
  (def status-bots @{})
  (def status-scans @{})
  (def status-visitors @{})
  (each entry hourly-json
    (+= total-bots (or (entry :bots) 0))
    (+= total-scans (or (entry :scans) 0))
    (+= total-visitors (or (entry :visitors) 0))
    (eachp [k v] (get (get entry :status_by_type @{}) :bots @{})
      (put status-bots k (+ (get status-bots k 0) v)))
    (eachp [k v] (get (get entry :status_by_type @{}) :scans @{})
      (put status-scans k (+ (get status-scans k 0) v)))
    (eachp [k v] (get (get entry :status_by_type @{}) :visitors @{})
      (put status-visitors k (+ (get status-visitors k 0) v)))
    (eachp [k v] (get entry :status_codes @{})
      (put all-status k (+ (get all-status k 0) v))))

  # Find max daily requests for bar chart scaling
  (def daily-requests @{})
  (each entry hourly-json
    (when (string? (entry :hour))
      (def day (string/slice (entry :hour) 0 10))
      (put daily-requests day (+ (get daily-requests day 0) (or (entry :requests) 0)))))
  (def max-daily (max ;(values daily-requests) 1))

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
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #111; --fg: #e5e5e5; --muted: #999; --border: #333;
        --accent: #60a5fa; --bar-bg: #1f2937; --card-bg: #1a1a2e;
      }
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: system-ui, sans-serif; background: var(--bg); color: var(--fg);
           max-width: 900px; margin: 0 auto; padding: 1rem; line-height: 1.5; }
    h1 { margin-bottom: 0.25rem; }
    h2 { margin: 1.5rem 0 0.75rem; border-bottom: 2px solid var(--accent); padding-bottom: 0.25rem; }
    h3 { margin: 1.25rem 0 0.5rem; color: var(--fg); }
    .meta { color: var(--muted); margin-bottom: 1rem; font-size: 0.9rem; }
    .cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 1rem; margin: 1rem 0; }
    .card { background: var(--card-bg); border: 1px solid var(--border); border-radius: 8px; padding: 1rem; }
    .card .label { font-size: 0.8rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.05em; }
    .card .value { font-size: 1.5rem; font-weight: bold; margin-top: 0.25rem; }
    table { width: 100%; border-collapse: collapse; margin: 0.5rem 0; }
    th, td { text-align: left; padding: 0.4rem 0.75rem; border-bottom: 1px solid var(--border); }
    th { font-size: 0.8rem; color: var(--muted); text-transform: uppercase; }
    td.num { text-align: right; font-variant-numeric: tabular-nums; }
    .bar-container { background: var(--bar-bg); border-radius: 4px; height: 1.2rem; overflow: hidden; }
    .bar { background: var(--accent); height: 100%; border-radius: 4px; min-width: 2px; }
    footer { margin-top: 2rem; padding-top: 1rem; border-top: 1px solid var(--border);
             font-size: 0.8rem; color: var(--muted); }
    </style>
    </head>
    <body>
    ```)

  (buffer/push-string buf
    (string/format "<h1>Log Analysis Report</h1>\n<p class=\"meta\">Generated: %s</p>\n" generated-at))

  (buffer/push-string buf "<div class=\"cards\">\n")
  (buffer/push-string buf
    (string/format
      "<div class=\"card\"><div class=\"label\">Total Requests</div><div class=\"value\">%d</div></div>\n"
      total-requests))
  (buffer/push-string buf
    (string/format
      "<div class=\"card\"><div class=\"label\">Unique Visitors</div><div class=\"value\">%d</div></div>\n"
      total-uniques))
  (buffer/push-string buf
    (string/format
      "<div class=\"card\"><div class=\"label\">Months</div><div class=\"value\">%d</div></div>\n"
      (length monthly-json)))
  (buffer/push-string buf
    (string/format
      "<div class=\"card\"><div class=\"label\">Log Files</div><div class=\"value\">%d</div></div>\n"
      (length log-files)))
  (buffer/push-string buf "</div>\n")

  (defn- status-summary [tbl]
    (if (empty? tbl)
      "-"
      (string/join
        (map (fn [[s c]] (string/format "%s: %d" s c))
             (sort-by |(- (get $ 1)) (pairs tbl)))
        ", ")))

  (buffer/push-string buf "<h2>Traffic Breakdown</h2>\n<table>\n")
  (buffer/push-string buf "<tr><th>Type</th><th>Requests</th><th>%</th><th>Status Codes</th></tr>\n")
  (each [label count status-tbl]
        [["Visitors" total-visitors status-visitors]
         ["Scans" total-scans status-scans]
         ["Bots" total-bots status-bots]]
    (def pct (if (> total-requests 0) (/ (* 100 count) total-requests) 0))
    (buffer/push-string buf
      (string/format "<tr><td>%s</td><td class=\"num\">%d</td><td class=\"num\">%.1f%%</td><td>%s</td></tr>\n"
        label count pct (status-summary status-tbl))))
  (buffer/push-string buf
    (string/format "<tr><td><strong>Total</strong></td><td class=\"num\"><strong>%d</strong></td><td class=\"num\"><strong>100%%</strong></td><td>%s</td></tr>\n"
      total-requests (status-summary all-status)))
  (buffer/push-string buf "</table>\n")

  (def sorted-days (sort (keys daily-requests)))
  (buffer/push-string buf "<h2>Daily Requests</h2>\n<table>\n")
  (buffer/push-string buf "<tr><th>Date</th><th>Requests</th><th></th></tr>\n")
  (each day sorted-days
    (def count (get daily-requests day))
    (def pct (math/round (* 100 (/ count max-daily))))
    (buffer/push-string buf
      (string/format
        "<tr><td>%s</td><td class=\"num\">%d</td><td><div class=\"bar-container\"><div class=\"bar\" style=\"width:%d%%\"></div></div></td></tr>\n"
        day count pct)))
  (buffer/push-string buf "</table>\n")

  (buffer/push-string buf "<h2>Top Paths</h2>\n<table>\n")
  (buffer/push-string buf "<tr><th>Path</th><th>Requests</th></tr>\n")
  (each [path count] top-paths-all
    (buffer/push-string buf
      (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" path count)))
  (buffer/push-string buf "</table>\n")

  (when (not (empty? all-referrers))
    (def sorted-refs (sort-by |(- (get $ 1)) (pairs all-referrers)))
    (buffer/push-string buf "<h2>Referrer Categories</h2>\n<table>\n")
    (buffer/push-string buf "<tr><th>Category</th><th>Requests</th></tr>\n")
    (each [cat count] sorted-refs
      (buffer/push-string buf
        (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" cat count)))
    (buffer/push-string buf "</table>\n"))

  (buffer/push-string buf "<h2>Visitors</h2>\n<p class=\"meta\">The following sections exclude bots and vulnerability scans.</p>\n")
  (buffer/push-string buf "<h3>Monthly Summary</h3>\n<table>\n")
  (buffer/push-string buf "<tr><th>Month</th><th>All Requests</th><th>Unique Visitors</th><th>HLL Estimate</th></tr>\n")
  (each m monthly-json
    (buffer/push-string buf
      (string/format "<tr><td>%s</td><td class=\"num\">%d</td><td class=\"num\">%d</td><td class=\"num\">%d</td></tr>\n"
        (or (m :month) "?")
        (or (m :requests) 0)
        (or (m :unique_visitors_exact) 0)
        (or (m :unique_visitors_estimate) 0))))
  (buffer/push-string buf "</table>\n")

  (when (not (empty? all-browsers))
    (def sorted-browsers (sort-by |(- (get $ 1)) (pairs all-browsers)))
    (buffer/push-string buf "<h3>Browsers</h3>\n<table>\n")
    (buffer/push-string buf "<tr><th>Browser</th><th>Requests</th></tr>\n")
    (each [browser count] sorted-browsers
      (buffer/push-string buf
        (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" browser count)))
    (buffer/push-string buf "</table>\n"))

  (when (not (empty? all-platforms))
    (def sorted-platforms (sort-by |(- (get $ 1)) (pairs all-platforms)))
    (buffer/push-string buf "<h3>Platforms</h3>\n<table>\n")
    (buffer/push-string buf "<tr><th>Platform</th><th>Requests</th></tr>\n")
    (each [platform count] sorted-platforms
      (buffer/push-string buf
        (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" platform count)))
    (buffer/push-string buf "</table>\n"))

  (buffer/push-string buf "<footer>Generated by log-analyzer</footer>\n</body>\n</html>")
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
