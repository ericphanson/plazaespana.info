# Simple JSON encoder for Janet
#
# Converts Janet data structures to JSON strings

(defn- escape-string
  "Escape a string for JSON output"
  [s]
  (def buf @"\"")
  (each byte s
    (case byte
      (chr "\"") (buffer/push-string buf "\\\"")
      (chr "\\") (buffer/push-string buf "\\\\")
      (chr "\n") (buffer/push-string buf "\\n")
      (chr "\r") (buffer/push-string buf "\\r")
      (chr "\t") (buffer/push-string buf "\\t")
      # Control characters
      (if (< byte 32)
        (buffer/push-string buf (string/format "\\u%04x" byte))
        (buffer/push-byte buf byte))))
  (buffer/push-byte buf (chr "\""))
  (string buf))

(defn encode
  "Encode a Janet value as a JSON string.
   Supports: nil, booleans, numbers, strings, arrays/tuples, tables/structs"
  [val]
  (cond
    (nil? val) "null"
    (boolean? val) (if val "true" "false")
    (number? val) (if (= val (math/floor val))
                    (string/format "%d" val)
                    (string/format "%.15g" val))
    (string? val) (escape-string val)
    (buffer? val) (escape-string (string val))
    (keyword? val) (escape-string (string val))
    (symbol? val) (escape-string (string val))

    # Arrays and tuples become JSON arrays
    (or (array? val) (tuple? val))
    (string "[" (string/join (map encode val) ",") "]")

    # Tables and structs become JSON objects
    (or (table? val) (struct? val))
    (do
      (def pairs @[])
      (eachp [k v] val
        (def key-str (cond
                       (string? k) k
                       (keyword? k) (string k)
                       (symbol? k) (string k)
                       (string k)))
        (array/push pairs (string (escape-string key-str) ":" (encode v))))
      (string "{" (string/join pairs ",") "}"))

    # Fallback: convert to string
    (escape-string (string val))))

(defn encode-pretty
  "Encode a Janet value as a pretty-printed JSON string"
  [val &opt indent-str current-indent]
  (default indent-str "  ")
  (default current-indent "")
  (def next-indent (string current-indent indent-str))

  (cond
    (nil? val) "null"
    (boolean? val) (if val "true" "false")
    (number? val) (if (= val (math/floor val))
                    (string/format "%d" val)
                    (string/format "%.15g" val))
    (string? val) (escape-string val)
    (buffer? val) (escape-string (string val))
    (keyword? val) (escape-string (string val))
    (symbol? val) (escape-string (string val))

    # Arrays and tuples become JSON arrays
    (or (array? val) (tuple? val))
    (if (empty? val)
      "[]"
      (do
        (def items (map |(encode-pretty $ indent-str next-indent) val))
        (string "[\n" next-indent
                (string/join items (string ",\n" next-indent))
                "\n" current-indent "]")))

    # Tables and structs become JSON objects
    (or (table? val) (struct? val))
    (if (empty? val)
      "{}"
      (do
        (def pairs @[])
        # Sort keys for consistent output
        (def sorted-keys (sort (map string (keys val))))
        (each k sorted-keys
          (def orig-key (find |(= (string $) k) (keys val)))
          (def v (get val orig-key))
          (array/push pairs (string (escape-string k) ": "
                                    (encode-pretty v indent-str next-indent))))
        (string "{\n" next-indent
                (string/join pairs (string ",\n" next-indent))
                "\n" current-indent "}")))

    # Fallback: convert to string
    (escape-string (string val))))
