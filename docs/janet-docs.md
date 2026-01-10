# Janet Language Syntax and Grammar Reference

## Introduction

Janet is a functional, dynamic programming language with Lisp-inspired syntax. It features built-in concurrency, PEG-based text processing, C FFI, and compiles to statically-linked native binaries. Janet uses S-expressions (symbolic expressions) for all code and data.

## Basic Syntax

### Comments

Comments start with `#` and continue to end of line:

```janet
# This is a comment
(def x 42)  # Inline comment
```

### S-Expressions

All Janet code consists of S-expressions—either atoms (literals, symbols, keywords) or lists enclosed in parentheses:

```janet
42                    # Atom: number literal
:keyword              # Atom: keyword
(+ 1 2)              # List: function call
(def name "value")   # List: special form
```

## Literals and Constants

### Booleans and Nil

```janet
true     # Boolean true
false    # Boolean false
nil      # Null/nothing value
```

### Numbers

Janet uses IEEE 754 floating point. Numbers support multiple formats:

```janet
# Decimal
42
3.14159
1_000_000        # Underscores for readability

# Scientific notation
1.3e18
6.022e23

# Hexadecimal
0xFF
0x1A2B

# Arbitrary radix (2-36)
2r1010           # Binary: 10
16rDEADBEEF      # Hex: 3735928559
36rZZZ           # Base-36: 46655
```

### Strings

Strings use double quotes with escape sequences:

```janet
"Hello, World!"
"Line 1\nLine 2"
"Tab\there"
"Unicode: \u03B1\u03B2\u03B3"
"Hex byte: \xFF"
```

Long strings use backticks with no escape processing:

```janet
``Raw string with \n literal backslash``
``Can span
multiple
lines``
```

### Buffers

Mutable strings prefixed with `@`:

```janet
@"mutable string"
@``mutable long string``
```

### Symbols

Identifiers that can include alphanumerics, Unicode, and special characters `!@$%^&*-_+=:<>.?`:

```janet
my-var
set!
valid?
some-module/function
kebab-case-name
```

### Keywords

Constants beginning with `:`, primarily used as keys:

```janet
:name
:age
:first-name
```

## Data Structures

### Tuples (Immutable)

Ordered sequences in parentheses `()` or brackets `[]`:

```janet
'(1 2 3)          # Quoted tuple
[1 2 3]           # Bracket notation (preferred, no quote needed)
(tuple 1 2 3)     # Explicit construction
```

### Arrays (Mutable)

Mutable tuples prefixed with `@`:

```janet
@(1 2 3)
@[1 2 3]
(array 1 2 3)    # Explicit construction
```

Array operations:

```janet
(def arr @[1 2 3])
(array/push arr 4)        # arr is now @[1 2 3 4]
(array/pop arr)           # Returns 4, arr is @[1 2 3]
(array/concat arr @[5 6]) # Mutates arr to @[1 2 3 5 6] and returns it
```

### Structs (Immutable)

Hash tables in braces `{}`:

```janet
{:name "Alice" :age 30}
{:x 10 :y 20}
{1 "one" 2 "two"}         # Non-keyword keys allowed
```

### Tables (Mutable)

Mutable structs prefixed with `@`:

```janet
@{:name "Bob" :age 25}
@{:x 0 :y 0}
```

Table operations:

```janet
(def tbl @{:a 1})
(put tbl :b 2)            # tbl is now @{:a 1 :b 2}
(get tbl :a)              # Returns 1
(in tbl :c)               # Returns nil
```

### Accessing Data Structures

```janet
(def arr @[10 20 30])
(get arr 0)               # Returns 10
(arr 1)                   # Returns 20 (shorthand)

(def tbl @{:x 100})
(get tbl :x)              # Returns 100
(tbl :x)                  # Returns 100 (shorthand)
(get tbl :y 0)            # Returns 0 (default value)
```

### Equality Testing

**Important:** Arrays and tables use identity equality with `=`, not structural equality. Use `deep=` for value comparison:

```janet
# Tuples and structs compare by value
(= [1 2 3] [1 2 3])           # true
(= {:a 1} {:a 1})             # true

# Arrays and tables compare by identity
(= @[1 2 3] @[1 2 3])         # false (different objects)
(= @{:a 1} @{:a 1})           # false (different objects)

# Use deep= for structural equality
(deep= @[1 2 3] @[1 2 3])     # true
(deep= @{:a 1} @{:a 1})       # true
```

## Functions

### Defining Functions

Using `defn` (most common):

```janet
(defn add
  "Adds two numbers."
  [x y]
  (+ x y))

(add 5 3)  # Returns 8
```

### Anonymous Functions

Using `fn`:

```janet
((fn [x] (* x x)) 5)  # Returns 25

(def square (fn [x] (* x x)))
(square 4)  # Returns 16
```

Using short function syntax `|`:

```janet
(|(* $ $) 5)              # Returns 25, $ is first arg
(|(+ $0 $1 $2) 1 2 3)    # Returns 6, $0/$1/$2 are positional
(|$& 1 2 3)              # Returns (1 2 3), $& is all args tuple
```

### Function Parameters

**Optional parameters** with `&opt`:

```janet
(defn greet [name &opt title]
  (if title
    (string title " " name)
    name))

(greet "Alice")              # "Alice"
(greet "Bob" "Dr.")          # "Dr. Bob"
```

**Variadic functions** with `&`:

```janet
(defn sum [& numbers]
  (+ ;numbers))  # Splice operator ;

(sum 1 2 3 4)  # Returns 10
```

**Keyword arguments** with `&keys`:

```janet
(defn make-person [name &keys {:age age :city city}]
  {:name name :age age :city city})

(make-person "Alice" :age 30 :city "NYC")
```

**Named arguments** with `&named` (Janet 1.23.0+):

```janet
(defn configure [&named debug verbose]
  (print "Debug: " debug ", Verbose: " verbose))

(configure :debug true :verbose false)
```

## Special Forms

Special forms are built-in language constructs that cannot be implemented as functions or macros due to special evaluation rules.

### Variable Binding

**`def`** - Immutable binding:

```janet
(def pi 3.14159)
(def name "Alice")
(def [x y] [10 20])  # Destructuring
```

**`var`** - Mutable binding:

```janet
(var counter 0)
(set counter (+ counter 1))
```

### Functions and Scope

**`fn`** - Create function:

```janet
(def double (fn [x] (* 2 x)))
```

**`do`** - Sequential execution with new scope:

```janet
(do
  (def x 10)
  (def y 20)
  (+ x y))  # Returns 30
```

**`upscope`** - Sequential execution without new scope:

```janet
(upscope
  (def z 100))
z  # z is accessible here
```

### Control Flow

**`if`** - Conditional branching:

```janet
(if (< x 0)
  "negative"
  "non-negative")

(if condition
  then-branch)  # No else branch
```

Only `nil` and `false` are falsy; everything else (including `0` and `""`) is truthy.

**`while`** - Loop while condition is true:

```janet
(var i 0)
(while (< i 5)
  (print i)
  (set i (+ i 1)))
```

**`break`** - Exit loop or return from function:

```janet
(while true
  (if (> x 10)
    (break x)))  # Returns x and exits loop
```

### Quoting

**`quote`** (shorthand `'`) - Return literal without evaluation:

```janet
(quote (+ 1 2))      # Returns the list (+ 1 2), not 3
'(+ 1 2)             # Same as above
```

**`quasiquote`** (shorthand `~`) - Quote with selective unquoting:

```janet
(def x 42)
~(the answer is ,x)  # Returns (the answer is 42)
```

**`unquote`** (shorthand `,`) - Evaluate within quasiquote:

```janet
(def y 10)
~(add ,(+ 3 y) 5)    # Returns (add 13 5)
```

**`splice`** (shorthand `;`) - Insert array/tuple contents inline:

```janet
(def args [1 2 3])
(+ ;args)            # Equivalent to (+ 1 2 3)
(array 0 ;args 4)    # Returns @[0 1 2 3 4]
```

### Assignment

**`set`** - Update mutable binding or data structure:

```janet
(var x 10)
(set x 20)

(def tbl @{:a 1})
(set (tbl :a) 100)   # Update table value
```

## Control Flow and Iteration

### Conditional Expressions

**`cond`** - Multiple conditions (macro):

```janet
(cond
  (< x 0) "negative"
  (> x 0) "positive"
  "zero")
```

**`when`** - Single-branch conditional (macro):

```janet
(when (> x 10)
  (print "x is large")
  (do-something x))
```

**`case`** - Pattern matching (macro):

```janet
(case value
  :red "stop"
  :yellow "slow"
  :green "go"
  "unknown")
```

### Loops

**`for`** - Iteration over range (macro):

```janet
(for i 0 10
  (print i))  # Prints 0 through 9
```

**`each`** - Iteration over collection (macro):

```janet
(each item [1 2 3 4]
  (print item))
```

**`map`** - Transform collection:

```janet
(map |(* $ $) [1 2 3 4])  # Returns @[1 4 9 16]
```

**`reduce`** - Accumulate values:

```janet
(reduce + 0 [1 2 3 4])  # Returns 10
```

**`filter`** - Select matching elements:

```janet
(filter |(> $ 5) [3 7 2 9 4])  # Returns @[7 9]
```

## PEG (Parsing Expression Grammars)

Janet includes powerful built-in pattern matching through PEGs, which are more powerful than regular expressions and easier than custom parsers.

### Basic PEG Matching

**`peg/match`** - Match pattern against text:

```janet
(peg/match "cat" "cat")           # Returns @[]
(peg/match "cat" "dog")           # Returns nil
(peg/match '(* "c" "at") "cat")  # Returns @[]
```

### Primitive Patterns

```janet
# Exact string
(peg/match "hello" "hello world")  # Match

# Character count
(peg/match 3 "abc")                # Match exactly 3 chars
(peg/match -1 "")                  # Match if at least 1 char missing

# Character ranges
(peg/match '(range "az") "m")     # Match lowercase letter
(peg/match '(range "AZ") "M")     # Match uppercase letter

# Character set
(peg/match '(set "aeiou") "e")    # Match any vowel
```

### Pattern Combinators

```janet
# Sequence (* or :*)
(peg/match '(* "hello" " " "world") "hello world")

# Choice (+ or :+)
(peg/match '(+ "cat" "dog") "dog")  # Try each option

# Any/some (0+ or 1+)
(peg/match '(any "a") "aaaa")       # Match 0 or more 'a'
(peg/match '(some "a") "aaaa")      # Match 1 or more 'a'

# Optional (? or :?)
(peg/match '(? "s") "s")            # Match 0 or 1 's'
(peg/match '(? "s") "")             # Also matches

# Repetition
(peg/match '(between 2 4 "a") "aaa")    # 2-4 repetitions
(peg/match '(at-least 3 "b") "bbbb")    # At least 3
(peg/match '(at-most 2 "c") "cc")       # At most 2
(peg/match '(repeat 3 "x") "xxx")       # Exactly 3
```

### Captures

```janet
# Capture matched text
(peg/match '(capture "cat") "cat")        # Returns @["cat"]
(peg/match '(<- "dog") "dog")             # Same, shorthand

# Capture groups
(peg/match '(group (capture "a") (capture "b")) "ab")  # @[@["a" "b"]]

# Position capture
(peg/match '(* "hello" (position)) "hello")  # @[5]
(peg/match '($) "")                          # Shorthand

# Constant capture
(peg/match '(constant :success) "anything")  # @[:success]

# Replace/substitute
(peg/match '(/ "cat" "feline") "cat")       # @["feline"]
(peg/match '(/ (<- "dog") |("canine")) "dog")  # @["canine"]
```

### Built-in Aliases

```janet
:d   # Digit [0-9]
:a   # Letter [a-zA-Z]
:w   # Word character [a-zA-Z0-9]
:s   # Whitespace [ \t\n\r]
:h   # Hex digit [0-9a-fA-F]

:D   # Non-digit
:A   # Non-letter
:W   # Non-word character
:S   # Non-whitespace
:H   # Non-hex

:d+  # One or more digits
:d*  # Zero or more digits
# Similar for :a, :w, :s, :h
```

### Grammars

Grammars enable recursive patterns using keyword-keyed structs:

```janet
(def json-grammar
  '{:main :value
    :value (+ :null :bool :number :string :array :object)
    :null (* "null" (constant nil))
    :bool (+ (* "true" (constant true)) (* "false" (constant false)))
    :number (<- (some :d))
    :string (* "\"" (<- (any (if-not "\"" 1))) "\"")
    :array (* "[" (any (* :value (? (* "," :value)))) "]")
    :object (* "{" (any (* :string ":" :value (? (* "," :string ":" :value)))) "}")})

(peg/match json-grammar `{"name": "Alice", "age": 30}`)
```

Every grammar requires `:main` as the entry point.

### Practical PEG Examples

**Email validation:**

```janet
(def email-peg
  ~{:main (* :name "@" :domain)
    :name (some (+ :a :d (set ".-_")))
    :domain (some (+ :a :d (set ".-")))})

(peg/match email-peg "user@example.com")  # Match
```

**Parse key-value pairs:**

```janet
(def kv-peg
  ~{:main (some (* :s* (group (* :key "=" :value)) :s*))
    :key (<- (some :a))
    :value (<- (some (if-not (set "=\n ") 1)))
    :s* (any :s)})

(peg/match kv-peg "name=Alice age=30")
# Returns @[@["name" "Alice"] @["age" "30"]]
```

## Macros and Metaprogramming

### Defining Macros

Macros transform code at compile time:

```janet
(defmacro unless [condition & body]
  ~(if (not ,condition)
     (do ,;body)))

(unless false
  (print "This runs"))  # Expands to (if (not false) (do (print "This runs")))
```

### Macro Expansion

**`macex`** - Expand macro once:

```janet
(macex '(unless false (print "hi")))
# Returns: (if (not false) (do (print "hi")))
```

**`macex1`** - Expand only outermost macro.

### Compile and Eval

**`eval`** - Evaluate code:

```janet
(eval '(+ 1 2))  # Returns 3
```

**`compile`** - Compile Janet code to bytecode:

```janet
(compile '(defn double [x] (* 2 x)))
```

## Module System

### Importing Modules

**`import`** - Load module with prefix:

```janet
(import json)
(json/encode {:a 1})
```

**`require`** - Load module without prefix:

```janet
(require "mymodule")
```

**`use`** - Import all bindings into current namespace:

```janet
(use ./utils)  # Not recommended except for REPL
```

### Creating Modules

```janet
# mymodule.janet
(defn public-fn [] :public)
(def- private-var 42)  # Private (not exported)

# Export selected bindings
(def module/public-fn public-fn)
```

## Concurrency

### Fibers

Fibers are lightweight coroutines:

```janet
(def f (fiber/new
  (fn []
    (yield 1)
    (yield 2)
    3)))

(resume f)  # Returns 1
(resume f)  # Returns 2
(resume f)  # Returns 3
```

### Event Loop

**`ev/go`** - Spawn concurrent task:

```janet
(ev/go
  (fn []
    (print "Running in background")))
```

**`ev/sleep`** - Async sleep:

```janet
(ev/sleep 1)  # Sleep for 1 second
```

## Error Handling

### Try/Catch

```janet
(try
  (error "something failed")
  ([err]
   (print "Caught error: " err)))
```

### Error Function

```janet
(error "Custom error message")
```

## Common Built-in Functions

### String Operations

```janet
(string "hello" " " "world")    # "hello world"
(string/split "," "a,b,c")      # @["a" "b" "c"]
(string/join ["a" "b"] "-")     # "a-b"
(string/trim "  hello  ")       # "hello"
```

### Array/Tuple Operations

```janet
(length [1 2 3])                # 3
(first [1 2 3])                 # 1
(last [1 2 3])                  # 3
(slice [1 2 3 4] 1 3)          # @[2 3]
```

### Mathematical Operations

```janet
(+ 1 2 3)                       # 6
(- 10 3)                        # 7
(* 2 3 4)                       # 24
(/ 10 2)                        # 5
(% 10 3)                        # 1 (modulo)
(math/pow 2 8)                  # 256
(math/sqrt 16)                  # 4
```

### Type Checking

```janet
(type 42)                       # :number
(type "hello")                  # :string
(type @[])                      # :array
(type nil)                      # :nil
(int? 42)                       # true
(string? "hi")                  # true
(indexed? [1 2])                # true
```

### I/O Operations

```janet
(print "Hello")                 # Print with newline
(prin "No newline")             # Print without newline
(pp {:a 1})                     # Pretty print

(def f (file/open "file.txt" :r))
(file/read f :all)              # Read entire file
(file/close f)

(spit "out.txt" "content")      # Write to file
(slurp "in.txt")                # Read file to string
```

## Example Programs

### FizzBuzz

```janet
(defn fizzbuzz [n]
  (for i 1 (+ n 1)
    (cond
      (zero? (% i 15)) (print "FizzBuzz")
      (zero? (% i 3)) (print "Fizz")
      (zero? (% i 5)) (print "Buzz")
      (print i))))

(fizzbuzz 100)
```

### Factorial

```janet
# Recursive
(defn factorial [n]
  (if (<= n 1)
    1
    (* n (factorial (- n 1)))))

# Tail recursive
(defn factorial-tail [n &opt acc]
  (default acc 1)
  (if (<= n 1)
    acc
    (factorial-tail (- n 1) (* n acc))))

(factorial 5)       # 120
(factorial-tail 5)  # 120
```

### Simple Web Server (requires spork)

```janet
(import spork/http)

(defn handler [request]
  {:status 200
   :headers {"Content-Type" "text/html"}
   :body "<h1>Hello from Janet!</h1>"})

(http/server handler 8000)
```

### File Processing

```janet
(defn count-lines [filename]
  (with [f (file/open filename :r)]
    (length (string/split "\n" (file/read f :all)))))

(count-lines "myfile.txt")
```

### Simple DSL with Macros

```janet
(defmacro html [& body]
  ~(string "<html>" ,;body "</html>"))

(defmacro tag [name & content]
  ~(string "<" ,name ">" ,;content "</" ,name ">"))

(html
  (tag "h1" "Welcome")
  (tag "p" "This is a paragraph"))
# Returns: "<html><h1>Welcome</h1><p>This is a paragraph</p></html>"
```

## Additional Resources

- Official documentation: https://janet-lang.org/docs/
- Janet Guide ("Janet for Mortals"): https://janet.guide/
- API Reference: https://janet-lang.org/api/
- Source code: https://github.com/janet-lang/janet
