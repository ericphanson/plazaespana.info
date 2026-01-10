#!/usr/bin/env janet
# Compare HyperLogLog estimates against exact unique counts from real logs
#
# Usage: janet scripts/hll-accuracy-check.janet [log-dir]

(import ../src/hll)

(def apache-log-peg
  "PEG grammar for Apache Combined Log Format"
  ~{:main (* :ip :s "-" :s "-" :s "[" :date "]" :s (to "\n"))
    :ip (<- (some (+ :d (set ".:a-f"))))
    :date (* :day "/" :month "/" :year ":" :time :s :tz)
    :day (<- (between 1 2 :d))
    :month (<- (+ "Jan" "Feb" "Mar" "Apr" "May" "Jun"
                  "Jul" "Aug" "Sep" "Oct" "Nov" "Dec"))
    :year (<- (repeat 4 :d))
    :time (to :s)
    :tz (* (set "+-") (repeat 4 :d))
    :s (some (set " \t"))})

(def month-nums
  {"Jan" "01" "Feb" "02" "Mar" "03" "Apr" "04"
   "May" "05" "Jun" "06" "Jul" "07" "Aug" "08"
   "Sep" "09" "Oct" "10" "Nov" "11" "Dec" "12"})

(defn parse-log-line [line]
  (when-let [match (peg/match apache-log-peg line)]
    (let [[ip day month year] match
          month-num (get month-nums month)]
      [ip (string year "-" month-num)])))

(defn process-logs [log-dir]
  "Process all logs and return both exact counts and HLL estimates per month"
  (def exact-stats @{})    # month -> set of IPs
  (def hll-stats @{})      # month -> HLL sketch
  (def all-ips-exact @{})  # all IPs across all months
  (def all-ips-hll (hll/new))

  (def log-files
    (sort (filter |(string/has-prefix? "access_log" $)
                  (os/dir log-dir))))

  (when (empty? log-files)
    (eprint "No access_log files found in " log-dir)
    (os/exit 1))

  (print "Processing " (length log-files) " log files...")
  (print)

  (var total-lines 0)

  (each filename log-files
    (def filepath (string log-dir "/" filename))
    (prin "  " filename " ")
    (with [f (file/open filepath :r)]
      (var file-lines 0)
      (loop [line :iterate (file/read f :line)]
        (++ file-lines)
        (++ total-lines)
        (when (zero? (% file-lines 10000))
          (prin "."))
        (when-let [[ip month-year] (parse-log-line line)]
          # Exact counting
          (unless (get exact-stats month-year)
            (put exact-stats month-year @{}))
          (put (get exact-stats month-year) ip true)
          (put all-ips-exact ip true)

          # HLL counting
          (unless (get hll-stats month-year)
            (put hll-stats month-year (hll/new)))
          (hll/add (get hll-stats month-year) ip)
          (hll/add all-ips-hll ip)))
      (print " " file-lines " lines")))

  (print)
  (printf "Total lines processed: %d" total-lines)

  {:exact exact-stats
   :hll hll-stats
   :all-exact all-ips-exact
   :all-hll all-ips-hll})

(defn print-comparison [results]
  "Print comparison between exact and HLL counts"
  (def exact (results :exact))
  (def hll-sketches (results :hll))
  (def all-exact (results :all-exact))
  (def all-hll (results :all-hll))

  (print)
  (print "=" (string/repeat "=" 70))
  (print "HyperLogLog Accuracy Report")
  (print "=" (string/repeat "=" 70))
  (print)
  (printf "%-10s %10s %10s %10s %10s" "Month" "Exact" "HLL Est" "Error" "Error %")
  (print "-" (string/repeat "-" 70))

  (var total-abs-error 0)
  (var count 0)

  (each month (sort (keys exact))
    (def exact-count (length (get exact month)))
    (def hll-estimate (math/round (hll/count-estimate (get hll-sketches month))))
    (def error (- hll-estimate exact-count))
    (def error-pct (* 100 (/ (math/abs error) exact-count)))

    (printf "%-10s %10d %10d %+10d %9.2f%%"
            month exact-count hll-estimate error error-pct)

    (set total-abs-error (+ total-abs-error (math/abs error-pct)))
    (++ count))

  (print "-" (string/repeat "-" 70))

  # All-time totals
  (def exact-total (length all-exact))
  (def hll-total (math/round (hll/count-estimate all-hll)))
  (def total-error (- hll-total exact-total))
  (def total-error-pct (* 100 (/ (math/abs total-error) exact-total)))

  (printf "%-10s %10d %10d %+10d %9.2f%%"
          "ALL TIME" exact-total hll-total total-error total-error-pct)

  (print "=" (string/repeat "=" 70))
  (print)

  # Summary stats
  (printf "Average monthly error: %.2f%%" (/ total-abs-error count))
  (printf "All-time error:        %.2f%%" total-error-pct)
  (print)

  # HLL theoretical error
  (print "Expected HLL error (theoretical): ~0.81% for p=14 (16384 buckets)")
  (print)

  # Memory comparison
  (def hll-size (* 16384 5 (/ 1 8)))  # 5 bits per bucket
  (def exact-size-estimate (* exact-total 40))  # ~40 bytes per IP string in hash table
  (printf "Memory usage comparison:")
  (printf "  HLL sketch:     %6.1f KB (fixed)" (/ hll-size 1024))
  (printf "  Exact hash set: %6.1f KB (grows with cardinality)" (/ exact-size-estimate 1024)))

(defn main [& args]
  (def log-dir
    (if (> (length args) 1)
      (get args 1)
      "/Users/eph/plazaespana.info/awstats-data/logs"))

  (print "HLL Accuracy Check")
  (print "Log directory: " log-dir)
  (print)

  (def results (process-logs log-dir))
  (print-comparison results))
