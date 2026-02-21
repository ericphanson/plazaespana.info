# Tests for lifetime rebuild mode.

(import ../src/hll)
(import ../src/json)
(import ../src/json_decode :as decode)
(import ../src/rebuild :as rebuild)

(var tests-run 0)
(var tests-passed 0)

(defmacro deftest
  "Define a named test"
  [name & body]
  ~(do
     (++ tests-run)
     (try
       (do
         ,;body
         (++ tests-passed)
         (print "✓ " ,(string name)))
       ([err]
        (print "✗ " ,(string name) ": " err)))))

(defn- rm-rf
  [path]
  (os/execute ["/bin/rm" "-rf" path]))

(defn- test-dir
  [label]
  (def path
    (string "/tmp/log-analyzer-rebuild-test-"
            label "-"
            (os/getpid) "-"
            tests-run "-"
            (math/floor (* (math/random) 1000000))))
  (rm-rf path)
  (os/mkdir path)
  path)

(defn- write-json-file
  [path doc]
  (spit path (string (json/encode-pretty doc) "\n")))

(defn- write-month-file
  [dir month requests hll-b64]
  (write-json-file
    (string dir "/" month ".json")
    @{:generated_at "2026-02-21T00:00:00Z"
      :month month
      :summary @{:month month
                 :requests requests
                 :unique_visitors_exact 0
                 :unique_visitors_estimate 0
                 :unique_visitors_hll hll-b64}
      :hourly @[]}))

(deftest "rebuild-lifetime merges HLLs and preserves log files"
  (def dir (test-dir "merge"))
  (try
    (do
      (def h1 (hll/new))
      (for i 0 2200
        (hll/add h1 (string "returning-" i)))

      (def h2 (hll/new))
      (for i 1200 3600
        (hll/add h2 (string "returning-" i)))

      (write-month-file dir "2026-02" 20 (hll/to-base64 h2))
      (write-month-file dir "2026-01" 10 (hll/to-base64 h1))
      (write-json-file
        (string dir "/generated-lifetime.json")
        @{:log_files_processed @["access_log" "access_log.1"]})

      (rebuild/rebuild-lifetime-from-json-dir dir (string dir "/generated-lifetime.json"))
      (def lifetime (decode/decode (slurp (string dir "/lifetime.json"))))
      (def merged (hll/merge h1 h2))
      (def expected-estimate (math/floor (+ (hll/count-estimate merged) 0.5)))

      (assert (= (lifetime :total_requests) 30)
              (string/format "expected total_requests=30, got %d" (lifetime :total_requests)))
      (assert (= (lifetime :total_unique_visitors_estimate) expected-estimate)
              "expected half-up rounded HLL estimate")
      (assert (= (lifetime :total_unique_visitors_hll) (hll/to-base64 merged))
              "expected merged HLL to match")

      (def months (lifetime :months))
      (assert (= (length months) 2) "expected two months")
      (assert (= ((get months 0) :month) "2026-01") "months should be sorted")
      (assert (= ((get months 1) :month) "2026-02") "months should be sorted")

      (def log-files (lifetime :log_files_processed))
      (assert (= (length log-files) 2) "expected preserved log file list")
      (assert (= (get log-files 0) "access_log") "expected first log file")
      (assert (= (get log-files 1) "access_log.1") "expected second log file"))
    ([err]
     (rm-rf dir)
     (error err)))
  (rm-rf dir))

(deftest "rebuild-lifetime errors when month summary is missing HLL"
  (def dir (test-dir "missing-hll"))
  (try
    (do
      (write-json-file
        (string dir "/2026-02.json")
        @{:generated_at "2026-02-21T00:00:00Z"
          :month "2026-02"
          :summary @{:month "2026-02" :requests 10}
          :hourly @[]})

      (var failed false)
      (try
        (rebuild/rebuild-lifetime-from-json-dir dir nil)
        ([err]
         (set failed true)
         (assert (string/find "missing summary.unique_visitors_hll" (string err))
                 "error should mention missing HLL")))
      (assert failed "expected rebuild to fail when HLL is missing"))
    ([err]
     (rm-rf dir)
     (error err)))
  (rm-rf dir))

(print)
(print "=" (string/repeat "=" 50))
(printf "Rebuild Tests: %d/%d passed" tests-passed tests-run)
(print "=" (string/repeat "=" 50))

(when (not= tests-passed tests-run)
  (os/exit 1))
