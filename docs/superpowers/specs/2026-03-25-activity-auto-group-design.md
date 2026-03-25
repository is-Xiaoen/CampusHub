# Activity Auto Group Schema Self-Heal Design

## Background

Users expect a group chat to be created automatically after an activity is published.

The activity creation flow already publishes `activity.created`, and chat RPC already subscribes to that event and calls `CreateGroup`. The breakage is in the chat persistence layer: the `Group` model writes `cover_url`, but the repository bootstrap SQL for `groups` does not define that column. In environments initialized from the current SQL, `CreateGroup` fails during insert, so the automatic group is never created.

## Goal

Restore automatic group creation for newly created or newly submitted activities, and make existing chat databases self-heal on service startup without introducing a broad migration framework.

## Non-Goals

- No redesign of the event flow between activity and chat services
- No generic schema migration system
- No change to activity publication semantics

## Chosen Approach

Implement a startup-time schema check in chat RPC that verifies `groups.cover_url` exists. If the column is missing, execute one targeted `ALTER TABLE` to add it with the same definition used by the GORM model.

Also update `deploy/sql/chat.sql` so fresh databases are initialized correctly.

## Alternatives Considered

### 1. Targeted startup self-heal

Recommended.

Pros:
- Fixes existing deployed databases
- Small blast radius
- Matches the exact confirmed root cause

Cons:
- Adds one metadata query at startup
- First startup after deploy may execute DDL

### 2. `AutoMigrate` on startup

Rejected.

Pros:
- Less custom code

Cons:
- May mutate unrelated schema details
- Harder to reason about in production

### 3. Runtime fallback in `CreateGroup`

Rejected.

Pros:
- Keeps startup path unchanged

Cons:
- Defers failure to request time
- Leaves database drift in place

## Design

### Chat RPC startup

Add a focused schema guard in `app/chat/rpc/internal/svc`:

1. Open MySQL connection as today
2. Run `ensureChatSchema(db)`
3. If `groups.cover_url` is missing, execute:

```sql
ALTER TABLE `groups`
ADD COLUMN `cover_url` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '封面图URL（活动封面）' AFTER `name`
```

4. Continue normal service initialization only after the schema is valid

### Failure handling

- Metadata query fails: fail startup
- `ALTER TABLE` fails: fail startup
- Column exists: continue startup, low-noise info log
- Column added successfully: continue startup, explicit info log

Failing startup is intentional. If the database cannot be validated or repaired, the MQ consumer would continue to fail automatic group creation anyway.

### SQL source of truth

Update `deploy/sql/chat.sql` so new environments create `groups.cover_url` from the start. The SQL definition must stay aligned with `app/chat/model/group.go`.

## Testing Strategy

Add focused unit tests for the schema guard:

1. Column already exists:
- metadata query reports `cover_url`
- no `ALTER TABLE` is executed

2. Column missing:
- metadata query reports no row
- one exact `ALTER TABLE groups ... cover_url ...` is executed

3. Metadata query error:
- function returns error

4. `ALTER TABLE` error:
- function returns error

Use `sqlmock` so tests are deterministic and do not depend on a live MySQL instance.

## Verification

- `go test ./app/chat/...`
- Static field alignment check across:
  - `app/chat/model/group.go`
  - `app/chat/rpc/internal/logic/creategrouplogic.go`
  - `deploy/sql/chat.sql`

## Risks

- Startup DDL may require privileges not present in some environments
- Concurrent startup of multiple chat instances could race on first repair

## Risk Mitigation

- Treat duplicate-column style outcomes as success if the column already exists after the attempt
- Keep the repair targeted to a single column to minimize lock time and schema drift

