// Package lazyjobs provides a small background job runner contract.
//
// Applications define typed jobs, register them with a JobRunner, and enqueue
// JSON-backed payloads. The runner owns worker lifecycle, retries, state
// transitions, and read-only inspection. Storage is delegated to a Backend so
// applications can choose in-memory, PostgreSQL, or another durable backend.
package lazyjobs
