# Rebuild lifetime.json from persisted month shards.

(import ./hll)
(import ./json)
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

(defn- parse-request-count
  [summary month-path]
  (def raw (get summary :requests 0))
  (if (number? raw)
    raw
    (or (scan-number (string raw))
        (error (string "Invalid summary.requests in " month-path)))))

(defn- round-half-up
  "Match publish script semantics: floor(x + 0.5)."
  [x]
  (math/floor (+ x 0.5)))

(defn- load-log-files-processed
  [lifetime-source]
  (if (or (nil? lifetime-source) (nil? (os/stat lifetime-source)))
    @[]
    (let [source-doc (load-json lifetime-source)
          log-files (get source-doc :log_files_processed nil)]
      (if (or (array? log-files) (tuple? log-files))
        log-files
        @[]))))

(defn rebuild-lifetime-from-json-dir
  "Rebuild lifetime.json from month shards in json-dir.
   Optionally preserve :log_files_processed from lifetime-source.
   Returns {:lifetime-path ... :month-count ... :generated-at ...}."
  [json-dir &opt lifetime-source]
  (def month-files (sort (filter month-file? (os/dir json-dir))))
  (when (empty? month-files)
    (error (string "No month JSON files found in " json-dir)))

  (var total-requests 0)
  (var merged-hll (hll/new))
  (def months @[])

  (each fname month-files
    (def month-path (string json-dir "/" fname))
    (def doc (load-json month-path))
    (def summary (get doc :summary nil))
    (unless (or (table? summary) (struct? summary))
      (error (string "missing summary object in " month-path)))

    (when (nil? (get summary :month nil))
      (put summary :month (string/slice fname 0 7)))

    (+= total-requests (parse-request-count summary month-path))

    (def hll-b64 (get summary :unique_visitors_hll nil))
    (unless (and (string? hll-b64) (> (length hll-b64) 0))
      (error (string "missing summary.unique_visitors_hll in " month-path)))

    (set merged-hll (hll/merge merged-hll (hll/from-base64 hll-b64)))
    (array/push months summary))

  (def months-sorted (sort-by |(or ($ :month) "") months))
  (def generated-at (os/strftime "%Y-%m-%dT%H:%M:%SZ" (os/time)))
  (def lifetime-doc
    @{:generated_at generated-at
      :log_files_processed (load-log-files-processed lifetime-source)
      :total_requests total-requests
      :total_unique_visitors_estimate (round-half-up (hll/count-estimate merged-hll))
      :total_unique_visitors_hll (hll/to-base64 merged-hll)
      :months months-sorted})

  (def lifetime-path (string json-dir "/lifetime.json"))
  (write-file lifetime-path (json/encode-pretty lifetime-doc))
  @{:lifetime-path lifetime-path
    :month-count (length month-files)
    :generated-at generated-at})
