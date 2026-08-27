# GO-HTTP-001 — bounded HTTP bodies

JSON handlers call `response.DecodeJSON`. The shared decoder enforces content type, a configured byte limit, a single JSON value, and unknown-field rejection.

Direct `json.NewDecoder(r.Body)` and `io.ReadAll(r.Body)` in HTTP transport are forbidden. A genuinely streaming endpoint needs a documented byte/time bound and an ADR.

Run `make arch-explain RULE=GO-HTTP-001` and `make arch`.
