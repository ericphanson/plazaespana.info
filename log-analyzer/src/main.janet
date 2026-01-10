# Parse Apache access logs and count unique visitors per month

(def apache-log-peg
  "PEG grammar for Apache Combined Log Format"
  ~{:main (* :ip :s "-" :s "-" :s "[" :date "]" :s (to "\n"))
    :ip (<- (some (+ :d (set ".:a-f"))))  # IPv4 or IPv6
    :date (* :day "/" :month "/" :year ":" :time :s :tz)
    :day (<- (between 1 2 :d))
    :month (<- (+ "Jan" "Feb" "Mar" "Apr" "May" "Jun"
                  "Jul" "Aug" "Sep" "Oct" "Nov" "Dec"))
    :year (<- (repeat 4 :d))
    :time (to :s)
    :tz (* (set "+-") (repeat 4 :d))
    :s (some (set " \t"))})

(def month-nums
  "Map month names to numbers"
  {"Jan" "01" "Feb" "02" "Mar" "03" "Apr" "04"
   "May" "05" "Jun" "06" "Jul" "07" "Aug" "08"
   "Sep" "09" "Oct" "10" "Nov" "11" "Dec" "12"})

(defn parse-log-line
  "Parse a single Apache log line and return [ip month-year] or nil"
  [line]
  (when-let [match (peg/match apache-log-peg line)]
    (let [[ip day month year] match
          month-num (get month-nums month)]
      [ip (string year "-" month-num)])))

(defn process-log-file
  "Process a log file and accumulate unique IPs per month"
  [filepath stats]
  (print "Processing: " filepath)
  (with [f (file/open filepath :r)]
    (var line-count 0)
    (loop [line :iterate (file/read f :line)]
      (++ line-count)
      (when (zero? (% line-count 10000))
        (prin "."))
      (when-let [[ip month-year] (parse-log-line line)]
        # Get or create set for this month
        (def month-ips (get stats month-year @{}))
        (put month-ips ip true)
        (put stats month-year month-ips))))
  (print))

(defn analyze-logs
  "Analyze all access log files in directory"
  [log-dir]
  (def stats @{})

  # Find all access_log* files
  (def log-files
    (sort (filter |(string/has-prefix? "access_log" $)
                  (os/dir log-dir))))

  (when (empty? log-files)
    (eprint "No access_log files found in " log-dir)
    (os/exit 1))

  (print "Found " (length log-files) " log files")
  (print)

  # Process each log file
  (each filename log-files
    (def filepath (string log-dir "/" filename))
    (process-log-file filepath stats))

  stats)

(defn print-results
  "Print unique visitor counts per month"
  [stats]
  (print)
  (print "=" (string/repeat "=" 50))
  (print "Unique Visitors Per Month")
  (print "=" (string/repeat "=" 50))
  (print)

  # Sort months chronologically
  (def sorted-months (sort (keys stats)))

  (var total-unique 0)
  (def all-ips @{})

  (each month sorted-months
    (def month-ips (get stats month))
    (def count (length month-ips))
    (printf "%s: %6d unique visitors" month count)

    # Track all unique IPs across all months
    (eachp [ip _] month-ips
      (put all-ips ip true)))

  (print)
  (print "-" (string/repeat "-" 50))
  (printf "Total unique visitors (all time): %d" (length all-ips))
  (print "=" (string/repeat "=" 50)))

(defn main [& args]
  # In compiled binaries, args[0] is the executable path
  # User arguments start at index 1
  (def log-dir
    (if (> (length args) 1)
      (get args 1)
      "/Users/eph/plazaespana.info/awstats-data/logs"))

  (print "Analyzing Apache access logs in: " log-dir)
  (print)

  # Analyze logs
  (def stats (analyze-logs log-dir))

  # Print results
  (print-results stats))
