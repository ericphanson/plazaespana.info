# HyperLogLog Implementation for Janet
#
# HyperLogLog is a probabilistic cardinality estimation algorithm.
# It uses O(m) space to estimate cardinality with ~1.04/sqrt(m) standard error.
#
# This implementation uses:
# - p = 14 bits for bucket index (16384 buckets)
# - 5 bits per bucket (values 0-31 for leading zero count)
# - Total memory: ~16KB per HLL
# - Expected error: ~0.81%
#
# References:
# - Flajolet et al. "HyperLogLog: the analysis of a near-optimal cardinality estimation algorithm"
# - Google's HyperLogLog++ improvements

(def- hll-precision 14)
(def- hll-num-buckets (blshift 1 hll-precision))  # 16384 buckets
(def- hll-bucket-mask (- hll-num-buckets 1))      # 0x3FFF
(def- hll-hash-bits 32)
(def- hll-max-zeros (- hll-hash-bits hll-precision))  # 18

# Alpha constants for bias correction (from HyperLogLog paper)
# alpha_m = 1 / (m * integral_0^inf (log2((2+u)/(1+u)))^m du)
(def- hll-alpha-16 0.673)
(def- hll-alpha-32 0.697)
(def- hll-alpha-64 0.709)
(def- hll-alpha-large 0.7213)  # For m >= 128: alpha = 0.7213 / (1 + 1.079/m)

(defn- get-alpha
  "Get bias correction constant for given number of buckets"
  [m]
  (cond
    (= m 16) hll-alpha-16
    (= m 32) hll-alpha-32
    (= m 64) hll-alpha-64
    (/ hll-alpha-large (+ 1 (/ 1.079 m)))))

# Hash function for HLL using unsigned 64-bit integers
# Janet's int/u64 type handles full 32-bit unsigned values correctly

(defn- u64-mask32
  "Mask a u64 to 32 bits"
  [n]
  (band n (int/u64 0xFFFFFFFF)))

(defn- u64->number
  "Convert u64 to regular number (for use in bucket indexing)"
  [u]
  (int/to-number u))

(defn hash-value
  "Hash any value to 32-bit unsigned integer for HLL.
   Returns a regular Janet number in range [0, 2^32)."
  [val]
  (def s (if (string? val) val (string val)))
  # FNV-1a hash using u64 arithmetic
  (var hash (int/u64 0x811c9dc5))  # FNV offset basis
  (each byte s
    (set hash (bxor hash (int/u64 byte)))
    (set hash (u64-mask32 (* hash (int/u64 0x01000193)))))  # FNV prime

  # MurmurHash3 finalizer for better distribution
  (set hash (bxor hash (brshift hash 16)))
  (set hash (u64-mask32 (* hash (int/u64 0x85ebca6b))))
  (set hash (bxor hash (brshift hash 13)))
  (set hash (u64-mask32 (* hash (int/u64 0xc2b2ae35))))
  (set hash (bxor hash (brshift hash 16)))

  # Convert to regular number
  (u64->number hash))

(defn- count-leading-zeros
  "Count leading zeros in the lower (32-p) bits of a hash value.
   Uses arithmetic instead of bit operations for unsigned safety."
  [hash]
  # We only look at bits [0, hll-max-zeros) after extracting bucket index
  # mask = 2^18 - 1 = 262143
  (def mask (- (math/pow 2 hll-max-zeros) 1))
  (def value (math/floor (mod hash (+ mask 1))))
  (if (= value 0)
    hll-max-zeros
    (do
      (var zeros 0)
      (var v value)
      # Check each bit from MSB to LSB
      (while (and (< zeros hll-max-zeros)
                  (< v (math/pow 2 (- hll-max-zeros 1 zeros))))
        (++ zeros))
      zeros)))

(defn- rho
  "Calculate rho (position of leftmost 1-bit + 1) for HLL.
   This is equivalent to count-leading-zeros + 1"
  [hash]
  (+ 1 (count-leading-zeros hash)))

# ============================================================================
# Public API
# ============================================================================

(defn new
  "Create a new HyperLogLog sketch with default precision (p=14).
   Returns a table with :buckets array."
  []
  @{:buckets (array/new-filled hll-num-buckets 0)
    :precision hll-precision})

(defn add
  "Add a value to the HyperLogLog sketch.
   Mutates the sketch and returns it."
  [hll value]
  (def hash (hash-value value))
  # Use upper p bits for bucket index: hash >> (32-p) = hash / 2^(32-p)
  (def bucket-idx (math/floor (/ hash (math/pow 2 hll-max-zeros))))
  # Use lower (32-p) bits for leading zero count
  (def zeros (rho hash))
  (def buckets (hll :buckets))
  # Update bucket with max(current, zeros)
  (when (> zeros (get buckets bucket-idx))
    (put buckets bucket-idx zeros))
  hll)

(defn count-estimate
  "Estimate the cardinality (number of unique elements) in the sketch.
   Uses bias correction and small/large range corrections from HLL paper."
  [hll]
  (def buckets (hll :buckets))
  (def m hll-num-buckets)
  (def alpha (get-alpha m))

  # Calculate raw estimate: alpha * m^2 / sum(2^-bucket[i])
  (var sum 0)
  (var zero-buckets 0)
  (each v buckets
    (set sum (+ sum (math/pow 2 (- v))))
    (when (= v 0)
      (++ zero-buckets)))

  (def raw-estimate (* alpha m m (/ 1 sum)))

  # Apply small range correction (linear counting) if estimate is small
  # and there are empty buckets
  (if (and (<= raw-estimate (* 2.5 m)) (> zero-buckets 0))
    # Linear counting: m * ln(m / V) where V = number of empty buckets
    (* m (math/log (/ m zero-buckets)))
    # Large range correction for 32-bit hash (if estimate > 2^32 / 30)
    (if (> raw-estimate (/ 0x100000000 30))
      (* -1 0x100000000 (math/log (- 1 (/ raw-estimate 0x100000000))))
      raw-estimate)))

(defn merge
  "Merge two HyperLogLog sketches into a new sketch.
   Takes element-wise maximum of buckets.
   Both sketches must have the same precision."
  [hll1 hll2]
  (assert (= (hll1 :precision) (hll2 :precision))
          "Cannot merge HLLs with different precisions")
  (def result (new))
  (def b1 (hll1 :buckets))
  (def b2 (hll2 :buckets))
  (def br (result :buckets))
  (for i 0 hll-num-buckets
    (put br i (max (get b1 i) (get b2 i))))
  result)

(defn serialize
  "Serialize HLL to a compact buffer for storage.
   Each bucket is 5 bits, packed into bytes.
   Returns a buffer."
  [hll]
  (def buckets (hll :buckets))
  # Pack 5-bit values: 8 buckets fit into 5 bytes (40 bits)
  # Total: 16384 buckets * 5 bits / 8 = 10240 bytes
  (def packed-size (math/ceil (/ (* hll-num-buckets 5) 8)))
  (def buf @"")
  (var bit-pos 0)
  (var current-byte 0)

  (each v buckets
    # Write 5 bits of v into the buffer
    (def v5 (band v 0x1F))  # Ensure 5 bits
    (def bits-remaining (- 8 (% bit-pos 8)))

    (if (>= bits-remaining 5)
      # All 5 bits fit in current byte
      (do
        (set current-byte (bor current-byte (blshift v5 (- bits-remaining 5))))
        (set bit-pos (+ bit-pos 5))
        (when (= (% bit-pos 8) 0)
          (buffer/push-byte buf current-byte)
          (set current-byte 0)))
      # Need to split across bytes
      (do
        # Write upper bits to current byte
        (set current-byte (bor current-byte (brshift v5 (- 5 bits-remaining))))
        (buffer/push-byte buf current-byte)
        # Write lower bits to next byte
        (set current-byte (blshift (band v5 (- (blshift 1 (- 5 bits-remaining)) 1))
                                   (+ 8 bits-remaining (- 5))))
        (set bit-pos (+ bit-pos 5)))))

  # Flush any remaining bits
  (when (not= (% bit-pos 8) 0)
    (buffer/push-byte buf current-byte))

  buf)

(defn deserialize
  "Deserialize an HLL from a packed buffer.
   Returns a new HLL sketch."
  [buf]
  (def hll (new))
  (def buckets (hll :buckets))
  (var bit-pos 0)
  (var bucket-idx 0)

  (while (< bucket-idx hll-num-buckets)
    (def byte-pos (math/floor (/ bit-pos 8)))
    (def bit-offset (% bit-pos 8))
    (def bits-in-first-byte (- 8 bit-offset))

    (var value 0)
    (if (>= bits-in-first-byte 5)
      # All 5 bits in one byte
      (set value (band (brshift (get buf byte-pos) (- bits-in-first-byte 5)) 0x1F))
      # Split across two bytes
      (do
        (def first-bits (band (get buf byte-pos) (- (blshift 1 bits-in-first-byte) 1)))
        (def second-bits (brshift (get buf (+ byte-pos 1)) (+ 8 bits-in-first-byte (- 5))))
        (set value (bor (blshift first-bits (- 5 bits-in-first-byte)) second-bits))))

    (put buckets bucket-idx (band value 0x1F))
    (set bit-pos (+ bit-pos 5))
    (++ bucket-idx))

  hll)

(defn to-base64
  "Serialize HLL to base64 string for JSON storage"
  [hll]
  # Simple base64 encoding
  (def alphabet "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
  (def buf (serialize hll))
  (def result @"")

  (var i 0)
  (while (< i (length buf))
    (def b0 (get buf i 0))
    (def b1 (get buf (+ i 1) 0))
    (def b2 (get buf (+ i 2) 0))

    # Encode 3 bytes as 4 base64 chars
    (buffer/push-byte result (get alphabet (brshift b0 2)))
    (buffer/push-byte result (get alphabet (bor (blshift (band b0 0x03) 4)
                                                 (brshift b1 4))))
    (if (< (+ i 1) (length buf))
      (buffer/push-byte result (get alphabet (bor (blshift (band b1 0x0F) 2)
                                                   (brshift b2 6))))
      (buffer/push-byte result (chr "=")))
    (if (< (+ i 2) (length buf))
      (buffer/push-byte result (get alphabet (band b2 0x3F)))
      (buffer/push-byte result (chr "=")))

    (set i (+ i 3)))

  (string result))

(defn from-base64
  "Deserialize HLL from base64 string"
  [s]
  (def alphabet "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
  (def decode-table @{})
  (for i 0 64
    (put decode-table (get alphabet i) i))

  (def buf @"")
  (var i 0)
  (while (< i (length s))
    (def c0 (get s i))
    (def c1 (get s (+ i 1)))
    (def c2 (get s (+ i 2)))
    (def c3 (get s (+ i 3)))

    (def v0 (get decode-table c0 0))
    (def v1 (get decode-table c1 0))
    (def v2 (if (= c2 (chr "=")) 0 (get decode-table c2 0)))
    (def v3 (if (= c3 (chr "=")) 0 (get decode-table c3 0)))

    (buffer/push-byte buf (bor (blshift v0 2) (brshift v1 4)))
    (when (not= c2 (chr "="))
      (buffer/push-byte buf (bor (blshift (band v1 0x0F) 4) (brshift v2 2))))
    (when (not= c3 (chr "="))
      (buffer/push-byte buf (bor (blshift (band v2 0x03) 6) v3)))

    (set i (+ i 4)))

  (deserialize buf))

(defn empty?
  "Check if an HLL sketch is empty (no elements added)"
  [hll]
  (def buckets (hll :buckets))
  (var all-zero true)
  (each v buckets
    (when (not= v 0)
      (set all-zero false)
      (break)))
  all-zero)

(defn clear
  "Reset an HLL sketch to empty state. Mutates and returns the sketch."
  [hll]
  (def buckets (hll :buckets))
  (for i 0 (length buckets)
    (put buckets i 0))
  hll)
