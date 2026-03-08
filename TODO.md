# Homelab Refactoring TODO

## 1. Pagination Refactoring (Cursor-based)
- [x] Phase 1: Core Models Definition
  - [x] Define `PaginationRequest` and `PaginationResponse[T]` in `pkg/models/pagination.go`.
  - [x] Implement `Bind` for `PaginationRequest`.
  - [x] Update `common.PaginatedResponse` to support `NextCursor`.
- [x] Phase 2: Repository Layer Refactor
  - [x] Refactor `audit/repo.go` to use `ScanLogs` and cross-month cursors.
  - [x] Implement `ScanXXX` in all repositories using `db.ListCurrentCursor`.
- [x] Phase 3: Service Layer Integration
  - [x] Update `discovery.LookupFunc` signature to `(cursor, limit)`.
  - [x] Add `ScanXXX` proxy methods with permission checks in services.
- [x] Phase 4: Controller Layer Update
  - [x] Replace `getPaginationParams` with `getCursorParams`.
  - [x] Update handlers to use `common.CursorSuccess`.
- [x] Phase 5: Outdated Code Cleanup
  - [x] Remove all `ListXXX(page, pageSize)` methods from Repositories.
  - [x] Refactor remaining Service methods to use `Scan` instead of `List`.
  - [x] Clean up `getPaginationParams` and `PaginatedSuccess` in `pkg/controllers/utils.go` and `pkg/common/common.go`.
- [x] Phase 6: API Documentation & Client SDK
  - [x] Update Swagger/OpenAPI annotations across all controllers.
  - [x] Run `make backend-gen` to sync `client-go` and frontend models.
- [x] Phase 7: Frontend Migration
  - [x] Update Angular services to track `nextCursor`.
  - [x] Modify UI tables to support "Load More" or "Next Page" via cursors.
- [x] Phase 8: Validation
  - [x] Verify no `page`/`pageSize` parameters remain in the entire `backend/` codebase.
