# Add a cross-stack feature

1. Extend `contracts/openapi.yaml` and regenerate web DTOs.
2. Add domain invariants and tests in `apps/api/internal/domain`.
3. Define narrow ports and a command/query service boundary.
4. Implement the PostgreSQL adapter, migration, and isolated integration test.
5. Add explicit HTTP translation and handler tests.
6. Add a web feature with its own runtime schema, service adapter, query ownership, components, and public `index.ts`.
7. Compose the feature from a lazy route. Keep local workflow state out of the server cache.
8. Prove the critical path in the real system test.
9. Run `make check` and `make test-system`.

Do not create a generic shared abstraction until two implemented consumers demonstrate the same stable concept.
