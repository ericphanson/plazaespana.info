# HyperLogLog Test Suite
#
# Tests for the HLL implementation covering:
# - Basic operations (new, add, count)
# - Hash function quality
# - Cardinality estimation accuracy
# - Merge operations
# - Serialization/deserialization
# - Edge cases
#
# Run with: jpm test

(import ../src/hll)

# ============================================================================
# Test Helpers
# ============================================================================

(defn- approx=
  "Check if two numbers are approximately equal within relative error"
  [a b tolerance]
  (<= (math/abs (- a b)) (* tolerance (max (math/abs a) (math/abs b) 1))))

(defn- assert-approx
  "Assert that actual is within tolerance of expected"
  [actual expected tolerance &opt msg]
  (def message (or msg (string/format "Expected %f ≈ %f (±%.1f%%)"
                                       actual expected (* tolerance 100))))
  (assert (approx= actual expected tolerance) message))

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
# Hash Function Tests
# ============================================================================

(deftest "hash-value produces consistent results"
  (def h1 (hll/hash-value "test"))
  (def h2 (hll/hash-value "test"))
  (assert (= h1 h2) "Same input should produce same hash"))

(deftest "hash-value produces different results for different inputs"
  (def h1 (hll/hash-value "test1"))
  (def h2 (hll/hash-value "test2"))
  (assert (not= h1 h2) "Different inputs should produce different hashes"))

(deftest "hash-value handles various types"
  # Strings
  (assert (number? (hll/hash-value "hello")))
  # Numbers (converted to string)
  (assert (number? (hll/hash-value 12345)))
  # Keywords
  (assert (number? (hll/hash-value :keyword))))

(deftest "hash-value produces 32-bit values"
  (for i 0 100
    (def h (hll/hash-value (string "test" i)))
    (assert (>= h 0) "Hash should be non-negative")
    (assert (< h 0x100000000) "Hash should be < 2^32")))

# ============================================================================
# Basic Operation Tests
# ============================================================================

(deftest "new creates empty HLL"
  (def h (hll/new))
  (assert (hll/empty? h) "New HLL should be empty")
  (assert (= (hll/count-estimate h) 0) "Empty HLL should have count 0"))

(deftest "add returns the HLL (for chaining)"
  (def h (hll/new))
  (def result (hll/add h "test"))
  (assert (= h result) "add should return the HLL"))

(deftest "add makes HLL non-empty"
  (def h (hll/new))
  (hll/add h "test")
  (assert (not (hll/empty? h)) "HLL should not be empty after add"))

(deftest "clear resets HLL to empty"
  (def h (hll/new))
  (hll/add h "test1")
  (hll/add h "test2")
  (hll/clear h)
  (assert (hll/empty? h) "HLL should be empty after clear"))

# ============================================================================
# Cardinality Estimation Tests
# ============================================================================

(deftest "count-estimate returns 0 for empty HLL"
  (def h (hll/new))
  (assert (= (hll/count-estimate h) 0)))

(deftest "count-estimate is approximately correct for small sets"
  (def h (hll/new))
  (for i 0 100
    (hll/add h (string "item-" i)))
  (def estimate (hll/count-estimate h))
  # Allow 20% error for small sets (HLL is less accurate for small n)
  (assert-approx estimate 100 0.20 "100 items"))

(deftest "count-estimate is approximately correct for medium sets"
  (def h (hll/new))
  (for i 0 1000
    (hll/add h (string "item-" i)))
  (def estimate (hll/count-estimate h))
  # Allow 10% error for medium sets
  (assert-approx estimate 1000 0.10 "1000 items"))

(deftest "count-estimate is approximately correct for larger sets"
  (def h (hll/new))
  (for i 0 10000
    (hll/add h (string "item-" i)))
  (def estimate (hll/count-estimate h))
  # Allow 5% error for larger sets (HLL is more accurate)
  (assert-approx estimate 10000 0.05 "10000 items"))

(deftest "duplicate items don't increase count"
  (def h (hll/new))
  # Add same 100 items 10 times each
  (for _ 0 10
    (for i 0 100
      (hll/add h (string "item-" i))))
  (def estimate (hll/count-estimate h))
  # Should still be approximately 100
  (assert-approx estimate 100 0.20 "100 unique items added 10x each"))

# ============================================================================
# Merge Tests
# ============================================================================

(deftest "merge of empty HLLs is empty"
  (def h1 (hll/new))
  (def h2 (hll/new))
  (def merged (hll/merge h1 h2))
  (assert (hll/empty? merged) "Merge of empty HLLs should be empty"))

(deftest "merge of HLL with empty HLL preserves count"
  (def h1 (hll/new))
  (for i 0 100
    (hll/add h1 (string "item-" i)))
  (def h2 (hll/new))
  (def merged (hll/merge h1 h2))
  (def original-count (hll/count-estimate h1))
  (def merged-count (hll/count-estimate merged))
  (assert-approx merged-count original-count 0.01
                 "Merge with empty should preserve count"))

(deftest "merge combines disjoint sets correctly"
  (def h1 (hll/new))
  (def h2 (hll/new))
  # Add 500 items to each, no overlap
  (for i 0 500
    (hll/add h1 (string "set1-item-" i)))
  (for i 0 500
    (hll/add h2 (string "set2-item-" i)))
  (def merged (hll/merge h1 h2))
  (def estimate (hll/count-estimate merged))
  # Should be approximately 1000
  (assert-approx estimate 1000 0.10 "Merge of disjoint 500+500"))

(deftest "merge handles overlapping sets correctly"
  (def h1 (hll/new))
  (def h2 (hll/new))
  # Add items 0-499 to h1, 250-749 to h2
  # Overlap is 250-499 (250 items), total unique is 750
  (for i 0 500
    (hll/add h1 (string "item-" i)))
  (for i 250 750
    (hll/add h2 (string "item-" i)))
  (def merged (hll/merge h1 h2))
  (def estimate (hll/count-estimate merged))
  (assert-approx estimate 750 0.10 "Merge with 50% overlap"))

(deftest "merge is commutative"
  (def h1 (hll/new))
  (def h2 (hll/new))
  (for i 0 200
    (hll/add h1 (string "a-" i)))
  (for i 0 300
    (hll/add h2 (string "b-" i)))
  (def m1 (hll/merge h1 h2))
  (def m2 (hll/merge h2 h1))
  (def c1 (hll/count-estimate m1))
  (def c2 (hll/count-estimate m2))
  (assert-approx c1 c2 0.001 "Merge should be commutative"))

(deftest "merge does not modify originals"
  (def h1 (hll/new))
  (def h2 (hll/new))
  (for i 0 100
    (hll/add h1 (string "a-" i)))
  (for i 0 100
    (hll/add h2 (string "b-" i)))
  (def c1-before (hll/count-estimate h1))
  (def c2-before (hll/count-estimate h2))
  (hll/merge h1 h2)
  (def c1-after (hll/count-estimate h1))
  (def c2-after (hll/count-estimate h2))
  (assert (= c1-before c1-after) "h1 should not be modified")
  (assert (= c2-before c2-after) "h2 should not be modified"))

# ============================================================================
# Serialization Tests
# ============================================================================

(deftest "serialize returns a buffer"
  (def h (hll/new))
  (def buf (hll/serialize h))
  (assert (buffer? buf) "serialize should return a buffer"))

(deftest "serialize uses sparse format for sparse HLLs"
  (def h (hll/new))
  (def buf (hll/serialize h))
  # Empty HLL: header (6 bytes) + count (2 bytes) = 8 bytes
  (assert (= (length buf) 8)
          (string/format "Empty HLL should be 8 bytes, got %d" (length buf)))
  # Verify header
  (assert (= (string/slice buf 0 3) "HLL") "Should have HLL magic")
  (assert (= (get buf 3) 1) "Should be version 1")
  (assert (= (get buf 4) 14) "Should have precision 14")
  (assert (= (get buf 5) 1) "Empty HLL should use sparse format")
  # After adding items, should grow
  (for i 0 100
    (hll/add h (string "item-" i)))
  (def buf2 (hll/serialize h))
  # ~100 non-zero buckets * 3 bytes + 8 header = ~308 bytes
  (assert (and (> (length buf2) 100) (< (length buf2) 500))
          (string/format "Sparse HLL with 100 items should be 100-500 bytes, got %d" (length buf2))))

(deftest "deserialize restores empty HLL"
  (def h1 (hll/new))
  (def buf (hll/serialize h1))
  (def h2 (hll/deserialize buf))
  (assert (hll/empty? h2) "Deserialized empty HLL should be empty"))

(deftest "deserialize preserves count estimate"
  (def h1 (hll/new))
  (for i 0 1000
    (hll/add h1 (string "item-" i)))
  (def c1 (hll/count-estimate h1))
  (def buf (hll/serialize h1))
  (def h2 (hll/deserialize buf))
  (def c2 (hll/count-estimate h2))
  (assert-approx c2 c1 0.001 "Deserialized count should match original"))

(deftest "to-base64 returns a string"
  (def h (hll/new))
  (def s (hll/to-base64 h))
  (assert (string? s) "to-base64 should return a string"))

(deftest "from-base64 restores HLL"
  (def h1 (hll/new))
  (for i 0 500
    (hll/add h1 (string "item-" i)))
  (def c1 (hll/count-estimate h1))
  (def b64 (hll/to-base64 h1))
  (def h2 (hll/from-base64 b64))
  (def c2 (hll/count-estimate h2))
  (assert-approx c2 c1 0.001 "Base64 round-trip should preserve count"))

(deftest "base64 encoding is valid"
  (def h (hll/new))
  (for i 0 100
    (hll/add h (string "x-" i)))
  (def b64 (hll/to-base64 h))
  # Should only contain valid base64 characters
  (each c b64
    (assert (or (and (>= c (chr "A")) (<= c (chr "Z")))
                (and (>= c (chr "a")) (<= c (chr "z")))
                (and (>= c (chr "0")) (<= c (chr "9")))
                (= c (chr "+"))
                (= c (chr "/"))
                (= c (chr "=")))
            (string/format "Invalid base64 char: %c" c))))

# ============================================================================
# Edge Cases
# ============================================================================

(deftest "handles empty string input"
  (def h (hll/new))
  (hll/add h "")
  (assert (not (hll/empty? h)) "Empty string is still a valid input"))

(deftest "handles very long strings"
  (def h (hll/new))
  (def long-str (string/repeat "x" 10000))
  (hll/add h long-str)
  (assert (not (hll/empty? h)) "Long string should be handled"))

(deftest "handles unicode strings"
  (def h (hll/new))
  (hll/add h "日本語")
  (hll/add h "émojis 🎉")
  (hll/add h "中文")
  (def estimate (hll/count-estimate h))
  (assert (> estimate 0) "Unicode strings should be handled"))

(deftest "single item has count ~1"
  (def h (hll/new))
  (hll/add h "single")
  (def estimate (hll/count-estimate h))
  # Single item might estimate anywhere from 1-3 due to HLL nature
  (assert (and (>= estimate 0.5) (<= estimate 5))
          (string/format "Single item estimate %f should be near 1" estimate)))

# ============================================================================
# IP Address Specific Tests (for log analysis use case)
# ============================================================================

(deftest "handles IPv4 addresses"
  (def h (hll/new))
  (for i 0 256
    (hll/add h (string/format "192.168.1.%d" i)))
  (def estimate (hll/count-estimate h))
  (assert-approx estimate 256 0.15 "256 unique IPv4 addresses"))

(deftest "handles IPv6 addresses"
  (def h (hll/new))
  (for i 0 100
    (hll/add h (string/format "2001:db8::%x" i)))
  (def estimate (hll/count-estimate h))
  (assert-approx estimate 100 0.20 "100 unique IPv6 addresses"))

(deftest "realistic log analysis scenario"
  # Simulate a day of log data with returning visitors
  (def hourly-hlls @[])

  # Create 24 hourly HLLs
  (for hour 0 24
    (def h (hll/new))
    # Each hour has ~50 unique IPs, with some overlap between hours
    (for i 0 50
      # 30% of IPs are "returning" (same across hours)
      (if (< i 15)
        (hll/add h (string/format "returning-%d" i))
        (hll/add h (string/format "hour%d-unique-%d" hour i))))
    (array/push hourly-hlls h))

  # Merge all hourly HLLs to get daily unique
  (var daily (hll/new))
  (each hourly hourly-hlls
    (set daily (hll/merge daily hourly)))

  (def daily-estimate (hll/count-estimate daily))
  # Expected: 15 returning + (24 * 35 unique per hour) = 15 + 840 = 855
  (assert-approx daily-estimate 855 0.10
                 "Daily unique from merged hourly HLLs"))

# ============================================================================
# Test Runner
# ============================================================================

(print)
(print "=" (string/repeat "=" 50))
(printf "HLL Tests: %d/%d passed" tests-passed tests-run)
(print "=" (string/repeat "=" 50))

(when (not= tests-passed tests-run)
  (os/exit 1))
