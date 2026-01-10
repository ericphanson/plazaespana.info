# Parse Apache access logs and output JSON with HLL unique visitor tracking
#
# Output schema matches docs/plans/2026-01-10-replace-awstats.md
# Monthly HLL buckets are stored as base64 so they can be merged later
# after log rotation discards the original files.
#
# Hourly stats use simple hash sets (faster) - we just store the count,
# not the full HLL, since we only need HLL at monthly granularity for merging.

(import ./hll)
(import ./json)

(def apache-log-peg
  "PEG grammar for Apache Combined Log Format"
  ~{:main (* :ip :s "-" :s :user :s "[" :datetime "]" :s
            "\"" :request "\"" :s :status :s :bytes :s
            (? (* "\"" :referrer "\"" :s "\"" :user-agent "\"")))
    :ip (<- (some (+ :d (set ".:a-fA-F"))))  # IPv4 or IPv6
    :user (+ "-" (<- (some (if-not :s 1))))
    :datetime (* :day "/" :month "/" :year ":" :hour ":" :min ":" :sec :s :tz)
    :day (<- (between 1 2 :d))
    :month (<- (+ "Jan" "Feb" "Mar" "Apr" "May" "Jun"
                  "Jul" "Aug" "Sep" "Oct" "Nov" "Dec"))
    :year (<- (repeat 4 :d))
    :hour (<- (repeat 2 :d))
    :min (repeat 2 :d)
    :sec (repeat 2 :d)
    :tz (* (set "+-") (repeat 4 :d))
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

(defn parse-log-line
  "Parse a single Apache log line and return parsed data or nil.
   Returns table with :ip :hour :month :request :status :bytes :referrer :user-agent"
  [line]
  (when-let [match (peg/match apache-log-peg line)]
    (let [[ip day month year hour request status bytes referrer user-agent] match
          month-num (get month-nums month)
          # Pad day with leading zero if needed
          day-padded (if (= 1 (length day)) (string "0" day) day)]
      @{:ip ip
        :hour (string year "-" month-num "-" day-padded "T" hour ":00:00Z")
        :month (string year "-" month-num)
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

(defn process-log-file
  "Process a log file and accumulate stats.
   stats is a table with:
   - :hourly - table of hour -> {:ips (hash set) :requests :bytes :status-codes :paths :bots :humans}
   - :monthly - table of month -> {:hll :requests}
   - :total-hll - HLL for all unique visitors"
  [filepath stats]
  (eprint "Processing: " filepath)
  (with [f (file/open filepath :r)]
    (var line-count 0)
    (var parse-errors 0)
    (loop [line :iterate (file/read f :line)]
      (++ line-count)
      (when (zero? (% line-count 10000))
        (eprin "."))
      (if-let [parsed (parse-log-line line)]
        (let [{:ip ip :hour hour :month month :request request
               :status status :bytes bytes :user-agent ua} parsed
              path (extract-path request)
              bot (is-bot? ua)]

          # Update hourly stats (use hash set for unique IPs, not HLL)
          (def hourly-entry
            (or (get (stats :hourly) hour)
                (let [entry @{:ips @{}  # Hash set for unique IPs
                              :requests 0
                              :bytes 0
                              :bots 0
                              :humans 0
                              :status-codes @{}
                              :paths @{}}]
                  (put (stats :hourly) hour entry)
                  entry)))

          (put (hourly-entry :ips) ip true)  # Add to hash set
          (put hourly-entry :requests (+ 1 (hourly-entry :requests)))
          (put hourly-entry :bytes (+ bytes (hourly-entry :bytes)))
          (if bot
            (put hourly-entry :bots (+ 1 (hourly-entry :bots)))
            (put hourly-entry :humans (+ 1 (hourly-entry :humans))))

          # Track status codes
          (def status-counts (hourly-entry :status-codes))
          (put status-counts status (+ 1 (get status-counts status 0)))

          # Track paths (top N)
          (def path-counts (hourly-entry :paths))
          (put path-counts path (+ 1 (get path-counts path 0)))

          # Update monthly HLL (this is what we need for merging after log rotation)
          (def monthly-entry
            (or (get (stats :monthly) month)
                (let [entry @{:hll (hll/new) :requests 0}]
                  (put (stats :monthly) month entry)
                  entry)))

          (hll/add (monthly-entry :hll) ip)
          (put monthly-entry :requests (+ 1 (monthly-entry :requests)))

          # Update total HLL
          (hll/add (stats :total-hll) ip))

        # Parse error
        (++ parse-errors)))

    (eprint "\n")
    (when (> parse-errors 0)
      (eprintf "  Warning: %d lines could not be parsed\n" parse-errors))))

(defn analyze-logs
  "Analyze all access log files in directory"
  [log-dir]
  (def stats @{:hourly @{}
               :monthly @{}
               :total-hll (hll/new)})

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

  stats)

(defn top-n
  "Get top N items from a frequency table"
  [freq-table n]
  (def items (pairs freq-table))
  (def sorted (sort-by |(- (get $ 1)) items))
  (take n sorted))

(defn format-top-paths
  "Convert top-n paths to array of objects"
  [top-paths-tuples]
  (map (fn [[path count]]
         @{:path path :requests count})
       top-paths-tuples))

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
      :unique_ips (length (entry :ips))  # Just the count, not HLL
      :bots (entry :bots)
      :humans (entry :humans)
      :status_codes (entry :status-codes)
      :top_paths (format-top-paths (top-n (entry :paths) 20))}))

(defn build-monthly-json
  "Build monthly summary for JSON output.
   Monthly data includes HLL for merging after log rotation."
  [monthly-stats]
  (def months (sort (keys monthly-stats)))
  (map (fn [month]
         (def entry (get monthly-stats month))
         @{:month month
           :requests (entry :requests)
           :unique_visitors_estimate (math/round (hll/count-estimate (entry :hll)))
           :unique_visitors_hll (hll/to-base64 (entry :hll))})
       months))

(defn output-json
  "Output analytics data as JSON"
  [stats log-files]
  (def hourly-json (build-hourly-json (stats :hourly)))
  (def monthly-json (build-monthly-json (stats :monthly)))

  (def result
    @{:generated_at (os/strftime "%Y-%m-%dT%H:%M:%SZ" (os/time))
      :log_files_processed log-files
      :total_requests (sum (map |(get $ :requests) (values (stats :monthly))))
      :total_unique_visitors_estimate (math/round (hll/count-estimate (stats :total-hll)))
      :total_unique_visitors_hll (hll/to-base64 (stats :total-hll))
      :hourly hourly-json
      :monthly monthly-json})

  # Output pretty-printed JSON
  (print (json/encode-pretty result)))

(defn main [& args]
  # In compiled binaries, args[0] is the executable path
  # User arguments start at index 1
  (def log-dir
    (if (> (length args) 1)
      (get args 1)
      "/Users/eph/plazaespana.info/awstats-data/logs"))

  (eprintf "Analyzing Apache access logs in: %s\n\n" log-dir)

  # Get list of log files before analysis
  (def log-files
    (sort (filter |(string/has-prefix? "access_log" $)
                  (os/dir log-dir))))

  # Analyze logs
  (def stats (analyze-logs log-dir))

  # Output JSON to stdout (status messages go to stderr)
  (output-json stats log-files))
