package runtime

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
	"github.com/go-mizu/blueprints/localflare/store/sqlite"
)

// r2TestHelper creates a test runtime with R2 store
type r2TestHelper struct {
	rt       *Runtime
	store    store.Store
	bucketID string
	cleanup  func()
}

func newR2TestHelper(t *testing.T) *r2TestHelper {
	t.Helper()

	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "r2test-*")
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

	// Create test bucket
	bucketID := "test-bucket-id"
	bucket := &store.R2Bucket{
		ID:        bucketID,
		Name:      "test-bucket",
		Location:  "auto",
		CreatedAt: time.Now(),
	}
	if err := s.R2().CreateBucket(context.Background(), bucket); err != nil {
		s.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create bucket: %v", err)
	}

	// Create runtime with store
	rt := New(Config{
		Store: s,
		Bindings: map[string]string{
			"BUCKET": "r2:" + bucketID,
		},
	})

	return &r2TestHelper{
		rt:       rt,
		store:    s,
		bucketID: bucketID,
		cleanup: func() {
			rt.Close()
			s.Close()
			os.RemoveAll(tmpDir)
		},
	}
}

// executeR2Script executes a script with R2 binding
func (h *r2TestHelper) executeR2Script(t *testing.T, script string) *WorkerResponse {
	t.Helper()

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := h.rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	return resp
}

// ===========================================================================
// Basic Operations Tests
// ===========================================================================

func TestR2_Put_Get_Delete(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			// Put
			BUCKET.put('test-key', 'Hello R2!').then(putResult => {
				if (!putResult) {
					event.respondWith(new Response('put failed'));
					return;
				}

				// Get
				BUCKET.get('test-key').then(getResult => {
					if (!getResult) {
						event.respondWith(new Response('get failed'));
						return;
					}
					getResult.text().then(text => {
						// Delete
						BUCKET.delete('test-key').then(() => {
							// Verify delete
							BUCKET.get('test-key').then(deleted => {
								event.respondWith(new Response(text + ':' + (deleted === null)));
							});
						});
					});
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "Hello R2!:true" {
		t.Errorf("Expected 'Hello R2!:true', got '%s'", string(resp.Body))
	}
}

func TestR2_Head_ReturnsMetadataOnly(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test object
	_, err := h.store.R2().PutObject(context.Background(), h.bucketID, "head-test",
		[]byte("test content"), &store.R2PutOptions{
			HTTPMetadata: &store.R2HTTPMetadata{ContentType: "text/plain"},
		})
	if err != nil {
		t.Fatalf("Failed to put object: %v", err)
	}

	script := `
		addEventListener('fetch', event => {
			BUCKET.head('head-test').then(result => {
				if (!result) {
					event.respondWith(new Response('null'));
					return;
				}

				// head() returns R2Object (no body)
				const hasBody = result.body !== undefined;
				const hasKey = result.key === 'head-test';
				const hasSize = result.size === 12;
				const hasEtag = typeof result.etag === 'string' && result.etag.length > 0;

				event.respondWith(new Response([hasBody, hasKey, hasSize, hasEtag].join(':')));
			});
		});
	`

	resp := h.executeR2Script(t, script)

	// head() should not have body
	if string(resp.Body) != "false:true:true:true" {
		t.Errorf("Expected 'false:true:true:true', got '%s'", string(resp.Body))
	}
}

func TestR2_Get_NonExistent_ReturnsNull(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			BUCKET.get('non-existent-key').then(result => {
				event.respondWith(new Response(String(result === null)));
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true" {
		t.Errorf("Expected 'true', got '%s'", string(resp.Body))
	}
}

func TestR2_Put_ReturnsR2Object(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			BUCKET.put('put-test', 'test data').then(result => {
				const hasKey = result.key === 'put-test';
				const hasSize = result.size === 9;
				const hasEtag = typeof result.etag === 'string';
				const hasHttpEtag = typeof result.httpEtag === 'string' && result.httpEtag.startsWith('"');
				const hasUploaded = result.uploaded instanceof Date;

				event.respondWith(new Response([hasKey, hasSize, hasEtag, hasHttpEtag, hasUploaded].join(':')));
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true:true:true:true:true" {
		t.Errorf("Expected all true, got '%s'", string(resp.Body))
	}
}

func TestR2_Delete_Single(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test object
	h.store.R2().PutObject(context.Background(), h.bucketID, "delete-single", []byte("data"), nil)

	script := `
		addEventListener('fetch', event => {
			// Verify exists
			BUCKET.head('delete-single').then(before => {
				// Delete
				BUCKET.delete('delete-single').then(() => {
					// Verify deleted
					BUCKET.head('delete-single').then(after => {
						event.respondWith(new Response((before !== null) + ':' + (after === null)));
					});
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true:true" {
		t.Errorf("Expected 'true:true', got '%s'", string(resp.Body))
	}
}

func TestR2_Delete_Batch(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test objects
	for i := 1; i <= 5; i++ {
		h.store.R2().PutObject(context.Background(), h.bucketID,
			"batch-"+string(rune('0'+i)), []byte("data"), nil)
	}

	script := `
		addEventListener('fetch', event => {
			// Delete multiple keys
			BUCKET.delete(['batch-1', 'batch-2', 'batch-3', 'batch-4', 'batch-5']).then(() => {
				// Check all deleted using Promise.all
				Promise.all([
					BUCKET.head('batch-1'),
					BUCKET.head('batch-2'),
					BUCKET.head('batch-3'),
					BUCKET.head('batch-4'),
					BUCKET.head('batch-5')
				]).then(results => {
					const allDeleted = results.every(r => r === null);
					event.respondWith(new Response(String(allDeleted)));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true" {
		t.Errorf("Expected 'true', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// List Operations Tests
// ===========================================================================

func TestR2_List_Basic(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test objects
	for i := 1; i <= 3; i++ {
		h.store.R2().PutObject(context.Background(), h.bucketID,
			"list-"+string(rune('0'+i)), []byte("data"), nil)
	}

	script := `
		addEventListener('fetch', event => {
			BUCKET.list().then(result => {
				const hasObjects = Array.isArray(result.objects);
				const count = result.objects.length;
				const hasTruncated = typeof result.truncated === 'boolean';

				event.respondWith(new Response([hasObjects, count, hasTruncated].join(':')));
			});
		});
	`

	resp := h.executeR2Script(t, script)

	// truncated should be boolean, so typeof check returns true
	if string(resp.Body) != "true:3:true" {
		t.Errorf("Expected 'true:3:true', got '%s'", string(resp.Body))
	}
}

func TestR2_List_WithPrefix(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put objects with different prefixes
	h.store.R2().PutObject(context.Background(), h.bucketID, "images/photo1.jpg", []byte("data"), nil)
	h.store.R2().PutObject(context.Background(), h.bucketID, "images/photo2.jpg", []byte("data"), nil)
	h.store.R2().PutObject(context.Background(), h.bucketID, "docs/file.txt", []byte("data"), nil)

	script := `
		addEventListener('fetch', event => {
			BUCKET.list({ prefix: 'images/' }).then(result => {
				const count = result.objects.length;
				const allMatch = result.objects.every(obj => obj.key.startsWith('images/'));

				event.respondWith(new Response([count, allMatch].join(':')));
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "2:true" {
		t.Errorf("Expected '2:true', got '%s'", string(resp.Body))
	}
}

func TestR2_List_WithDelimiter(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put objects with hierarchy
	h.store.R2().PutObject(context.Background(), h.bucketID, "folder1/file1.txt", []byte("data"), nil)
	h.store.R2().PutObject(context.Background(), h.bucketID, "folder1/file2.txt", []byte("data"), nil)
	h.store.R2().PutObject(context.Background(), h.bucketID, "folder2/file3.txt", []byte("data"), nil)
	h.store.R2().PutObject(context.Background(), h.bucketID, "root.txt", []byte("data"), nil)

	script := `
		addEventListener('fetch', event => {
			BUCKET.list({ delimiter: '/' }).then(result => {
				// With delimiter, we should get delimitedPrefixes
				const objectCount = result.objects.length;
				const hasPrefixes = Array.isArray(result.delimitedPrefixes);
				const prefixCount = result.delimitedPrefixes ? result.delimitedPrefixes.length : 0;

				event.respondWith(new Response([objectCount, hasPrefixes, prefixCount].join(':')));
			});
		});
	`

	resp := h.executeR2Script(t, script)

	// Should have 1 root object (root.txt) and 2 prefixes (folder1/, folder2/)
	if string(resp.Body) != "1:true:2" {
		t.Errorf("Expected '1:true:2', got '%s'", string(resp.Body))
	}
}

func TestR2_List_Pagination(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put 10 objects
	for i := 0; i < 10; i++ {
		key := "page-" + string(rune('0'+i))
		h.store.R2().PutObject(context.Background(), h.bucketID, key, []byte("data"), nil)
	}

	script := `
		addEventListener('fetch', event => {
			// First page
			BUCKET.list({ limit: 5 }).then(page1 => {
				const truncated1 = page1.truncated;
				const count1 = page1.objects.length;
				const hasCursor = typeof page1.cursor === 'string' && page1.cursor.length > 0;

				// Second page
				BUCKET.list({ limit: 5, cursor: page1.cursor }).then(page2 => {
					const count2 = page2.objects.length;

					event.respondWith(new Response([truncated1, count1, hasCursor, count2].join(':')));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true:5:true:5" {
		t.Errorf("Expected 'true:5:true:5', got '%s'", string(resp.Body))
	}
}

func TestR2_List_IncludeMetadata(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put object with metadata
	h.store.R2().PutObject(context.Background(), h.bucketID, "meta-test", []byte("data"), &store.R2PutOptions{
		HTTPMetadata:   &store.R2HTTPMetadata{ContentType: "text/plain"},
		CustomMetadata: map[string]string{"custom": "value"},
	})

	script := `
		addEventListener('fetch', event => {
			BUCKET.list({ include: ['httpMetadata', 'customMetadata'] }).then(result => {
				const obj = result.objects[0];
				const hasHttpMeta = obj.httpMetadata !== undefined;
				const hasCustomMeta = obj.customMetadata !== undefined;

				let httpType = '';
				let customVal = '';
				if (hasHttpMeta && obj.httpMetadata) {
					httpType = obj.httpMetadata.contentType || '';
				}
				if (hasCustomMeta && obj.customMetadata) {
					customVal = obj.customMetadata.custom || '';
				}

				event.respondWith(new Response([hasHttpMeta, hasCustomMeta, httpType, customVal].join(':')));
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true:true:text/plain:value" {
		t.Errorf("Expected 'true:true:text/plain:value', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// Conditional Operations Tests
// ===========================================================================

func TestR2_Get_OnlyIf_EtagMatches(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test object
	obj, _ := h.store.R2().PutObject(context.Background(), h.bucketID, "etag-match",
		[]byte("test content"), nil)
	etag := obj.ETag

	script := strings.Replace(`
		addEventListener('fetch', event => {
			// Should succeed with matching etag
			BUCKET.get('etag-match', { onlyIf: { etagMatches: 'ETAG' } }).then(match => {
				// Should fail with non-matching etag
				BUCKET.get('etag-match', { onlyIf: { etagMatches: 'wrong-etag' } }).then(noMatch => {
					const matchHasBody = match && typeof match.text === 'function';
					const noMatchIsObject = noMatch && !noMatch.body;

					event.respondWith(new Response([matchHasBody, noMatchIsObject].join(':')));
				});
			});
		});
	`, "ETAG", etag, 1)

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true:true" {
		t.Errorf("Expected 'true:true', got '%s'", string(resp.Body))
	}
}

func TestR2_Get_OnlyIf_EtagDoesNotMatch(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test object
	obj, _ := h.store.R2().PutObject(context.Background(), h.bucketID, "etag-notmatch",
		[]byte("test content"), nil)
	etag := obj.ETag

	script := strings.Replace(`
		addEventListener('fetch', event => {
			// Should succeed with non-matching etag
			BUCKET.get('etag-notmatch', { onlyIf: { etagDoesNotMatch: 'different-etag' } }).then(success => {
				// Should return metadata only with matching etag
				BUCKET.get('etag-notmatch', { onlyIf: { etagDoesNotMatch: 'ETAG' } }).then(fail => {
					const successHasBody = success && typeof success.text === 'function';
					const failNoBody = fail && !fail.body;

					event.respondWith(new Response([successHasBody, failNoBody].join(':')));
				});
			});
		});
	`, "ETAG", etag, 1)

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true:true" {
		t.Errorf("Expected 'true:true', got '%s'", string(resp.Body))
	}
}

func TestR2_Get_OnlyIf_UploadedBefore(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test object
	h.store.R2().PutObject(context.Background(), h.bucketID, "time-before",
		[]byte("test content"), nil)

	script := `
		addEventListener('fetch', event => {
			const future = new Date(Date.now() + 86400000); // Tomorrow
			const past = new Date(Date.now() - 86400000); // Yesterday

			// Should succeed - object was uploaded before tomorrow
			BUCKET.get('time-before', { onlyIf: { uploadedBefore: future } }).then(success => {
				// Should fail - object was not uploaded before yesterday
				BUCKET.get('time-before', { onlyIf: { uploadedBefore: past } }).then(fail => {
					const successHasBody = success && typeof success.text === 'function';
					const failNoBody = fail && !fail.body;

					event.respondWith(new Response([successHasBody, failNoBody].join(':')));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true:true" {
		t.Errorf("Expected 'true:true', got '%s'", string(resp.Body))
	}
}

func TestR2_Get_OnlyIf_UploadedAfter(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test object
	h.store.R2().PutObject(context.Background(), h.bucketID, "time-after",
		[]byte("test content"), nil)

	script := `
		addEventListener('fetch', event => {
			const future = new Date(Date.now() + 86400000); // Tomorrow
			const past = new Date(Date.now() - 86400000); // Yesterday

			// Should succeed - object was uploaded after yesterday
			BUCKET.get('time-after', { onlyIf: { uploadedAfter: past } }).then(success => {
				// Should fail - object was not uploaded after tomorrow
				BUCKET.get('time-after', { onlyIf: { uploadedAfter: future } }).then(fail => {
					const successHasBody = success && typeof success.text === 'function';
					const failNoBody = fail && !fail.body;

					event.respondWith(new Response([successHasBody, failNoBody].join(':')));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true:true" {
		t.Errorf("Expected 'true:true', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// Range Requests Tests
// ===========================================================================

func TestR2_Get_Range_OffsetLength(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test object with known content
	h.store.R2().PutObject(context.Background(), h.bucketID, "range-test",
		[]byte("0123456789ABCDEF"), nil)

	script := `
		addEventListener('fetch', event => {
			// Get bytes 5-9 (5 bytes starting at offset 5)
			BUCKET.get('range-test', { range: { offset: 5, length: 5 } }).then(result => {
				result.text().then(text => {
					const hasRange = result.range !== undefined;
					const rangeOffset = result.range ? result.range.offset : -1;
					const rangeLength = result.range ? result.range.length : -1;

					event.respondWith(new Response([text, hasRange, rangeOffset, rangeLength].join(':')));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "56789:true:5:5" {
		t.Errorf("Expected '56789:true:5:5', got '%s'", string(resp.Body))
	}
}

func TestR2_Get_Range_OffsetOnly(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test object
	h.store.R2().PutObject(context.Background(), h.bucketID, "range-offset",
		[]byte("0123456789ABCDEF"), nil)

	script := `
		addEventListener('fetch', event => {
			// Get all bytes from offset 10 to end
			BUCKET.get('range-offset', { range: { offset: 10 } }).then(result => {
				result.text().then(text => {
					event.respondWith(new Response(text));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "ABCDEF" {
		t.Errorf("Expected 'ABCDEF', got '%s'", string(resp.Body))
	}
}

func TestR2_Get_Range_Suffix(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put test object
	h.store.R2().PutObject(context.Background(), h.bucketID, "range-suffix",
		[]byte("0123456789ABCDEF"), nil)

	script := `
		addEventListener('fetch', event => {
			// Get last 4 bytes
			BUCKET.get('range-suffix', { range: { suffix: 4 } }).then(result => {
				result.text().then(text => {
					event.respondWith(new Response(text));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "CDEF" {
		t.Errorf("Expected 'CDEF', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// HTTP Metadata Tests
// ===========================================================================

func TestR2_Put_WithHTTPMetadata(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			BUCKET.put('http-meta', 'content', {
				httpMetadata: {
					contentType: 'application/json',
					contentLanguage: 'en-US',
					contentDisposition: 'attachment; filename="test.json"',
					contentEncoding: 'gzip',
					cacheControl: 'max-age=3600'
				}
			}).then(() => {
				BUCKET.get('http-meta').then(obj => {
					const meta = obj.httpMetadata;

					event.respondWith(new Response([
						meta.contentType,
						meta.contentLanguage,
						meta.contentDisposition,
						meta.contentEncoding,
						meta.cacheControl
					].join('|')));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	expected := "application/json|en-US|attachment; filename=\"test.json\"|gzip|max-age=3600"
	if string(resp.Body) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(resp.Body))
	}
}

func TestR2_WriteHttpMetadata(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put object with HTTP metadata
	h.store.R2().PutObject(context.Background(), h.bucketID, "write-http-meta", []byte("data"),
		&store.R2PutOptions{
			HTTPMetadata: &store.R2HTTPMetadata{
				ContentType:        "text/html",
				CacheControl:       "no-cache",
				ContentDisposition: "inline",
			},
		})

	script := `
		addEventListener('fetch', event => {
			BUCKET.get('write-http-meta').then(obj => {
				const headers = new Headers();
				obj.writeHttpMetadata(headers);

				event.respondWith(new Response([
					headers.get('content-type'),
					headers.get('cache-control'),
					headers.get('content-disposition')
				].join('|')));
			});
		});
	`

	resp := h.executeR2Script(t, script)

	expected := "text/html|no-cache|inline"
	if string(resp.Body) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(resp.Body))
	}
}

// ===========================================================================
// Custom Metadata Tests
// ===========================================================================

func TestR2_Put_WithCustomMetadata(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			BUCKET.put('custom-meta', 'content', {
				customMetadata: {
					'x-custom-key': 'custom-value',
					'x-another': 'another-value'
				}
			}).then(() => {
				BUCKET.get('custom-meta').then(obj => {
					const meta = obj.customMetadata;

					event.respondWith(new Response([
						meta['x-custom-key'],
						meta['x-another']
					].join(':')));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "custom-value:another-value" {
		t.Errorf("Expected 'custom-value:another-value', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// Checksums Tests
// ===========================================================================

func TestR2_Put_WithMD5(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	content := []byte("test content for md5")
	hash := md5.Sum(content)

	script := strings.Replace(`
		addEventListener('fetch', event => {
			// Put with correct MD5
			BUCKET.put('md5-test', 'test content for md5', { md5: 'MD5_HEX' }).then(result => {
				BUCKET.get('md5-test').then(obj => {
					const hasMd5 = obj.checksums && obj.checksums.md5;

					event.respondWith(new Response(String(hasMd5 !== undefined)));
				});
			});
		});
	`, "MD5_HEX", hex.EncodeToString(hash[:]), 1)

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true" {
		t.Errorf("Expected 'true', got '%s'", string(resp.Body))
	}
}

func TestR2_Put_WithSHA256(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	content := []byte("test content for sha256")
	hash := sha256.Sum256(content)

	script := strings.Replace(`
		addEventListener('fetch', event => {
			BUCKET.put('sha256-test', 'test content for sha256', { sha256: 'SHA256_HEX' }).then(result => {
				BUCKET.get('sha256-test').then(obj => {
					const hasSha256 = obj.checksums && obj.checksums.sha256;

					event.respondWith(new Response(String(hasSha256 !== undefined)));
				});
			});
		});
	`, "SHA256_HEX", hex.EncodeToString(hash[:]), 1)

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true" {
		t.Errorf("Expected 'true', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// Multipart Upload Tests
// ===========================================================================

func TestR2_CreateMultipartUpload(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			BUCKET.createMultipartUpload('multipart-key').then(upload => {
				const hasKey = upload.key === 'multipart-key';
				const hasUploadId = typeof upload.uploadId === 'string' && upload.uploadId.length > 0;
				const hasUploadPart = typeof upload.uploadPart === 'function';
				const hasAbort = typeof upload.abort === 'function';
				const hasComplete = typeof upload.complete === 'function';

				event.respondWith(new Response([hasKey, hasUploadId, hasUploadPart, hasAbort, hasComplete].join(':')));
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true:true:true:true:true" {
		t.Errorf("Expected all true, got '%s'", string(resp.Body))
	}
}

func TestR2_MultipartUpload_FullFlow(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			// Create multipart upload
			BUCKET.createMultipartUpload('full-multipart', {
				httpMetadata: { contentType: 'application/octet-stream' }
			}).then(upload => {
				// Upload parts
				upload.uploadPart(1, 'Part 1 content ').then(part1 => {
					upload.uploadPart(2, 'Part 2 content ').then(part2 => {
						upload.uploadPart(3, 'Part 3 content').then(part3 => {
							// Complete
							upload.complete([part1, part2, part3]).then(result => {
								// Verify
								BUCKET.get('full-multipart').then(obj => {
									obj.text().then(text => {
										event.respondWith(new Response(text));
									});
								});
							});
						});
					});
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	expected := "Part 1 content Part 2 content Part 3 content"
	if string(resp.Body) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(resp.Body))
	}
}

func TestR2_ResumeMultipartUpload(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			// Create multipart upload
			BUCKET.createMultipartUpload('resume-multipart').then(upload => {
				const uploadId = upload.uploadId;

				// Upload a part
				upload.uploadPart(1, 'Part 1 ').then(part1 => {
					// Resume from upload ID
					const resumed = BUCKET.resumeMultipartUpload('resume-multipart', uploadId);

					// Upload another part via resumed
					resumed.uploadPart(2, 'Part 2').then(part2 => {
						// Complete via resumed
						resumed.complete([part1, part2]).then(result => {
							// Verify
							BUCKET.get('resume-multipart').then(obj => {
								obj.text().then(text => {
									event.respondWith(new Response(text));
								});
							});
						});
					});
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "Part 1 Part 2" {
		t.Errorf("Expected 'Part 1 Part 2', got '%s'", string(resp.Body))
	}
}

func TestR2_AbortMultipartUpload(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			// Create multipart upload
			BUCKET.createMultipartUpload('abort-multipart').then(upload => {
				// Upload a part
				upload.uploadPart(1, 'Part 1').then(part1 => {
					// Abort
					upload.abort().then(() => {
						// Verify object doesn't exist
						BUCKET.head('abort-multipart').then(obj => {
							event.respondWith(new Response(String(obj === null)));
						});
					});
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true" {
		t.Errorf("Expected 'true', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// Body Methods Tests
// ===========================================================================

func TestR2ObjectBody_Text(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	h.store.R2().PutObject(context.Background(), h.bucketID, "text-test",
		[]byte("Hello World"), nil)

	script := `
		addEventListener('fetch', event => {
			BUCKET.get('text-test').then(obj => {
				obj.text().then(text => {
					event.respondWith(new Response(text));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", string(resp.Body))
	}
}

func TestR2ObjectBody_JSON(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	h.store.R2().PutObject(context.Background(), h.bucketID, "json-test",
		[]byte(`{"name":"test","value":42}`), nil)

	script := `
		addEventListener('fetch', event => {
			BUCKET.get('json-test').then(obj => {
				obj.json().then(data => {
					event.respondWith(new Response(data.name + ':' + data.value));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "test:42" {
		t.Errorf("Expected 'test:42', got '%s'", string(resp.Body))
	}
}

func TestR2ObjectBody_ArrayBuffer(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	h.store.R2().PutObject(context.Background(), h.bucketID, "arraybuffer-test",
		[]byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}, nil) // "Hello" in bytes

	script := `
		addEventListener('fetch', event => {
			BUCKET.get('arraybuffer-test').then(obj => {
				obj.arrayBuffer().then(buffer => {
					const bytes = new Uint8Array(buffer);
					event.respondWith(new Response(bytes.length.toString()));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "5" {
		t.Errorf("Expected '5', got '%s'", string(resp.Body))
	}
}

func TestR2ObjectBody_BodyUsed(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	h.store.R2().PutObject(context.Background(), h.bucketID, "bodyused-test",
		[]byte("content"), nil)

	script := `
		addEventListener('fetch', event => {
			BUCKET.get('bodyused-test').then(obj => {
				const before = obj.bodyUsed;
				obj.text().then(() => {
					const after = obj.bodyUsed;
					event.respondWith(new Response(before + ':' + after));
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "false:true" {
		t.Errorf("Expected 'false:true', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// Cloudflare Worker Compatibility Tests
// ===========================================================================

func TestCompat_R2_BasicWorker(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Simulate a basic R2-using worker
	script := `
		addEventListener('fetch', event => {
			const url = new URL(event.request.url);
			const key = url.pathname.slice(1) || 'default';

			if (event.request.method === 'PUT') {
				BUCKET.put(key, 'test content', {
					httpMetadata: { contentType: 'text/plain' }
				}).then(() => {
					event.respondWith(new Response('Created', { status: 201 }));
				});
				return;
			}

			if (event.request.method === 'GET') {
				BUCKET.get(key).then(object => {
					if (object === null) {
						event.respondWith(new Response('Not Found', { status: 404 }));
						return;
					}
					const headers = new Headers();
					object.writeHttpMetadata(headers);
					headers.set('etag', object.httpEtag);
					event.respondWith(new Response(object.body, { headers }));
				});
				return;
			}

			event.respondWith(new Response('Method not allowed', { status: 405 }));
		});
	`

	// Test PUT
	h.rt.setupR2Binding("BUCKET", h.bucketID)
	req := httptest.NewRequest("PUT", "/my-file", nil)
	resp, err := h.rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("PUT failed: %v", err)
	}
	if resp.Status != 201 {
		t.Errorf("Expected status 201, got %d", resp.Status)
	}

	// Test GET
	h.rt.setupR2Binding("BUCKET", h.bucketID)
	req = httptest.NewRequest("GET", "/my-file", nil)
	resp, err = h.rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}
	if resp.Headers.Get("etag") == "" {
		t.Error("Expected etag header")
	}
}

func TestCompat_R2_ConditionalGet(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Put object
	obj, _ := h.store.R2().PutObject(context.Background(), h.bucketID, "conditional-get",
		[]byte("test content"), nil)

	script := strings.Replace(`
		addEventListener('fetch', event => {
			const ifNoneMatch = event.request.headers.get('if-none-match');

			BUCKET.get('conditional-get', {
				onlyIf: ifNoneMatch ? { etagDoesNotMatch: ifNoneMatch } : undefined
			}).then(object => {
				if (object === null) {
					event.respondWith(new Response('Not Found', { status: 404 }));
					return;
				}

				// Check if it's a 304 response (no body)
				if (!object.body) {
					event.respondWith(new Response(null, { status: 304 }));
					return;
				}

				event.respondWith(new Response(object.body, {
					headers: { 'etag': object.httpEtag }
				}));
			});
		});
	`, "ETAG", `"`+obj.ETag+`"`, 1)

	// Test without If-None-Match
	h.rt.setupR2Binding("BUCKET", h.bucketID)
	req := httptest.NewRequest("GET", "/", nil)
	resp, err := h.rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("Expected 200, got %d", resp.Status)
	}

	// Test with matching If-None-Match (should return 304)
	h.rt.setupR2Binding("BUCKET", h.bucketID)
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("If-None-Match", `"`+obj.ETag+`"`)
	resp, err = h.rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp.Status != 304 {
		t.Errorf("Expected 304, got %d", resp.Status)
	}
}

func TestCompat_R2_StorageClass(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			// Default storage class
			BUCKET.put('standard-obj', 'content').then(() => {
				BUCKET.head('standard-obj').then(standard => {
					// InfrequentAccess storage class
					BUCKET.put('ia-obj', 'content', { storageClass: 'InfrequentAccess' }).then(() => {
						BUCKET.head('ia-obj').then(ia => {
							event.respondWith(new Response([
								standard.storageClass,
								ia.storageClass
							].join(':')));
						});
					});
				});
			});
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "Standard:InfrequentAccess" {
		t.Errorf("Expected 'Standard:InfrequentAccess', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// R2 Binding API Tests (verifies all methods exist)
// ===========================================================================

func TestR2_BindingAPI_AllMethodsExist(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			const methods = [
				typeof BUCKET.get === 'function',
				typeof BUCKET.put === 'function',
				typeof BUCKET.delete === 'function',
				typeof BUCKET.list === 'function',
				typeof BUCKET.head === 'function',
				typeof BUCKET.createMultipartUpload === 'function',
				typeof BUCKET.resumeMultipartUpload === 'function'
			];

			event.respondWith(new Response(methods.every(m => m).toString()));
		});
	`

	resp := h.executeR2Script(t, script)

	if string(resp.Body) != "true" {
		t.Errorf("Expected all methods to exist, got '%s'", string(resp.Body))
	}
}

func TestR2_BindingAPI_GetReturnsPromise(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			let result;
			let errorMsg = '';
			try {
				result = BUCKET.get('nonexistent');
			} catch (e) {
				errorMsg = e.message || e.toString();
			}
			const isPromise = result && typeof result.then === 'function';
			const resultType = typeof result;
			event.respondWith(new Response(resultType + ':' + isPromise + ':' + errorMsg));
		});
	`

	resp := h.executeR2Script(t, script)
	t.Logf("Response body: %s", string(resp.Body))
}

func TestR2_BindingAPI_ListReturnsPromise(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			let result;
			let errorMsg = '';
			try {
				result = BUCKET.list();
			} catch (e) {
				errorMsg = e.message || e.toString();
			}
			const isPromise = result && typeof result.then === 'function';
			const resultType = typeof result;
			event.respondWith(new Response(resultType + ':' + isPromise + ':' + errorMsg));
		});
	`

	resp := h.executeR2Script(t, script)
	t.Logf("Response body: %s", string(resp.Body))
}

func TestR2_Debug_FunctionInvocation(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	// Try calling a method and checking if it returns anything
	script := `
		addEventListener('fetch', event => {
			// Try to see what BUCKET is
			const bucketType = typeof BUCKET;
			const bucketKeys = Object.keys(BUCKET || {}).join(',');

			// Check if it's a function
			const isFunc = typeof BUCKET.get === 'function';

			// Try calling it directly on the object
			let callResult = 'not called';
			if (isFunc) {
				try {
					const r = BUCKET.get('test');
					callResult = typeof r;
					if (r !== undefined) {
						callResult += ':hasThen=' + (typeof r.then === 'function');
					}
				} catch (e) {
					callResult = 'error: ' + e.message;
				}
			}

			event.respondWith(new Response([bucketType, bucketKeys, isFunc, callResult].join('|')));
		});
	`

	resp := h.executeR2Script(t, script)
	t.Logf("Response body: %s", string(resp.Body))
}

func TestR2_Debug_PutThenCallback(t *testing.T) {
	h := newR2TestHelper(t)
	defer h.cleanup()

	script := `
		addEventListener('fetch', event => {
			console.log("Handler starting");
			var promise = BUCKET.put('test-key', 'test-value');
			console.log("Put returned:", promise);
			console.log("Promise type:", typeof promise);
			console.log("Then type:", typeof promise.then);

			var thenResult = promise.then(function(result) {
				console.log("Then callback called with:", result);
				event.respondWith(new Response('then-called:' + (result !== null)));
			});
			console.log("Then returned:", thenResult);
		});
	`

	resp := h.executeR2Script(t, script)
	t.Logf("Response body: %s", string(resp.Body))
	if !strings.Contains(string(resp.Body), "then-called") {
		t.Errorf("Then callback was not called")
	}
}

// ===========================================================================
// Benchmarks
// ===========================================================================

func BenchmarkR2_Put(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "r2bench-*")
	defer os.RemoveAll(tmpDir)

	s, _ := sqlite.New(tmpDir + "/test.db")
	defer s.Close()
	s.Ensure(context.Background())

	bucket := &store.R2Bucket{ID: "bench", Name: "bench", Location: "auto", CreatedAt: time.Now()}
	s.R2().CreateBucket(context.Background(), bucket)

	data := []byte("benchmark content")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.R2().PutObject(context.Background(), "bench", "key", data, nil)
	}
}

func BenchmarkR2_Get(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "r2bench-*")
	defer os.RemoveAll(tmpDir)

	s, _ := sqlite.New(tmpDir + "/test.db")
	defer s.Close()
	s.Ensure(context.Background())

	bucket := &store.R2Bucket{ID: "bench", Name: "bench", Location: "auto", CreatedAt: time.Now()}
	s.R2().CreateBucket(context.Background(), bucket)
	s.R2().PutObject(context.Background(), "bench", "key", []byte("benchmark content"), nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.R2().GetObject(context.Background(), "bench", "key", nil)
	}
}

func BenchmarkR2_List(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "r2bench-*")
	defer os.RemoveAll(tmpDir)

	s, _ := sqlite.New(tmpDir + "/test.db")
	defer s.Close()
	s.Ensure(context.Background())

	bucket := &store.R2Bucket{ID: "bench", Name: "bench", Location: "auto", CreatedAt: time.Now()}
	s.R2().CreateBucket(context.Background(), bucket)

	// Add 100 objects
	for i := 0; i < 100; i++ {
		s.R2().PutObject(context.Background(), "bench", "key"+string(rune(i)), []byte("data"), nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.R2().ListObjects(context.Background(), "bench", nil)
	}
}
