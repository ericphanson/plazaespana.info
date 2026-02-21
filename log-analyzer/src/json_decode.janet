# Minimal JSON decoder for log-analyzer report mode.
# Supports objects, arrays, strings, numbers, booleans, and null.

(defn- ws-byte?
  [b]
  (or (= b (chr " "))
      (= b (chr "\n"))
      (= b (chr "\r"))
      (= b (chr "\t"))))

(defn- digit-byte?
  [b]
  (and (>= b (chr "0")) (<= b (chr "9"))))

(defn- hex-byte-value
  [b]
  (cond
    (and (>= b (chr "0")) (<= b (chr "9"))) (- b (chr "0"))
    (and (>= b (chr "a")) (<= b (chr "f"))) (+ 10 (- b (chr "a")))
    (and (>= b (chr "A")) (<= b (chr "F"))) (+ 10 (- b (chr "A")))
    nil))

(defn decode
  "Decode a JSON string into Janet values."
  [s]
  (def n (length s))
  (var i 0)

  (defn- fail [msg]
    (error (string "JSON decode error at index " i ": " msg)))

  (defn- peek []
    (if (< i n) (get s i) nil))

  (defn- advance []
    (def c (peek))
    (++ i)
    c)

  (defn- skip-ws []
    (while (and (< i n) (ws-byte? (get s i)))
      (++ i)))

  (defn- parse-literal [lit value]
    (def end (+ i (length lit)))
    (if (or (> end n)
            (not= (string/slice s i end) lit))
      (fail (string "expected literal " lit))
      (do
        (set i end)
        value)))

  (var parse-value nil)

  (defn- parse-number []
    (def start i)

    (when (= (peek) (chr "-"))
      (advance))

    (def int-start i)
    (while (and (< i n) (digit-byte? (peek)))
      (advance))
    (when (= i int-start)
      (fail "expected digit"))

    (when (and (< i n) (= (peek) (chr ".")))
      (advance)
      (def frac-start i)
      (while (and (< i n) (digit-byte? (peek)))
        (advance))
      (when (= i frac-start)
        (fail "expected digit after decimal point")))

    (when (and (< i n) (or (= (peek) (chr "e")) (= (peek) (chr "E"))))
      (advance)
      (when (and (< i n) (or (= (peek) (chr "+")) (= (peek) (chr "-"))))
        (advance))
      (def exp-start i)
      (while (and (< i n) (digit-byte? (peek)))
        (advance))
      (when (= i exp-start)
        (fail "expected exponent digits")))

    (def tok (string/slice s start i))
    (or (scan-number tok)
        (fail (string "invalid number " tok))))

  (defn- parse-string []
    (when (not= (advance) (chr "\""))
      (fail "expected opening quote"))
    (def buf @"")

    (while (< i n)
      (def c (advance))
      (cond
        (= c (chr "\""))
        (break)

        (= c (chr "\\"))
        (do
          (when (>= i n)
            (fail "unexpected end after escape"))
          (def esc (advance))
          (case esc
            (chr "\"") (buffer/push-byte buf (chr "\""))
            (chr "\\") (buffer/push-byte buf (chr "\\"))
            (chr "/") (buffer/push-byte buf (chr "/"))
            (chr "b") (buffer/push-byte buf 8)
            (chr "f") (buffer/push-byte buf 12)
            (chr "n") (buffer/push-byte buf (chr "\n"))
            (chr "r") (buffer/push-byte buf (chr "\r"))
            (chr "t") (buffer/push-byte buf (chr "\t"))
            (chr "u")
            (do
              (when (> (+ i 4) n)
                (fail "incomplete unicode escape"))
              (def h0 (hex-byte-value (get s i)))
              (def h1 (hex-byte-value (get s (+ i 1))))
              (def h2 (hex-byte-value (get s (+ i 2))))
              (def h3 (hex-byte-value (get s (+ i 3))))
              (when (or (nil? h0) (nil? h1) (nil? h2) (nil? h3))
                (fail "invalid unicode escape"))
              (def codepoint (+ (blshift h0 12) (blshift h1 8) (blshift h2 4) h3))
              (set i (+ i 4))
              # Keep decoder simple; JSON keys/values here are expected to be ASCII.
              (if (< codepoint 256)
                (buffer/push-byte buf codepoint)
                (buffer/push-byte buf (chr "?"))))
            (fail (string "invalid escape sequence byte " esc))))

        true
        (buffer/push-byte buf c)))

    (string buf))

  (defn- parse-array []
    (when (not= (advance) (chr "["))
      (fail "expected ["))
    (skip-ws)
    (def arr @[])
    (if (= (peek) (chr "]"))
      (do
        (advance)
        arr)
      (do
        (while true
          (array/push arr (parse-value))
          (skip-ws)
          (def c (peek))
          (cond
            (= c (chr ","))
            (do
              (advance)
              (skip-ws))
            (= c (chr "]"))
            (do
              (advance)
              (break))
            true
            (fail "expected , or ] in array")))
        arr)))

  (defn- parse-object []
    (when (not= (advance) (chr "{"))
      (fail "expected {"))
    (skip-ws)
    (def obj @{})
    (if (= (peek) (chr "}"))
      (do
        (advance)
        obj)
      (do
        (while true
          (when (not= (peek) (chr "\""))
            (fail "expected object key string"))
          (def key (parse-string))
          (skip-ws)
          (when (not= (peek) (chr ":"))
            (fail "expected : after object key"))
          (advance)
          (skip-ws)
          (put obj (keyword key) (parse-value))
          (skip-ws)
          (def c (peek))
          (cond
            (= c (chr ","))
            (do
              (advance)
              (skip-ws))
            (= c (chr "}"))
            (do
              (advance)
              (break))
            true
            (fail "expected , or } in object")))
        obj)))

  (set parse-value
       (fn []
         (skip-ws)
         (def c (peek))
         (when (nil? c)
           (fail "unexpected end of input"))
         (cond
           (= c (chr "{")) (parse-object)
           (= c (chr "[")) (parse-array)
           (= c (chr "\"")) (parse-string)
           (= c (chr "t")) (parse-literal "true" true)
           (= c (chr "f")) (parse-literal "false" false)
           (= c (chr "n")) (parse-literal "null" nil)
           (or (= c (chr "-")) (digit-byte? c)) (parse-number)
           true (fail (string "unexpected character byte " c)))))

  (def result (parse-value))
  (skip-ws)
  (when (< i n)
    (fail "trailing input"))
  result)
