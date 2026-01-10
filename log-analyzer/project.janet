(declare-project
  :name "log-analyzer"
  :description "Apache access log analyzer - counts unique visitors per month"
  :version "1.0.0"
  :dependencies [])

(declare-executable
  :name "log-analyzer"
  :entry "src/main.janet"
  :install false)
