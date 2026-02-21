# Tests for main.janet functions
#
# Covers: parse-log-line, extract-path, is-bot?, classify-referrer,
#         classify-browser, classify-platform, timezone handling
#
# Run with: jpm test

(import ../src/main :as m)

# ============================================================================
# Test Helpers
# ============================================================================

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

# ============================================================================
# parse-log-line tests
# ============================================================================

(deftest "parse-log-line parses standard IPv4 line"
  (def line `4.230.44.177 - - [04/Jan/2026:00:35:02 +0000] "GET /index.html HTTP/1.1" 200 5432 "https://www.google.com/" "Mozilla/5.0"`)
  (def result (m/parse-log-line line))
  (assert result "should parse successfully")
  (assert (= (result :ip) "4.230.44.177") "ip")
  (assert (= (result :hour) "2026-01-04T00:00:00Z") "hour")
  (assert (= (result :month) "2026-01") "month")
  (assert (= (result :status) "200") "status")
  (assert (= (result :bytes) 5432) "bytes")
  (assert (= (result :referrer) "https://www.google.com/") "referrer"))

(deftest "parse-log-line parses IPv6 address"
  (def line `2001:db8::1 - - [15/Feb/2026:14:30:00 +0000] "GET / HTTP/1.1" 200 1024 "-" "Mozilla/5.0"`)
  (def result (m/parse-log-line line))
  (assert result "should parse successfully")
  (assert (= (result :ip) "2001:db8::1") "ipv6 address"))

(deftest "parse-log-line handles missing bytes (dash)"
  (def line `1.2.3.4 - - [01/Mar/2026:10:00:00 +0000] "GET / HTTP/1.1" 304 - "-" "Mozilla/5.0"`)
  (def result (m/parse-log-line line))
  (assert result "should parse successfully")
  (assert (= (result :bytes) 0) "dash bytes should be 0"))

(deftest "parse-log-line handles single-digit day"
  (def line `1.2.3.4 - - [5/Jan/2026:08:00:00 +0000] "GET / HTTP/1.1" 200 100 "-" "Mozilla/5.0"`)
  (def result (m/parse-log-line line))
  (assert result "should parse single-digit day")
  (assert (= (result :hour) "2026-01-05T08:00:00Z") "day should be zero-padded"))

(deftest "parse-log-line returns nil for garbage"
  (def result (m/parse-log-line "this is not a log line"))
  (assert (nil? result) "should return nil for unparseable lines"))

(deftest "parse-log-line returns nil for empty string"
  (def result (m/parse-log-line ""))
  (assert (nil? result) "should return nil for empty string"))

# ============================================================================
# Timezone conversion tests
# ============================================================================

(deftest "parse-log-line converts +0000 to UTC correctly"
  (def line `1.2.3.4 - - [15/Jan/2026:23:30:00 +0000] "GET / HTTP/1.1" 200 100 "-" "Mozilla/5.0"`)
  (def result (m/parse-log-line line))
  (assert (= (result :hour) "2026-01-15T23:00:00Z") "UTC+0 should stay the same"))

(deftest "parse-log-line converts +0100 to UTC"
  (def line `1.2.3.4 - - [15/Jan/2026:14:30:00 +0100] "GET / HTTP/1.1" 200 100 "-" "Mozilla/5.0"`)
  (def result (m/parse-log-line line))
  (assert (= (result :hour) "2026-01-15T13:00:00Z") "+0100 at 14:xx should become 13:xx UTC"))

(deftest "parse-log-line converts -0500 to UTC"
  (def line `1.2.3.4 - - [15/Jan/2026:20:30:00 -0500] "GET / HTTP/1.1" 200 100 "-" "Mozilla/5.0"`)
  (def result (m/parse-log-line line))
  (assert (= (result :hour) "2026-01-16T01:00:00Z") "-0500 at 20:xx should become 01:xx next day"))

(deftest "parse-log-line handles timezone day rollback"
  (def line `1.2.3.4 - - [01/Jan/2026:00:30:00 +0100] "GET / HTTP/1.1" 200 100 "-" "Mozilla/5.0"`)
  (def result (m/parse-log-line line))
  (assert (= (result :hour) "2025-12-31T23:00:00Z") "+0100 at 00:xx on Jan 1 should roll back to Dec 31"))

(deftest "parse-log-line handles timezone month rollover"
  (def line `1.2.3.4 - - [31/Jan/2026:23:30:00 -0200] "GET / HTTP/1.1" 200 100 "-" "Mozilla/5.0"`)
  (def result (m/parse-log-line line))
  (assert (= (result :hour) "2026-02-01T01:00:00Z") "-0200 at 23:xx on Jan 31 should become Feb 1")
  (assert (= (result :month) "2026-02") "month should also roll over"))

# ============================================================================
# extract-path tests
# ============================================================================

(deftest "extract-path gets path from GET request"
  (assert (= (m/extract-path "GET /index.html HTTP/1.1") "/index.html")))

(deftest "extract-path gets path from POST request"
  (assert (= (m/extract-path "POST /api/data HTTP/1.1") "/api/data")))

(deftest "extract-path handles path with query string"
  (assert (= (m/extract-path "GET /search?q=test HTTP/1.1") "/search?q=test")))

(deftest "extract-path returns / for empty request"
  (assert (= (m/extract-path "") "/")))

# ============================================================================
# is-bot? tests
# ============================================================================

(deftest "is-bot? detects Googlebot"
  (assert (m/is-bot? "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")))

(deftest "is-bot? detects curl"
  (assert (m/is-bot? "curl/7.68.0")))

(deftest "is-bot? detects Python requests"
  (assert (m/is-bot? "python-requests/2.28.0")))

(deftest "is-bot? does not flag real Chrome browser"
  (assert (not (m/is-bot? "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"))))

(deftest "is-bot? does not flag real Safari"
  (assert (not (m/is-bot? "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"))))

# ============================================================================
# classify-referrer tests
# ============================================================================

(deftest "classify-referrer returns direct for dash"
  (assert (= (m/classify-referrer "-") "direct")))

(deftest "classify-referrer returns direct for empty string"
  (assert (= (m/classify-referrer "") "direct")))

(deftest "classify-referrer detects Google as search"
  (assert (= (m/classify-referrer "https://www.google.com/") "search")))

(deftest "classify-referrer detects DuckDuckGo as search"
  (assert (= (m/classify-referrer "https://duckduckgo.com/?q=test") "search")))

(deftest "classify-referrer detects Bing as search"
  (assert (= (m/classify-referrer "https://www.bing.com/search?q=test") "search")))

(deftest "classify-referrer detects Twitter as social"
  (assert (= (m/classify-referrer "https://t.co/abc123") "social")))

(deftest "classify-referrer detects Reddit as social"
  (assert (= (m/classify-referrer "https://www.reddit.com/r/madrid") "social")))

(deftest "classify-referrer detects internal referrer"
  (assert (= (m/classify-referrer "https://plazaespana.info/events.json") "internal")))

(deftest "classify-referrer returns external for unknown domains"
  (assert (= (m/classify-referrer "https://someotherblog.com/link") "external")))

# ============================================================================
# classify-browser tests
# ============================================================================

(deftest "classify-browser detects Chrome"
  (assert (= (m/classify-browser "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36") "Chrome")))

(deftest "classify-browser detects Edge (not Chrome)"
  (assert (= (m/classify-browser "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0") "Edge")))

(deftest "classify-browser detects Firefox"
  (assert (= (m/classify-browser "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0") "Firefox")))

(deftest "classify-browser detects Safari"
  (assert (= (m/classify-browser "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15") "Safari")))

(deftest "classify-browser returns other for unknown UA"
  (assert (= (m/classify-browser "SomeRandomAgent/1.0") "other")))

# ============================================================================
# classify-platform tests
# ============================================================================

(deftest "classify-platform detects Windows"
  (assert (= (m/classify-platform "Mozilla/5.0 (Windows NT 10.0; Win64; x64)") "Windows")))

(deftest "classify-platform detects macOS"
  (assert (= (m/classify-platform "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0)") "macOS")))

(deftest "classify-platform detects Android"
  (assert (= (m/classify-platform "Mozilla/5.0 (Linux; Android 13; SM-S908E)") "Android")))

(deftest "classify-platform detects iOS (iPhone)"
  (assert (= (m/classify-platform "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)") "iOS")))

(deftest "classify-platform detects iOS (iPad)"
  (assert (= (m/classify-platform "Mozilla/5.0 (iPad; CPU OS 17_0_3 like Mac OS X)") "iOS")))

(deftest "classify-platform detects Linux"
  (assert (= (m/classify-platform "Mozilla/5.0 (X11; Linux x86_64; rv:120.0)") "Linux")))

(deftest "classify-platform returns other for unknown"
  (assert (= (m/classify-platform "SomeRandomAgent/1.0") "other")))

# ============================================================================
# format-top-paths tests (other bucket)
# ============================================================================

(deftest "format-top-paths adds other bucket"
  (def paths @{"/" 100 "/a" 50 "/b" 30 "/c" 20 "/d" 10 "/e" 5})
  (def result (m/format-top-paths paths 3))
  # Should have top 3 + other
  (assert (= (length result) 4) (string/format "expected 4, got %d" (length result)))
  (def other-entry (find |(= ($ :path) "other") result))
  (assert other-entry "should have other bucket")
  (assert (= (other-entry :requests) 35) (string/format "other should be 35, got %d" (other-entry :requests))))

(deftest "format-top-paths no other bucket when under limit"
  (def paths @{"/" 100 "/a" 50})
  (def result (m/format-top-paths paths 5))
  (assert (= (length result) 2) "should only have 2 entries")
  (def other-entry (find |(= ($ :path) "other") result))
  (assert (nil? other-entry) "should not have other bucket"))

# ============================================================================
# Test Runner
# ============================================================================

(print)
(print "=" (string/repeat "=" 50))
(printf "Main Tests: %d/%d passed" tests-passed tests-run)
(print "=" (string/repeat "=" 50))

(when (not= tests-passed tests-run)
  (os/exit 1))
