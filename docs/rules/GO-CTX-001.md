# GO-CTX-001 — context propagation

HTTP request paths propagate `r.Context()` through services and adapters. Creating `context.Background()` or `context.TODO()` in transport detaches cancellation, deadlines, request IDs, and traces.

Process startup and bounded cleanup contexts belong in bootstrap. Tests may create root contexts at their entrypoint.

Run `make arch-explain RULE=GO-CTX-001` and `make arch`.
