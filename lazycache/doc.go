// Package lazycache provides a small application cache contract.
//
// lazycache owns key construction, the enabled/disabled switch, standardized
// statistics, and typed convenience helpers. It intentionally does not import a
// concrete backend. Conventional applications receive the default in-memory
// backend through lazyapp, while custom setups can pass any Backend to New.
package lazycache
