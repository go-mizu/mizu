package runtime

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
	"github.com/go-mizu/blueprints/localflare/store/sqlite"
)

// d1TestHelper creates a test runtime with D1 store
type d1TestHelper struct {
	rt         *Runtime
	store      store.Store
	databaseID string
	cleanup    func()
}

func newD1TestHelper(t *testing.T) *d1TestHelper {
	t.Helper()

	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "d1test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create SQLite store
	s, err := sqlite.New(tmpDir + "/test.db")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	if err := s.Ensure(context.Background()); err != nil {
		s.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Create test database
	databaseID := "test-db-id"
	db := &store.D1Database{
		ID:        databaseID,
		Name:      "test-db",
		Version:   "1.0",
		NumTables: 0,
		FileSize:  0,
		CreatedAt: time.Now(),
	}
	if err := s.D1().CreateDatabase(context.Background(), db); err != nil {
		s.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create database: %v", err)
	}

	// Create runtime with store
	rt := New(Config{
		Store: s,
		Bindings: map[string]string{
			"DB": "d1:" + databaseID,
		},
	})

	return &d1TestHelper{
		rt:         rt,
		store:      s,
		databaseID: databaseID,
		cleanup: func() {
			rt.Close()
			s.Close()
			os.RemoveAll(tmpDir)
		},
	}
}

// executeD1Script executes a script with D1 binding
func (h *d1TestHelper) executeD1Script(t *testing.T, script string) *WorkerResponse {
	t.Helper()

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := h.rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	return resp
}

// setupTestTable creates a test table with sample data
func (h *d1TestHelper) setupTestTable(t *testing.T) {
	t.Helper()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT UNIQUE, age INTEGER, active INTEGER DEFAULT 1)").then(() => {
				DB.exec("INSERT INTO users (name, email, age) VALUES ('Alice', 'alice@example.com', 30), ('Bob', 'bob@example.com', 25), ('Charlie', 'charlie@example.com', 35)").then(() => {
					event.respondWith(new Response('OK'));
				});
			});
		});
	`
	h.executeD1Script(t, script)
}

// ===========================================================================
// Basic Operations Tests
// ===========================================================================

func TestD1_PrepareAndBind(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			const stmt = DB.prepare("SELECT * FROM users WHERE age > ?").bind(26);
			stmt.all().then(result => {
				event.respondWith(new Response(JSON.stringify(result)));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	if resp.Status != 200 {
		t.Fatalf("Expected status 200, got %d", resp.Status)
	}

	body := string(resp.Body)
	if !strings.Contains(body, "Alice") || !strings.Contains(body, "Charlie") {
		t.Errorf("Expected Alice and Charlie in results, got: %s", body)
	}
	if strings.Contains(body, `"name":"Bob"`) {
		t.Errorf("Bob should not be in results (age 25), got: %s", body)
	}
}

func TestD1_MultipleBind(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			const stmt = DB.prepare("SELECT * FROM users WHERE age > ? AND age < ?")
				.bind(24)
				.bind(32);
			stmt.all().then(result => {
				event.respondWith(new Response(JSON.stringify(result)));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	if resp.Status != 200 {
		t.Fatalf("Expected status 200, got %d", resp.Status)
	}

	body := string(resp.Body)
	if !strings.Contains(body, "Alice") || !strings.Contains(body, "Bob") {
		t.Errorf("Expected Alice and Bob in results, got: %s", body)
	}
}

func TestD1_BindChaining(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT * FROM users WHERE name = ? AND age = ?")
				.bind("Alice", 30)
				.first()
				.then(result => {
					event.respondWith(new Response(JSON.stringify(result)));
				});
		});
	`

	resp := h.executeD1Script(t, script)
	if resp.Status != 200 {
		t.Fatalf("Expected status 200, got %d", resp.Status)
	}

	body := string(resp.Body)
	if !strings.Contains(body, "Alice") || !strings.Contains(body, "alice@example.com") {
		t.Errorf("Expected Alice's full record, got: %s", body)
	}
}

// ===========================================================================
// Query Methods Tests
// ===========================================================================

func TestD1_First_ReturnsFirstRow(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT * FROM users ORDER BY id").first().then(result => {
				event.respondWith(new Response(JSON.stringify(result)));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	if resp.Status != 200 {
		t.Fatalf("Expected status 200, got %d", resp.Status)
	}

	body := string(resp.Body)
	if !strings.Contains(body, "Alice") {
		t.Errorf("Expected first row (Alice), got: %s", body)
	}
}

func TestD1_First_ReturnsNull_WhenEmpty(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE IF NOT EXISTS empty_table (id INTEGER PRIMARY KEY)").then(() => {
				DB.prepare("SELECT * FROM empty_table").first().then(result => {
					event.respondWith(new Response(JSON.stringify({ result: result })));
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	if resp.Status != 200 {
		t.Fatalf("Expected status 200, got %d", resp.Status)
	}

	body := string(resp.Body)
	if !strings.Contains(body, `"result":null`) {
		t.Errorf("Expected null result for empty table, got: %s", body)
	}
}

func TestD1_First_WithColumnName(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT * FROM users ORDER BY id").first("name").then(name => {
				DB.prepare("SELECT * FROM users ORDER BY id").first("email").then(email => {
					event.respondWith(new Response(JSON.stringify({ name: name, email: email })));
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	if resp.Status != 200 {
		t.Fatalf("Expected status 200, got %d", resp.Status)
	}

	body := string(resp.Body)
	if !strings.Contains(body, `"name":"Alice"`) {
		t.Errorf("Expected name Alice, got: %s", body)
	}
	if !strings.Contains(body, `"email":"alice@example.com"`) {
		t.Errorf("Expected email alice@example.com, got: %s", body)
	}
}

func TestD1_All_ReturnsAllRows(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT * FROM users ORDER BY id").all().then(result => {
				event.respondWith(new Response(JSON.stringify({
					count: result.results.length,
					names: result.results.map(r => r.name)
				})));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	if resp.Status != 200 {
		t.Fatalf("Expected status 200, got %d", resp.Status)
	}

	body := string(resp.Body)
	if !strings.Contains(body, `"count":3`) {
		t.Errorf("Expected 3 results, got: %s", body)
	}
	if !strings.Contains(body, "Alice") || !strings.Contains(body, "Bob") || !strings.Contains(body, "Charlie") {
		t.Errorf("Expected all three names, got: %s", body)
	}
}

func TestD1_All_ReturnsMetadata(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT * FROM users").all().then(result => {
				event.respondWith(new Response(JSON.stringify({
					success: result.success,
					hasMeta: result.meta !== undefined,
					hasRowsRead: result.meta && result.meta.rows_read !== undefined,
					hasServedBy: result.meta && result.meta.served_by !== undefined,
					hasDuration: result.meta && result.meta.duration !== undefined
				})));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"success":true`) {
		t.Errorf("Expected success:true, got: %s", body)
	}
	if !strings.Contains(body, `"hasMeta":true`) {
		t.Errorf("Expected hasMeta:true, got: %s", body)
	}
	if !strings.Contains(body, `"hasRowsRead":true`) {
		t.Errorf("Expected hasRowsRead:true, got: %s", body)
	}
}

func TestD1_Run_ReturnsChangesCount(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("UPDATE users SET age = age + 1 WHERE age < 30").run().then(result => {
				event.respondWith(new Response(JSON.stringify({
					success: result.success,
					changes: result.meta.changes
				})));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"success":true`) {
		t.Errorf("Expected success:true, got: %s", body)
	}
	if !strings.Contains(body, `"changes":1`) {
		t.Errorf("Expected changes:1 (Bob's age < 30), got: %s", body)
	}
}

func TestD1_Run_ReturnsLastRowID(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("INSERT INTO users (name, email, age) VALUES (?, ?, ?)")
				.bind("Dave", "dave@example.com", 40)
				.run()
				.then(result => {
					event.respondWith(new Response(JSON.stringify({
						success: result.success,
						lastRowId: result.meta.last_row_id
					})));
				});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"success":true`) {
		t.Errorf("Expected success:true, got: %s", body)
	}
	if !strings.Contains(body, `"lastRowId":4`) {
		t.Errorf("Expected lastRowId:4, got: %s", body)
	}
}

func TestD1_Raw_ReturnsArrayOfArrays(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT id, name FROM users ORDER BY id LIMIT 2").raw().then(result => {
				event.respondWith(new Response(JSON.stringify({
					isArray: Array.isArray(result),
					firstRowIsArray: Array.isArray(result[0]),
					firstRow: result[0]
				})));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"isArray":true`) {
		t.Errorf("Expected isArray:true, got: %s", body)
	}
	if !strings.Contains(body, `"firstRowIsArray":true`) {
		t.Errorf("Expected firstRowIsArray:true, got: %s", body)
	}
}

func TestD1_Raw_WithColumnNames(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT id, name FROM users ORDER BY id LIMIT 2")
				.raw({ columnNames: true })
				.then(result => {
					event.respondWith(new Response(JSON.stringify({
						firstRow: result[0],
						hasColumnNames: result[0][0] === "id" && result[0][1] === "name",
						dataRowCount: result.length - 1
					})));
				});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"hasColumnNames":true`) {
		t.Errorf("Expected hasColumnNames:true, got: %s", body)
	}
	if !strings.Contains(body, `"dataRowCount":2`) {
		t.Errorf("Expected dataRowCount:2, got: %s", body)
	}
}

// ===========================================================================
// Batch Operations Tests
// ===========================================================================

func TestD1_Batch_MultipleStatements(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.batch([
				DB.prepare("SELECT * FROM users WHERE id = 1"),
				DB.prepare("SELECT * FROM users WHERE id = 2"),
				DB.prepare("SELECT COUNT(*) as count FROM users"),
			]).then(results => {
				event.respondWith(new Response(JSON.stringify({
					count: results.length,
					firstResult: results[0].results[0].name,
					secondResult: results[1].results[0].name,
					totalCount: results[2].results[0].count
				})));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"count":3`) {
		t.Errorf("Expected 3 results, got: %s", body)
	}
	if !strings.Contains(body, `"firstResult":"Alice"`) {
		t.Errorf("Expected firstResult:Alice, got: %s", body)
	}
	if !strings.Contains(body, `"secondResult":"Bob"`) {
		t.Errorf("Expected secondResult:Bob, got: %s", body)
	}
	if !strings.Contains(body, `"totalCount":3`) {
		t.Errorf("Expected totalCount:3, got: %s", body)
	}
}

func TestD1_Batch_MixedOperations(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.batch([
				DB.prepare("INSERT INTO users (name, email, age) VALUES (?, ?, ?)").bind("Eve", "eve@example.com", 28),
				DB.prepare("SELECT COUNT(*) as count FROM users"),
				DB.prepare("UPDATE users SET age = 29 WHERE name = ?").bind("Eve"),
			]).then(results => {
				event.respondWith(new Response(JSON.stringify({
					insertSuccess: results[0].success,
					countAfterInsert: results[1].results[0].count,
					updateChanges: results[2].meta.changes
				})));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"insertSuccess":true`) {
		t.Errorf("Expected insertSuccess:true, got: %s", body)
	}
	if !strings.Contains(body, `"countAfterInsert":4`) {
		t.Errorf("Expected countAfterInsert:4, got: %s", body)
	}
	if !strings.Contains(body, `"updateChanges":1`) {
		t.Errorf("Expected updateChanges:1, got: %s", body)
	}
}

// ===========================================================================
// Exec Operations Tests
// ===========================================================================

func TestD1_Exec_SingleStatement(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)").then(result => {
				event.respondWith(new Response(JSON.stringify({
					count: result.count,
					hasDuration: result.duration !== undefined
				})));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"count":1`) {
		t.Errorf("Expected count:1, got: %s", body)
	}
	if !strings.Contains(body, `"hasDuration":true`) {
		t.Errorf("Expected hasDuration:true, got: %s", body)
	}
}

func TestD1_Exec_MultipleStatements(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE test1 (id INTEGER PRIMARY KEY); CREATE TABLE test2 (id INTEGER PRIMARY KEY); INSERT INTO test1 (id) VALUES (1);").then(result => {
				event.respondWith(new Response(JSON.stringify({
					count: result.count
				})));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"count":3`) {
		t.Errorf("Expected count:3, got: %s", body)
	}
}

// ===========================================================================
// Data Types Tests
// ===========================================================================

func TestD1_DataTypes_Integer(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE test (val INTEGER)").then(() => {
				DB.prepare("INSERT INTO test (val) VALUES (?)").bind(42).run().then(() => {
					DB.prepare("SELECT val FROM test").first("val").then(result => {
						event.respondWith(new Response(JSON.stringify({ value: result, type: typeof result })));
					});
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"value":42`) {
		t.Errorf("Expected value:42, got: %s", body)
	}
}

func TestD1_DataTypes_Real(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE test (val REAL)").then(() => {
				DB.prepare("INSERT INTO test (val) VALUES (?)").bind(3.14159).run().then(() => {
					DB.prepare("SELECT val FROM test").first("val").then(result => {
						event.respondWith(new Response(JSON.stringify({ value: result })));
					});
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, "3.14159") {
		t.Errorf("Expected 3.14159, got: %s", body)
	}
}

func TestD1_DataTypes_Text(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE test (val TEXT)").then(() => {
				DB.prepare("INSERT INTO test (val) VALUES (?)").bind("Hello, World!").run().then(() => {
					DB.prepare("SELECT val FROM test").first("val").then(result => {
						event.respondWith(new Response(JSON.stringify({ value: result })));
					});
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"value":"Hello, World!"`) {
		t.Errorf("Expected Hello, World!, got: %s", body)
	}
}

func TestD1_DataTypes_Null(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE test (val TEXT)").then(() => {
				DB.prepare("INSERT INTO test (val) VALUES (?)").bind(null).run().then(() => {
					DB.prepare("SELECT val FROM test").first("val").then(result => {
						event.respondWith(new Response(JSON.stringify({ value: result, isNull: result === null })));
					});
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"isNull":true`) {
		t.Errorf("Expected isNull:true, got: %s", body)
	}
}

// ===========================================================================
// Parameter Binding Tests
// ===========================================================================

func TestD1_Bind_AnonymousPlaceholders(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT * FROM users WHERE name = ? AND age = ?")
				.bind("Alice", 30)
				.first()
				.then(result => {
					event.respondWith(new Response(JSON.stringify(result)));
				});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, "alice@example.com") {
		t.Errorf("Expected Alice's email, got: %s", body)
	}
}

func TestD1_Bind_NullValue(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE test (id INTEGER, val TEXT)").then(() => {
				DB.prepare("INSERT INTO test (id, val) VALUES (?, ?)").bind(1, null).run().then(() => {
					DB.prepare("SELECT * FROM test WHERE id = ?").bind(1).first().then(result => {
						event.respondWith(new Response(JSON.stringify({ val: result.val, isNull: result.val === null })));
					});
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"isNull":true`) {
		t.Errorf("Expected isNull:true, got: %s", body)
	}
}

// ===========================================================================
// Metadata Tests
// ===========================================================================

func TestD1_Meta_RowsRead(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT * FROM users").all().then(result => {
				event.respondWith(new Response(JSON.stringify({ rowsRead: result.meta.rows_read })));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"rowsRead":3`) {
		t.Errorf("Expected rowsRead:3, got: %s", body)
	}
}

func TestD1_Meta_RowsWritten(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("UPDATE users SET age = age + 1").run().then(result => {
				event.respondWith(new Response(JSON.stringify({ rowsWritten: result.meta.rows_written })));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"rowsWritten":3`) {
		t.Errorf("Expected rowsWritten:3, got: %s", body)
	}
}

func TestD1_Meta_Changes(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("DELETE FROM users WHERE age < 30").run().then(result => {
				event.respondWith(new Response(JSON.stringify({ changes: result.meta.changes })));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"changes":1`) {
		t.Errorf("Expected changes:1, got: %s", body)
	}
}

func TestD1_Meta_ChangedDB(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT * FROM users").all().then(selectResult => {
				DB.prepare("UPDATE users SET age = age + 1 WHERE id = 1").run().then(updateResult => {
					event.respondWith(new Response(JSON.stringify({
						selectChangedDB: selectResult.meta.changed_db,
						updateChangedDB: updateResult.meta.changed_db
					})));
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"selectChangedDB":false`) {
		t.Errorf("Expected selectChangedDB:false, got: %s", body)
	}
	if !strings.Contains(body, `"updateChangedDB":true`) {
		t.Errorf("Expected updateChangedDB:true, got: %s", body)
	}
}

// ===========================================================================
// Error Handling Tests
// ===========================================================================

func TestD1_Error_InvalidSQL(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.prepare("INVALID SQL SYNTAX").all().then(result => {
				event.respondWith(new Response("Should have thrown"));
			}).catch(e => {
				event.respondWith(new Response(JSON.stringify({ error: true, message: e.message })));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"error":true`) {
		t.Errorf("Expected error:true, got: %s", body)
	}
}

func TestD1_Error_TableNotFound(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.prepare("SELECT * FROM nonexistent_table").all().then(result => {
				event.respondWith(new Response("Should have thrown"));
			}).catch(e => {
				event.respondWith(new Response(JSON.stringify({ error: true })));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"error":true`) {
		t.Errorf("Expected error:true, got: %s", body)
	}
}

func TestD1_Error_ConstraintViolation(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			// Try to insert duplicate email (UNIQUE constraint)
			DB.prepare("INSERT INTO users (name, email, age) VALUES (?, ?, ?)")
				.bind("Duplicate", "alice@example.com", 25)
				.run()
				.then(result => {
					event.respondWith(new Response("Should have thrown"));
				}).catch(e => {
					event.respondWith(new Response(JSON.stringify({ error: true })));
				});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"error":true`) {
		t.Errorf("Expected error:true for constraint violation, got: %s", body)
	}
}

// ===========================================================================
// Edge Cases Tests
// ===========================================================================

func TestD1_EmptyTable(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE empty_test (id INTEGER PRIMARY KEY, val TEXT)").then(() => {
				DB.prepare("SELECT * FROM empty_test").all().then(result => {
					event.respondWith(new Response(JSON.stringify({
						success: result.success,
						resultsLength: result.results.length,
						isArray: Array.isArray(result.results)
					})));
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"success":true`) {
		t.Errorf("Expected success:true, got: %s", body)
	}
	if !strings.Contains(body, `"resultsLength":0`) {
		t.Errorf("Expected resultsLength:0, got: %s", body)
	}
	if !strings.Contains(body, `"isArray":true`) {
		t.Errorf("Expected isArray:true, got: %s", body)
	}
}

func TestD1_UnicodeData(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE unicode_test (val TEXT)").then(() => {
				DB.prepare("INSERT INTO unicode_test (val) VALUES (?)").bind("Hello World").run().then(() => {
					DB.prepare("SELECT val FROM unicode_test").first("val").then(result => {
						event.respondWith(new Response(JSON.stringify({ value: result })));
					});
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, "Hello") {
		t.Errorf("Expected text data, got: %s", body)
	}
}

// ===========================================================================
// Cloudflare Compatibility Tests
// ===========================================================================

func TestD1_Compat_TodoApp(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE IF NOT EXISTS todos (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, completed INTEGER DEFAULT 0)").then(() => {
				// Add some todos
				DB.prepare("INSERT INTO todos (title) VALUES (?)").bind("Buy groceries").run().then(() => {
					DB.prepare("INSERT INTO todos (title) VALUES (?)").bind("Walk the dog").run().then(() => {
						DB.prepare("INSERT INTO todos (title) VALUES (?)").bind("Write code").run().then(() => {
							// Mark one as complete
							DB.prepare("UPDATE todos SET completed = 1 WHERE title = ?").bind("Walk the dog").run().then(() => {
								// Get all incomplete todos
								DB.prepare("SELECT * FROM todos WHERE completed = 0").all().then(incomplete => {
									// Get count of completed
									DB.prepare("SELECT COUNT(*) as count FROM todos WHERE completed = 1").first("count").then(completedCount => {
										event.respondWith(new Response(JSON.stringify({
											incompleteCount: incomplete.results.length,
											completedCount: completedCount
										})));
									});
								});
							});
						});
					});
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"incompleteCount":2`) {
		t.Errorf("Expected 2 incomplete todos, got: %s", body)
	}
	if !strings.Contains(body, `"completedCount":1`) {
		t.Errorf("Expected 1 completed todo, got: %s", body)
	}
}

func TestD1_Compat_Aggregations(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			DB.exec("CREATE TABLE sales (id INTEGER PRIMARY KEY, product TEXT, amount REAL, quantity INTEGER); INSERT INTO sales (product, amount, quantity) VALUES ('Widget', 19.99, 5), ('Widget', 19.99, 3), ('Gadget', 29.99, 2), ('Gadget', 29.99, 4), ('Gizmo', 9.99, 10)").then(() => {
				DB.prepare("SELECT product, SUM(amount * quantity) as total_revenue, SUM(quantity) as total_quantity FROM sales GROUP BY product ORDER BY total_revenue DESC").all().then(stats => {
					event.respondWith(new Response(JSON.stringify({ products: stats.results })));
				});
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, "Widget") || !strings.Contains(body, "Gadget") || !strings.Contains(body, "Gizmo") {
		t.Errorf("Expected product aggregations, got: %s", body)
	}
}

func TestD1_Dump_ReturnsArrayBuffer(t *testing.T) {
	h := newD1TestHelper(t)
	defer h.cleanup()

	h.setupTestTable(t)

	script := `
		addEventListener('fetch', event => {
			DB.dump().then(dump => {
				event.respondWith(new Response(JSON.stringify({
					isArrayBuffer: dump instanceof ArrayBuffer,
					hasSize: dump.byteLength > 0
				})));
			});
		});
	`

	resp := h.executeD1Script(t, script)
	body := string(resp.Body)

	if !strings.Contains(body, `"isArrayBuffer":true`) {
		t.Errorf("Expected isArrayBuffer:true, got: %s", body)
	}
	if !strings.Contains(body, `"hasSize":true`) {
		t.Errorf("Expected hasSize:true, got: %s", body)
	}
}
