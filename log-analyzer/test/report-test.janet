# Tests for report generation safety and behavior.
#
# Covers:
# - HTML escaping of dynamic values in report rendering

(import ../src/report :as report)

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

(deftest "build-html-report escapes unsafe path content"
  (def html
    (report/build-html-report
      @[@{:month "2026-02"
          :requests 1
          :unique_visitors_exact 1
          :unique_visitors_estimate 1}]
      @[@{:hour "2026-02-21T00:00:00Z"
          :requests 1
          :bots 0
          :scans 0
          :visitors 1
          :status_codes @{"200" 1}
          :status_by_type @{:bots @{} :scans @{} :visitors @{"200" 1}}
          :top_paths @[@{:path "/<script>alert(1)</script>" :requests 1}]
          :referrer_categories @{"direct" 1}
          :browsers @{"Chrome" 1}
          :platforms @{"macOS" 1}}]
      1
      1
      "2026-02-21T00:00:00Z"
      @[]))

  (assert (string/find "&lt;script&gt;alert(1)&lt;/script&gt;" html)
          "Escaped path should be present")
  (assert (nil? (string/find "<script>alert(1)</script>" html))
          "Raw script tag should not be present"))

(print)
(print "=" (string/repeat "=" 50))
(printf "Report Tests: %d/%d passed" tests-passed tests-run)
(print "=" (string/repeat "=" 50))

(when (not= tests-passed tests-run)
  (os/exit 1))
