# Activity Auto Group Schema Self-Heal Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore automatic group creation by repairing the missing `groups.cover_url` column in chat databases and keeping bootstrap SQL aligned.

**Architecture:** Add one startup-time schema guard in chat RPC service initialization. Keep the fix local to the chat service, cover it with focused unit tests, and update bootstrap SQL so new environments do not regress.

**Tech Stack:** Go, GORM, go-zero, MySQL, `sqlmock`, PowerShell, `go test`

---

### Task 1: Add failing tests for chat schema guard

**Files:**
- Create: `app/chat/rpc/internal/svc/schema_test.go`
- Modify: `go.mod` if `sqlmock` is not present

- [ ] **Step 1: Write the failing tests**

Cover:
- existing `cover_url` column does not trigger DDL
- missing `cover_url` triggers one `ALTER TABLE`
- metadata query error is returned
- alter error is returned

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/chat/rpc/internal/svc -run TestEnsureChatSchema`

Expected: FAIL because `ensureChatSchema` does not exist yet.

- [ ] **Step 3: Write minimal implementation**

Create the smallest `ensureChatSchema` helper needed for the tests to pass. Keep it local to chat RPC service initialization.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./app/chat/rpc/internal/svc -run TestEnsureChatSchema`

Expected: PASS

- [ ] **Step 5: Commit**

Run:

```bash
git add app/chat/rpc/internal/svc/schema_test.go app/chat/rpc/internal/svc/*.go go.mod go.sum
git commit -m "test: cover chat schema self-heal"
```

### Task 2: Wire schema self-heal into chat service startup

**Files:**
- Modify: `app/chat/rpc/internal/svc/servicecontext.go`
- Create: `app/chat/rpc/internal/svc/schema.go`

- [ ] **Step 1: Integrate the helper into DB initialization**

Call `ensureChatSchema(db)` immediately after the database connection is created and before models are used.

- [ ] **Step 2: Add explicit logs for each branch**

Log:
- column already present
- column added successfully
- validation or repair failure

- [ ] **Step 3: Run targeted tests**

Run: `go test ./app/chat/rpc/internal/svc`

Expected: PASS

- [ ] **Step 4: Run chat package regression tests**

Run: `go test ./app/chat/...`

Expected: PASS

- [ ] **Step 5: Commit**

Run:

```bash
git add app/chat/rpc/internal/svc/servicecontext.go app/chat/rpc/internal/svc/schema.go app/chat/rpc/internal/svc/schema_test.go
git commit -m "fix: self-heal chat group schema"
```

### Task 3: Align bootstrap SQL with runtime schema

**Files:**
- Modify: `deploy/sql/chat.sql`

- [ ] **Step 1: Update bootstrap SQL**

Add `cover_url` to the `groups` table definition in the same logical position and with the same type/default as the Go model.

- [ ] **Step 2: Verify field alignment**

Check:
- `app/chat/model/group.go`
- `app/chat/rpc/internal/logic/creategrouplogic.go`
- `deploy/sql/chat.sql`

- [ ] **Step 3: Run regression tests**

Run: `go test ./app/chat/...`

Expected: PASS

- [ ] **Step 4: Commit**

Run:

```bash
git add deploy/sql/chat.sql
git commit -m "fix: align chat bootstrap schema"
```

### Task 4: Final verification

**Files:**
- No new files

- [ ] **Step 1: Run final verification**

Run:

```bash
go test ./app/chat/...
```

Expected: PASS

- [ ] **Step 2: Summarize any remaining gaps**

Document that full-repo `go test ./...` timed out during baseline, so final verification is scoped to chat packages for this bugfix.

