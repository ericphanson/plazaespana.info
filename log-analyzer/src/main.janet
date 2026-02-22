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
(import ./report)
(import ./rebuild)

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

(defn- log-file-candidate?
  "Accept plain-text access log files; wrapper handles .gz expansion."
  [name]
  (and (string/has-prefix? "access_log" name)
       (not (string/has-suffix? ".gz" name))))

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
  "Extract path from request string like 'GET /path HTTP/1.1'.
   Drops query strings to avoid persisting tracking parameters."
  [request]
  (def path
    (if-let [match (peg/match ~(* (some :S) :s (<- (some (if-not :s 1)))) request)]
      (first match)
      "/"))
  (if-let [qmark (string/find "?" path)]
    (string/slice path 0 qmark)
    path))

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

(defn is-scan?
  "Check if a request looks like a vulnerability scan.
   Uses two strategies:
   1. Path-based: known scan paths (WordPress, PHP, config files, etc.)
   2. Status-based: 404s to paths that aren't plausible missing resources

   A static site serves a small set of known paths. A 404 to /wp-login.php
   is a scan. A 404 to /favicon.ico is a browser looking for an icon we
   don't serve. We distinguish between the two."
  [path status]
  (def path-lower (string/ascii-lower path))
  (or
    # Strategy 1: known scan paths (always a scan regardless of status)
    (string/has-prefix? "/wp-" path-lower)
    (string/has-prefix? "/wordpress" path-lower)
    (string/find "xmlrpc.php" path-lower)
    # PHP files (this is a static site, no PHP exists)
    (string/has-suffix? ".php" path-lower)
    # Config/sensitive file probes
    (string/has-prefix? "/.env" path-lower)
    (string/has-prefix? "/.git" path-lower)
    (string/has-prefix? "/.aws" path-lower)
    (string/has-prefix? "/.docker" path-lower)
    (string/has-prefix? "/.ssh" path-lower)
    (string/has-prefix? "/.svn" path-lower)
    (string/has-prefix? "/.htpasswd" path-lower)
    (string/has-prefix? "/.DS_Store" path-lower)
    # Admin panels
    (string/has-prefix? "/admin" path-lower)
    (string/has-prefix? "/phpmyadmin" path-lower)
    (string/has-prefix? "/cgi-bin" path-lower)
    # Common exploit paths
    (string/find "wlwmanifest.xml" path-lower)
    (string/find "/eval" path-lower)
    (string/find "/shell" path-lower)
    (string/find "/config." path-lower)
    (string/find "alfacgiapi" path-lower)
    (string/find "fileupload" path-lower)
    (string/has-prefix? "/login" path-lower)
    (string/has-prefix? "/signin" path-lower)
    (string/has-prefix? "/modules/" path-lower)

    # Strategy 2: 404s that aren't plausible browser requests
    # Browsers legitimately 404 on favicon, apple-touch-icon, .well-known, etc.
    # Everything else that 404s on a static site is a scan.
    (and (= status "404")
         (not (or (= path-lower "/favicon.ico")
                  (string/has-prefix? "/apple-touch-icon" path-lower)
                  (and (string/has-prefix? "/.well-known/" path-lower)
                       (not (string/has-suffix? ".php" path-lower)))
                  (string/has-prefix? "/assets/" path-lower))))))

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
   - :hourly - table of hour -> {:ips :requests :bytes :status-codes :paths :bots :visitors
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
                  bot (is-bot? ua)
                  scan (and (not bot) (is-scan? path status))]

              # Update hourly stats
              (def hourly-entry
                (or (get (stats :hourly) hour)
                    (let [entry @{:ips @{}
                                  :requests 0
                                  :bytes 0
                                  :bots 0
                                  :scans 0
                                  :visitors 0
                                  :status-codes @{}
                                  :status-by-type @{:bots @{} :scans @{} :visitors @{}}
                                  :paths @{}
                                  :referrer-categories @{}
                                  :browsers @{}
                                  :platforms @{}}]
                      (put (stats :hourly) hour entry)
                      entry)))

              (put hourly-entry :requests (+ 1 (hourly-entry :requests)))
              (put hourly-entry :bytes (+ bytes (hourly-entry :bytes)))
              (cond
                bot (put hourly-entry :bots (+ 1 (hourly-entry :bots)))
                scan (put hourly-entry :scans (+ 1 (hourly-entry :scans)))
                (put hourly-entry :visitors (+ 1 (hourly-entry :visitors))))

              # Track status codes (overall + per traffic type)
              (def status-counts (hourly-entry :status-codes))
              (put status-counts status (+ 1 (get status-counts status 0)))
              (def type-key (cond bot :bots scan :scans :visitors))
              (def type-status (get (hourly-entry :status-by-type) type-key))
              (put type-status status (+ 1 (get type-status status 0)))

              # Track paths
              (def path-counts (hourly-entry :paths))
              (put path-counts path (+ 1 (get path-counts path 0)))

              # Track referrer categories
              (def ref-cats (hourly-entry :referrer-categories))
              (def ref-cat (classify-referrer referrer))
              (put ref-cats ref-cat (+ 1 (get ref-cats ref-cat 0)))

              # Track unique visitors, browsers, platforms (visitors only)
              (when (not (or bot scan))
                (put (hourly-entry :ips) ip true)

                (def browsers (hourly-entry :browsers))
                (def browser (classify-browser ua))
                (put browsers browser (+ 1 (get browsers browser 0)))

                (def platforms (hourly-entry :platforms))
                (def platform (classify-platform ua))
                (put platforms platform (+ 1 (get platforms platform 0)))

                # Update monthly uniques (HLL + exact hash set)
                (def monthly-entry
                  (or (get (stats :monthly) month)
                      (let [entry @{:hll (hll/new) :ips @{} :requests 0}]
                        (put (stats :monthly) month entry)
                        entry)))

                (hll/add (monthly-entry :hll) ip)
                (put (monthly-entry :ips) ip true)

                # Update total HLL
                (hll/add (stats :total-hll) ip))

              # Update monthly request count (all traffic)
              (def monthly-entry
                (or (get (stats :monthly) month)
                    (let [entry @{:hll (hll/new) :ips @{} :requests 0}]
                      (put (stats :monthly) month entry)
                      entry)))
              (put monthly-entry :requests (+ 1 (monthly-entry :requests))))

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
    (sort (filter log-file-candidate? (os/dir log-dir))))

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
      :scans (entry :scans)
      :visitors (entry :visitors)
      :status_codes (entry :status-codes)
      :status_by_type (entry :status-by-type)
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
# CLI
# ============================================================================

(defn- parse-args
  "Parse CLI arguments.
   Modes:
   - analyze (default): read logs and emit JSON
   - report: read persisted JSON files and emit report.html
   - rebuild: rebuild lifetime.json from persisted month shards"
  [args]
  (def result @{:mode "analyze"
                :log-dir nil
                :out-dir nil
                :json-dir nil
                :report-path nil
                :lifetime-source nil})
  (var i 1)  # Skip executable name at index 0
  (while (< i (length args))
    (def arg (get args i))
    (cond
      (= arg "--mode")
      (do
        (++ i)
        (put result :mode (or (get args i) "")))

      (or (= arg "--out-dir") (= arg "-o"))
      (do
        (++ i)
        (put result :out-dir (or (get args i) "")))

      (= arg "--json-dir")
      (do
        (++ i)
        (put result :json-dir (or (get args i) "")))

      (= arg "--report-path")
      (do
        (++ i)
        (put result :report-path (or (get args i) "")))

      (= arg "--lifetime-source")
      (do
        (++ i)
        (put result :lifetime-source (or (get args i) "")))

      (string/has-prefix? "-" arg)
      (do
        (eprintf "Unknown option: %s\n" arg)
        (os/exit 1))

      # Positional arg
      true
      (if (or (= (result :mode) "report") (= (result :mode) "rebuild"))
        (if (nil? (result :json-dir))
          (put result :json-dir arg)
          (do
            (eprintf "Unexpected extra positional argument: %s\n" arg)
            (os/exit 1)))
        (if (nil? (result :log-dir))
          (put result :log-dir arg)
          (do
            (eprintf "Unexpected extra positional argument: %s\n" arg)
            (os/exit 1)))))

    (++ i))

  (def mode (result :mode))
  (when (and (not= mode "analyze") (not= mode "report") (not= mode "rebuild"))
    (eprintf "Invalid --mode: %s (expected analyze, report, or rebuild)\n" mode)
    (os/exit 1))

  (if (or (= mode "report") (= mode "rebuild"))
    (do
      (when (nil? (result :json-dir))
        (if (not (nil? (result :out-dir)))
          (put result :json-dir (result :out-dir))
          (put result :json-dir "/home/private/log-analyzer-data")))
      (when (and (= mode "report") (nil? (result :report-path)))
        (put result :report-path (string (result :json-dir) "/report.html"))))
    (when (nil? (result :log-dir))
      (put result :log-dir "/home/logs")))

  result)

(defn main [& args]
  (def opts (parse-args args))
  (def mode (opts :mode))

  (if (= mode "report")
    (let [json-dir (opts :json-dir)
          report-path (opts :report-path)
          result (report/generate-report-from-json-dir json-dir report-path)]
      (eprintf "Built report from JSON dir: %s\n" json-dir)
      (eprintf "Wrote %s\n" (result :report-path)))
    (if (= mode "rebuild")
      (let [json-dir (opts :json-dir)
            lifetime-source (opts :lifetime-source)
            result (rebuild/rebuild-lifetime-from-json-dir json-dir lifetime-source)]
        (eprintf "Rebuilt lifetime from JSON dir: %s\n" json-dir)
        (eprintf "Wrote %s\n" (result :lifetime-path)))
      (let [log-dir (opts :log-dir)
            out-dir (opts :out-dir)]
        (eprintf "Analyzing Apache access logs in: %s\n\n" log-dir)

        # Get list of log files before analysis
        (def log-files
          (sort (filter log-file-candidate? (os/dir log-dir))))

        # Analyze logs
        (def stats (analyze-logs log-dir))

        # Output
        (if out-dir
          (output-split stats log-files out-dir)
          (output-json stats log-files))))))
