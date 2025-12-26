Below is the **updated final list**, with **`fieldvalues` renamed to `values`** everywhere for clarity and simplicity. This keeps vocabulary tight and intuitive, similar to monday and database/table metaphors.

---

## Store (1:1 with feature)

| File                  | Description                                                                                            |
| --------------------- | ------------------------------------------------------------------------------------------------------ |
| `store.go`            | DuckDB handle, schema migration runner (`schema.sql`), transaction helpers, shared DB utilities.       |
| `users_store.go`      | Persist users and sessions; user lookup, create, password update, session lifecycle.                   |
| `workspaces_store.go` | Workspace CRUD; workspace membership CRUD and role management.                                         |
| `teams_store.go`      | Team CRUD (team is required); team membership CRUD and role checks.                                    |
| `projects_store.go`   | Project (board) CRUD within a team; manage `issue_counter` allocation.                                 |
| `columns_store.go`    | Column CRUD per project; reorder columns; set default; archive/unarchive.                              |
| `cycles_store.go`     | Cycle CRUD per team; status transitions (planning/active/completed); date range updates.               |
| `issues_store.go`     | Issue (card) CRUD; move between columns; reorder; attach/detach cycles; key/number allocation support. |
| `assignees_store.go`  | Issue assignees add/remove/list (many-to-many).                                                        |
| `comments_store.go`   | Comment CRUD; list by issue; basic counts.                                                             |
| `fields_store.go`     | Field definitions per project (custom columns); reorder; archive; required flags.                      |
| `values_store.go`     | Typed field values per issue; set/get/list values; bulk load for board and list views.                 |

---

## `feature/*` (contracts)

| Name       | Description                                                          | Data Model (Go struct names only) |
| ---------- | -------------------------------------------------------------------- | --------------------------------- |
| users      | Accounts and sessions (authentication and identity).                 | `User`, `Session`                 |
| workspaces | Tenant boundary; workspace membership and roles.                     | `Workspace`, `WorkspaceMember`    |
| teams      | Required organizational unit; team membership and roles.             | `Team`, `TeamMember`              |
| projects   | Boards container under teams.                                        | `Project`                         |
| columns    | Kanban columns per project (ordering, default column, archive).      | `Column`                          |
| cycles     | Team-scoped planning periods (Linear-style cycles).                  | `Cycle`                           |
| issues     | Cards within a board; column positioning; optional cycle attachment. | `Issue`                           |
| assignees  | Many-to-many assignment between issues and users.                    | `IssueAssignee`                   |
| comments   | Markdown discussion on issues.                                       | `Comment`                         |
| fields     | Custom fields (monday/GitHub style) defined per project.             | `Field`                           |
| values     | Typed values for fields per issue (cells).                           | `Value`                           |

---

### Why `values` is the right name

* Matches how users think: **field + value**
* Short and universal
* Aligns with monday (cells), GitHub Projects (field values), databases
* Easier for AI agents and tooling to reason about
* Avoids the overly technical `field_values` naming

At this point, your naming is:

* **UI-simple** (columns, fields, values, cards)
* **Schema-stable**
* **Migration-friendly**
* **AI-ready**

If you want, next we can:

* Lock down the **exact Go struct definitions** for `Column`, `Field`, `Value`
* Or generate **feature/*/api.go** contracts consistently for all features
