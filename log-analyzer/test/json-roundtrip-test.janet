# Tests for JSON encoder/decoder compatibility.
#
# Covers:
# - roundtrip of nested analytics-like payloads
# - escaped string handling
# - scalar/literal decoding

(import ../src/json :as enc)
(import ../src/json_decode :as dec)

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

(defn- assert-roundtrip
  [val]
  (def encoded (enc/encode-pretty val))
  (def decoded (dec/decode encoded))
  (def re-encoded (enc/encode-pretty decoded))
  (assert (= re-encoded encoded)
          (string "Roundtrip mismatch.\nencoded:\n" encoded "\nre-encoded:\n" re-encoded)))

(deftest "roundtrip nested analytics payload"
  (assert-roundtrip
    @{:generated_at "2026-02-21T00:00:00Z"
      :total_requests 1234
      :total_unique_visitors_estimate 321
      :month_count 2
      :months @[
        @{:month "2026-01"
          :requests 700
          :unique_visitors_exact 200
          :unique_visitors_estimate 205}
        @{:month "2026-02"
          :requests 534
          :unique_visitors_exact 150
          :unique_visitors_estimate 152}]
      :flags @{:ok true :preview nil}
      :samples @[1 2 3 4.5]}))

(deftest "roundtrip escaped strings"
  (def payload @{:line "line1\nline2\t\"quote\"\\slash"})
  (def decoded (dec/decode (enc/encode payload)))
  (assert (= (decoded :line) (payload :line))
          "Escaped string should survive encode/decode"))

(deftest "decode literals and numeric formats"
  (def parsed
    (dec/decode "{\"a\":true,\"b\":false,\"c\":null,\"d\":123,\"e\":-4.5,\"f\":1.2e3}"))
  (assert (= (parsed :a) true))
  (assert (= (parsed :b) false))
  (assert (= (parsed :c) nil))
  (assert (= (parsed :d) 123))
  (assert (= (parsed :e) -4.5))
  (assert (= (parsed :f) 1200)))

(deftest "roundtrip top-level array"
  (assert-roundtrip
    @[
      @{:hour "2026-02-01T00:00:00Z" :requests 10 :bots 2}
      @{:hour "2026-02-01T01:00:00Z" :requests 12 :bots 1}
      @{:hour "2026-02-01T02:00:00Z" :requests 8 :bots 0}]))

(print)
(print "=" (string/repeat "=" 50))
(printf "JSON Roundtrip Tests: %d/%d passed" tests-passed tests-run)
(print "=" (string/repeat "=" 50))

(when (not= tests-passed tests-run)
  (os/exit 1))
