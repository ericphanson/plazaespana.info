# Parse Apache access logs and output JSON with HLL unique visitor tracking
#
# Monthly HLL buckets are stored as base64 so they can be merged later
# after log rotation discards the original files.
#
# Hourly stats use simple hash sets (faster) - we just store the count,
# not the full HLL, since we only need HLL at monthly granularity for merging.
#
# All timestamps are converted to UTC before bucketing.

(import ./hll)
(import ./json)

(def apache-log-peg
  "PEG grammar for Apache Combined Log Format"
  ~{:main (* :ip :s "-" :s :user :s "[" :datetime "]" :s
            "\"" :request "\"" :s :status :s :bytes :s
            (? (* "\"" :referrer "\"" :s "\"" :user-agent "\"")))
    :ip (<- (some (+ :d (range "af") (range "AF") (set ".:"))))  # IPv4 or IPv6
    :user (+ "-" (some (if-not :s 1)))  # Don't capture user field
    :datetime (* :day "/" :month "/" :year ":" :hour ":" :min ":" :sec :s :tz)
    :day (<- (between 1 2 :d))
    :month (<- (+ "Jan" "Feb" "Mar" "Apr" "May" "Jun"
                  "Jul" "Aug" "Sep" "Oct" "Nov" "Dec"))
    :year (<- (repeat 4 :d))
    :hour (<- (repeat 2 :d))
    :min (repeat 2 :d)
    :sec (repeat 2 :d)
    :tz (<- (* (set "+-") (repeat 4 :d)))
    :request (* (<- (to (+ " HTTP" "\""))) (? (* " HTTP/" (some (+ :d ".")))))
    :status (<- (some :d))
    :bytes (<- (+ "-" (some :d)))
    :referrer (<- (to "\""))
    :user-agent (<- (to "\""))
    :s (some (set " \t"))})

(def month-nums
  "Map month names to numbers"
  {"Jan" "01" "Feb" "02" "Mar" "03" "Apr" "04"
   "May" "05" "Jun" "06" "Jul" "07" "Aug" "08"
   "Sep" "09" "Oct" "10" "Nov" "11" "Dec" "12"})

# ============================================================================
# Timezone conversion
# ============================================================================

(def- days-in-month
  "Days per month (non-leap and leap year)"
  {1 31 2 28 3 31 4 30 5 31 6 30
   7 31 8 31 9 30 10 31 11 30 12 31})

(defn- leap-year? [y]
  (and (zero? (% y 4))
       (or (not (zero? (% y 100)))
           (zero? (% y 400)))))

(defn- month-days [y m]
  (if (and (= m 2) (leap-year? y))
    29
    (get days-in-month m)))

(defn- to-utc
  "Convert year/month-num/day/hour with tz offset string to UTC.
   Returns [year month-num day hour]."
  [year month-num day hour tz]
  (def sign (if (= (get tz 0) (chr "+")) 1 -1))
  (def tz-hours (scan-number (string/slice tz 1 3)))
  (def tz-mins (scan-number (string/slice tz 3 5)))
  (def offset-minutes (* sign (+ (* tz-hours 60) tz-mins)))
  # Subtract offset to get UTC (e.g. +0100 means local is 1h ahead, so UTC = local - 1)
  (var utc-hour (- hour (math/floor (/ offset-minutes 60))))
  (var utc-day day)
  (var utc-month month-num)
  (var utc-year year)
  # Handle hour underflow/overflow
  (when (< utc-hour 0)
    (+= utc-hour 24)
    (-- utc-day))
  (when (>= utc-hour 24)
    (-= utc-hour 24)
    (++ utc-day))
  # Handle day underflow
  (when (<= utc-day 0)
    (-- utc-month)
    (when (<= utc-month 0)
      (set utc-month 12)
      (-- utc-year))
    (set utc-day (month-days utc-year utc-month)))
  # Handle day overflow
  (when (> utc-day (month-days utc-year utc-month))
    (set utc-day 1)
    (++ utc-month)
    (when (> utc-month 12)
      (set utc-month 1)
      (++ utc-year)))
  [utc-year utc-month utc-day utc-hour])

# ============================================================================
# Parsing
# ============================================================================

(defn parse-log-line
  "Parse a single Apache log line and return parsed data or nil.
   Returns table with :ip :hour :month :request :status :bytes :referrer :user-agent
   Timestamps are converted to UTC."
  [line]
  (when-let [match (peg/match apache-log-peg line)]
    (let [[ip day month year hour tz request status bytes referrer user-agent] match
          month-num-str (get month-nums month)
          month-num (scan-number month-num-str)
          year-num (scan-number year)
          hour-num (scan-number hour)
          day-num (scan-number day)
          [uy um ud uh] (to-utc year-num month-num day-num hour-num (or tz "+0000"))
          um-str (string/format "%02d" um)
          ud-str (string/format "%02d" ud)]
      @{:ip ip
        :hour (string/format "%04d-%s-%sT%02d:00:00Z" uy um-str ud-str uh)
        :month (string/format "%04d-%s" uy um-str)
        :request (or request "")
        :status (or status "0")
        :bytes (if (or (nil? bytes) (= bytes "-")) 0 (scan-number bytes))
        :referrer (or referrer "-")
        :user-agent (or user-agent "")})))

(defn extract-path
  "Extract path from request string like 'GET /path HTTP/1.1'"
  [request]
  (if-let [match (peg/match ~(* (some :S) :s (<- (some (if-not :s 1)))) request)]
    (first match)
    "/"))

# ============================================================================
# Classification functions
# ============================================================================

(defn is-bot?
  "Check if user-agent indicates a bot/crawler"
  [ua]
  (def ua-lower (string/ascii-lower ua))
  (or (string/find "bot" ua-lower)
      (string/find "crawler" ua-lower)
      (string/find "spider" ua-lower)
      (string/find "scraper" ua-lower)
      (string/find "curl" ua-lower)
      (string/find "wget" ua-lower)
      (string/find "python" ua-lower)
      (string/find "go-http-client" ua-lower)
      (string/find "gptbot" ua-lower)
      (string/find "claudebot" ua-lower)
      (string/find "bingbot" ua-lower)
      (string/find "googlebot" ua-lower)
      (string/find "facebookexternalhit" ua-lower)
      (string/find "twitterbot" ua-lower)))

(defn classify-referrer
  "Classify a referrer URL into a category.
   Returns one of: direct, search, social, internal, external"
  [referrer]
  (cond
    (or (= referrer "-") (= referrer ""))
    "direct"

    # Extract domain from URL
    (if-let [match (peg/match ~(* "http" (? "s") "://" (<- (to (+ "/" -1)))) referrer)]
      (let [domain (string/ascii-lower (first match))]
        (cond
          # Internal
          (or (string/find "plazaespana.info" domain)
              (string/find "plazaespana.info" domain))
          "internal"

          # Search engines
          (or (string/find "google." domain)
              (string/find "bing." domain)
              (string/find "duckduckgo." domain)
              (string/find "baidu." domain)
              (string/find "yandex." domain)
              (string/find "yahoo." domain)
              (string/find "ecosia." domain)
              (string/find "search." domain))
          "search"

          # Social
          (or (string/find "facebook." domain)
              (string/find "twitter." domain)
              (string/find "x.com" domain)
              (string/find "reddit." domain)
              (string/find "linkedin." domain)
              (string/find "t.co" domain)
              (string/find "instagram." domain)
              (string/find "mastodon." domain))
          "social"

          # Everything else
          "external"))

      # Couldn't parse URL
      "direct")))

(defn classify-browser
  "Classify user-agent into a browser family.
   Order matters: check more specific strings first."
  [ua]
  (def ua-lower (string/ascii-lower ua))
  (cond
    (string/find "edg" ua-lower) "Edge"
    (string/find "opr" ua-lower) "Opera"
    (string/find "opera" ua-lower) "Opera"
    (string/find "samsungbrowser" ua-lower) "Samsung Internet"
    (string/find "firefox" ua-lower) "Firefox"
    (string/find "chrome" ua-lower) "Chrome"
    (string/find "safari" ua-lower) "Safari"
    "other"))

(defn classify-platform
  "Classify user-agent into an OS/platform family."
  [ua]
  (cond
    (string/find "iPhone" ua) "iOS"
    (string/find "iPad" ua) "iOS"
    (string/find "Android" ua) "Android"
    (string/find "CrOS" ua) "ChromeOS"
    (string/find "Windows" ua) "Windows"
    (string/find "Macintosh" ua) "macOS"
    (string/find "Mac OS X" ua) "macOS"
    (string/find "Linux" ua) "Linux"
    "other"))

# ============================================================================
# Line deduplication
# ============================================================================

(defn- fnv1a-line
  "FNV-1a hash of a string, returning an integer for use as a set key."
  [s]
  (var hash (int/u64 0x811c9dc5))
  (each byte s
    (set hash (bxor hash (int/u64 byte)))
    (set hash (band (* hash (int/u64 0x01000193)) (int/u64 0xFFFFFFFF))))
  # Return as number for use in table keys
  (int/to-number hash))

# ============================================================================
# Processing
# ============================================================================

(defn process-log-file
  "Process a log file and accumulate stats.
   stats is a table with:
   - :hourly - table of hour -> {:ips :requests :bytes :status-codes :paths :bots :humans
                                  :referrer-categories :browsers :platforms}
   - :monthly - table of month -> {:hll :ips :requests}
   - :total-hll - HLL for all unique visitors
   - :seen-lines - hash set for line deduplication"
  [filepath stats]
  (eprint "Processing: " filepath)
  (with [f (file/open filepath :r)]
    (var line-count 0)
    (var parse-errors 0)
    (var dup-lines 0)
    (def seen (stats :seen-lines))
    (loop [line :iterate (file/read f :line)]
      (++ line-count)
      (when (zero? (% line-count 10000))
        (eprin "."))

      # Dedup: skip lines we've already seen
      (def line-hash (fnv1a-line line))
      (if (get seen line-hash)
        (++ dup-lines)
        (do
          (put seen line-hash true)
          (if-let [parsed (parse-log-line line)]
            (let [{:ip ip :hour hour :month month :request request
                   :status status :bytes bytes :referrer referrer
                   :user-agent ua} parsed
                  path (extract-path request)
                  bot (is-bot? ua)]

              # Update hourly stats
              (def hourly-entry
                (or (get (stats :hourly) hour)
                    (let [entry @{:ips @{}
                                  :requests 0
                                  :bytes 0
                                  :bots 0
                                  :humans 0
                                  :status-codes @{}
                                  :paths @{}
                                  :referrer-categories @{}
                                  :browsers @{}
                                  :platforms @{}}]
                      (put (stats :hourly) hour entry)
                      entry)))

              (put (hourly-entry :ips) ip true)
              (put hourly-entry :requests (+ 1 (hourly-entry :requests)))
              (put hourly-entry :bytes (+ bytes (hourly-entry :bytes)))
              (if bot
                (put hourly-entry :bots (+ 1 (hourly-entry :bots)))
                (put hourly-entry :humans (+ 1 (hourly-entry :humans))))

              # Track status codes
              (def status-counts (hourly-entry :status-codes))
              (put status-counts status (+ 1 (get status-counts status 0)))

              # Track paths
              (def path-counts (hourly-entry :paths))
              (put path-counts path (+ 1 (get path-counts path 0)))

              # Track referrer categories
              (def ref-cats (hourly-entry :referrer-categories))
              (def ref-cat (classify-referrer referrer))
              (put ref-cats ref-cat (+ 1 (get ref-cats ref-cat 0)))

              # Track browser and platform (non-bots only)
              (when (not bot)
                (def browsers (hourly-entry :browsers))
                (def browser (classify-browser ua))
                (put browsers browser (+ 1 (get browsers browser 0)))

                (def platforms (hourly-entry :platforms))
                (def platform (classify-platform ua))
                (put platforms platform (+ 1 (get platforms platform 0))))

              # Update monthly stats (HLL + exact hash set)
              (def monthly-entry
                (or (get (stats :monthly) month)
                    (let [entry @{:hll (hll/new) :ips @{} :requests 0}]
                      (put (stats :monthly) month entry)
                      entry)))

              (hll/add (monthly-entry :hll) ip)
              (put (monthly-entry :ips) ip true)
              (put monthly-entry :requests (+ 1 (monthly-entry :requests)))

              # Update total HLL
              (hll/add (stats :total-hll) ip))

            # Parse error
            (++ parse-errors)))))

    (eprint "\n")
    (when (> dup-lines 0)
      (eprintf "  Deduplicated: %d duplicate lines skipped\n" dup-lines))
    (when (> parse-errors 0)
      (eprintf "  Warning: %d lines could not be parsed\n" parse-errors))))

(defn analyze-logs
  "Analyze all access log files in directory"
  [log-dir]
  (def stats @{:hourly @{}
               :monthly @{}
               :total-hll (hll/new)
               :seen-lines @{}})

  # Find all access_log* files
  (def log-files
    (sort (filter |(string/has-prefix? "access_log" $)
                  (os/dir log-dir))))

  (when (empty? log-files)
    (eprint "No access_log files found in " log-dir)
    (os/exit 1))

  (eprintf "Found %d log files\n" (length log-files))

  # Process each log file
  (each filename log-files
    (def filepath (string log-dir "/" filename))
    (process-log-file filepath stats))

  # Report dedup stats
  (eprintf "\nTotal unique lines processed: %d\n" (length (stats :seen-lines)))

  stats)

# ============================================================================
# Output formatting
# ============================================================================

(defn top-n
  "Get top N items from a frequency table"
  [freq-table n]
  (def items (pairs freq-table))
  (def sorted (sort-by |(- (get $ 1)) items))
  (take n sorted))

(defn format-top-paths
  "Convert top-n paths to array of objects, with 'other' bucket for the rest"
  [path-counts n]
  (def items (pairs path-counts))
  (def sorted (sort-by |(- (get $ 1)) items))
  (def top (take n sorted))
  (def top-result
    (map (fn [[path count]]
           @{:path path :requests count})
         top))
  # Add "other" bucket for remaining paths
  (when (> (length sorted) n)
    (def other-count
      (sum (map |(get $ 1) (drop n sorted))))
    (when (> other-count 0)
      (array/push top-result @{:path "other" :requests other-count})))
  top-result)

(defn build-hourly-json
  "Build hourly array for JSON output"
  [hourly-stats]
  (def hours (sort (keys hourly-stats)))
  (seq [hour :in hours
        :let [entry (get hourly-stats hour)]
        # Skip malformed entries (from parse failures)
        :when (and (string/find "-" hour)
                   (> (length hour) 10))]
    @{:hour hour
      :requests (entry :requests)
      :bytes_sent (entry :bytes)
      :unique_ips (length (entry :ips))
      :bots (entry :bots)
      :humans (entry :humans)
      :status_codes (entry :status-codes)
      :top_paths (format-top-paths (entry :paths) 20)
      :referrer_categories (entry :referrer-categories)
      :browsers (entry :browsers)
      :platforms (entry :platforms)}))

(defn build-monthly-json
  "Build monthly summary for JSON output.
   Includes both exact and HLL-estimated unique visitor counts."
  [monthly-stats]
  (def months (sort (keys monthly-stats)))
  (map (fn [month]
         (def entry (get monthly-stats month))
         @{:month month
           :requests (entry :requests)
           :unique_visitors_exact (length (entry :ips))
           :unique_visitors_estimate (math/round (hll/count-estimate (entry :hll)))
           :unique_visitors_hll (hll/to-base64 (entry :hll))})
       months))

# ============================================================================
# Output modes
# ============================================================================

(defn build-full-json
  "Build the complete JSON output document"
  [stats log-files]
  (def hourly-json (build-hourly-json (stats :hourly)))
  (def monthly-json (build-monthly-json (stats :monthly)))

  @{:generated_at (os/strftime "%Y-%m-%dT%H:%M:%SZ" (os/time))
    :log_files_processed log-files
    :total_requests (sum (map |(get $ :requests) (values (stats :monthly))))
    :total_unique_visitors_estimate (math/round (hll/count-estimate (stats :total-hll)))
    :total_unique_visitors_hll (hll/to-base64 (stats :total-hll))
    :hourly hourly-json
    :monthly monthly-json})

(defn output-json
  "Output analytics data as JSON to stdout"
  [stats log-files]
  (print (json/encode-pretty (build-full-json stats log-files))))

(defn- write-file
  "Write string to file atomically (write to temp, then rename)"
  [path content]
  (def tmp (string path ".tmp"))
  (spit tmp content)
  (os/rename tmp path))

(defn output-split
  "Output analytics as per-month JSON files + lifetime.json in out-dir"
  [stats log-files out-dir]
  (os/mkdir out-dir)

  (def monthly-json (build-monthly-json (stats :monthly)))
  (def hourly-json (build-hourly-json (stats :hourly)))
  (def generated-at (os/strftime "%Y-%m-%dT%H:%M:%SZ" (os/time)))

  # Write per-month files
  (each month-entry monthly-json
    (def month (month-entry :month))
    # Filter hourly entries for this month
    (def month-hourly
      (filter |(string/has-prefix? month ($ :hour)) hourly-json))
    (def month-doc
      @{:generated_at generated-at
        :month month
        :summary month-entry
        :hourly month-hourly})
    (def path (string out-dir "/" month ".json"))
    (write-file path (json/encode-pretty month-doc))
    (eprintf "Wrote %s\n" path))

  # Write lifetime.json with merged HLLs
  (def lifetime-doc
    @{:generated_at generated-at
      :log_files_processed log-files
      :total_requests (sum (map |(get $ :requests) (values (stats :monthly))))
      :total_unique_visitors_estimate (math/round (hll/count-estimate (stats :total-hll)))
      :total_unique_visitors_hll (hll/to-base64 (stats :total-hll))
      :months monthly-json})
  (def lifetime-path (string out-dir "/lifetime.json"))
  (write-file lifetime-path (json/encode-pretty lifetime-doc))
  (eprintf "Wrote %s\n" lifetime-path))

# ============================================================================
# HTML report
# ============================================================================

(defn build-html-report
  "Generate a self-contained static HTML report (no JavaScript)"
  [stats log-files]
  (def monthly-json (build-monthly-json (stats :monthly)))
  (def hourly-json (build-hourly-json (stats :hourly)))
  (def generated-at (os/strftime "%Y-%m-%dT%H:%M:%SZ" (os/time)))
  (def total-requests (sum (map |(get $ :requests) (values (stats :monthly)))))
  (def total-uniques (math/round (hll/count-estimate (stats :total-hll))))

  # Aggregate top paths across all hours
  (def all-paths @{})
  (each entry hourly-json
    (each p (entry :top_paths)
      (def path (p :path))
      (when (not= path "other")
        (put all-paths path (+ (get all-paths path 0) (p :requests))))))
  (def top-paths-all (take 20 (sort-by |(- (get $ 1)) (pairs all-paths))))

  # Aggregate browsers/platforms/referrers across all hours
  (def all-browsers @{})
  (def all-platforms @{})
  (def all-referrers @{})
  (each entry hourly-json
    (eachp [k v] (entry :browsers)
      (put all-browsers k (+ (get all-browsers k 0) v)))
    (eachp [k v] (entry :platforms)
      (put all-platforms k (+ (get all-platforms k 0) v)))
    (eachp [k v] (entry :referrer_categories)
      (put all-referrers k (+ (get all-referrers k 0) v))))

  # Find max daily requests for bar chart scaling
  (def daily-requests @{})
  (each entry hourly-json
    (def day (string/slice (entry :hour) 0 10))
    (put daily-requests day (+ (get daily-requests day 0) (entry :requests))))
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

  # Summary cards
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

  # Monthly summary table
  (buffer/push-string buf "<h2>Monthly Summary</h2>\n<table>\n")
  (buffer/push-string buf "<tr><th>Month</th><th>Requests</th><th>Unique Visitors</th><th>HLL Estimate</th></tr>\n")
  (each m monthly-json
    (buffer/push-string buf
      (string/format "<tr><td>%s</td><td class=\"num\">%d</td><td class=\"num\">%d</td><td class=\"num\">%d</td></tr>\n"
        (m :month) (m :requests) (m :unique_visitors_exact) (m :unique_visitors_estimate))))
  (buffer/push-string buf "</table>\n")

  # Daily request bar chart
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

  # Top paths
  (buffer/push-string buf "<h2>Top Paths</h2>\n<table>\n")
  (buffer/push-string buf "<tr><th>Path</th><th>Requests</th></tr>\n")
  (each [path count] top-paths-all
    (buffer/push-string buf
      (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" path count)))
  (buffer/push-string buf "</table>\n")

  # Browser breakdown
  (when (not (empty? all-browsers))
    (def sorted-browsers (sort-by |(- (get $ 1)) (pairs all-browsers)))
    (buffer/push-string buf "<h2>Browsers</h2>\n<table>\n")
    (buffer/push-string buf "<tr><th>Browser</th><th>Requests</th></tr>\n")
    (each [browser count] sorted-browsers
      (buffer/push-string buf
        (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" browser count)))
    (buffer/push-string buf "</table>\n"))

  # Platform breakdown
  (when (not (empty? all-platforms))
    (def sorted-platforms (sort-by |(- (get $ 1)) (pairs all-platforms)))
    (buffer/push-string buf "<h2>Platforms</h2>\n<table>\n")
    (buffer/push-string buf "<tr><th>Platform</th><th>Requests</th></tr>\n")
    (each [platform count] sorted-platforms
      (buffer/push-string buf
        (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" platform count)))
    (buffer/push-string buf "</table>\n"))

  # Referrer categories
  (when (not (empty? all-referrers))
    (def sorted-refs (sort-by |(- (get $ 1)) (pairs all-referrers)))
    (buffer/push-string buf "<h2>Referrer Categories</h2>\n<table>\n")
    (buffer/push-string buf "<tr><th>Category</th><th>Requests</th></tr>\n")
    (each [cat count] sorted-refs
      (buffer/push-string buf
        (string/format "<tr><td>%s</td><td class=\"num\">%d</td></tr>\n" cat count)))
    (buffer/push-string buf "</table>\n"))

  (buffer/push-string buf "<footer>Generated by log-analyzer</footer>\n</body>\n</html>")
  (string buf))

# ============================================================================
# CLI
# ============================================================================

(defn- parse-args
  "Parse CLI arguments. Returns {:log-dir :out-dir}"
  [args]
  (def result @{:log-dir nil :out-dir nil})
  (var i 1)  # Skip executable name at index 0
  (while (< i (length args))
    (def arg (get args i))
    (cond
      (or (= arg "--out-dir") (= arg "-o"))
      (do (++ i)
          (put result :out-dir (get args i)))

      # Positional: log directory
      (nil? (result :log-dir))
      (put result :log-dir arg))

    (++ i))

  # Default log directory
  (when (nil? (result :log-dir))
    (put result :log-dir "/Users/eph/plazaespana.info/awstats-data/logs"))

  result)

(defn main [& args]
  (def opts (parse-args args))
  (def log-dir (opts :log-dir))
  (def out-dir (opts :out-dir))

  (eprintf "Analyzing Apache access logs in: %s\n\n" log-dir)

  # Get list of log files before analysis
  (def log-files
    (sort (filter |(string/has-prefix? "access_log" $)
                  (os/dir log-dir))))

  # Analyze logs
  (def stats (analyze-logs log-dir))

  # Output
  (if out-dir
    (do
      (output-split stats log-files out-dir)
      # Also write HTML report to out-dir
      (def html (build-html-report stats log-files))
      (write-file (string out-dir "/report.html") html)
      (eprintf "Wrote %s/report.html\n" out-dir))
    (output-json stats log-files)))
