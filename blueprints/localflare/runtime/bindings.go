package runtime

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dop251/goja"

	"github.com/go-mizu/blueprints/localflare/store"
)

// setupKVBinding creates a KV namespace binding.
func (r *Runtime) setupKVBinding(name, namespaceID string) {
	vm := r.vm
	kvStore := r.store.KV()

	kv := vm.NewObject()

	// get(key, options?)
	kv.Set("get", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()

		// Options
		returnType := "text"
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			opts := call.Arguments[1]
			if opts.ExportType().Kind().String() == "string" {
				returnType = opts.String()
			} else {
				optsObj := opts.ToObject(vm)
				if t := optsObj.Get("type"); t != nil && !goja.IsUndefined(t) {
					returnType = t.String()
				}
			}
		}

		pair, err := kvStore.Get(context.Background(), namespaceID, key)
		if err != nil || pair == nil {
			return r.createPromise(goja.Null())
		}

		var result goja.Value
		switch returnType {
		case "json":
			var data interface{}
			json.Unmarshal(pair.Value, &data)
			result = vm.ToValue(data)
		case "arrayBuffer":
			result = vm.ToValue(pair.Value)
		case "stream":
			// Simplified - return as bytes
			result = vm.ToValue(pair.Value)
		default: // text
			result = vm.ToValue(string(pair.Value))
		}

		return r.createPromise(result)
	})

	// getWithMetadata(key, options?)
	kv.Set("getWithMetadata", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()

		pair, err := kvStore.Get(context.Background(), namespaceID, key)
		if err != nil || pair == nil {
			result := vm.NewObject()
			result.Set("value", goja.Null())
			result.Set("metadata", goja.Null())
			return r.createPromise(result)
		}

		result := vm.NewObject()
		result.Set("value", string(pair.Value))

		if pair.Metadata != nil {
			metaObj := vm.NewObject()
			for k, v := range pair.Metadata {
				metaObj.Set(k, v)
			}
			result.Set("metadata", metaObj)
		} else {
			result.Set("metadata", goja.Null())
		}

		return r.createPromise(result)
	})

	// put(key, value, options?)
	kv.Set("put", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		value := call.Argument(1)

		var valueBytes []byte
		switch v := value.Export().(type) {
		case string:
			valueBytes = []byte(v)
		case []byte:
			valueBytes = v
		default:
			jsonBytes, _ := json.Marshal(v)
			valueBytes = jsonBytes
		}

		pair := &kvPair{
			Key:   key,
			Value: valueBytes,
		}

		// Parse options
		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) {
			opts := call.Arguments[2].ToObject(vm)

			// Expiration
			if exp := opts.Get("expiration"); exp != nil && !goja.IsUndefined(exp) {
				expTime := time.Unix(exp.ToInteger(), 0)
				pair.Expiration = &expTime
			}

			// ExpirationTTL
			if ttl := opts.Get("expirationTtl"); ttl != nil && !goja.IsUndefined(ttl) {
				expTime := time.Now().Add(time.Duration(ttl.ToInteger()) * time.Second)
				pair.Expiration = &expTime
			}

			// Metadata
			if meta := opts.Get("metadata"); meta != nil && !goja.IsUndefined(meta) {
				metaObj := meta.ToObject(vm)
				pair.Metadata = make(map[string]string)
				for _, k := range metaObj.Keys() {
					v := metaObj.Get(k)
					if v != nil && !goja.IsUndefined(v) {
						pair.Metadata[k] = v.String()
					}
				}
			}
		}

		// Store
		storePair := &store.KVPair{
			Key:        pair.Key,
			Value:      pair.Value,
			Metadata:   pair.Metadata,
			Expiration: pair.Expiration,
		}
		err := kvStore.Put(context.Background(), namespaceID, storePair)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		return r.createPromise(goja.Undefined())
	})

	// delete(key)
	kv.Set("delete", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()

		err := kvStore.Delete(context.Background(), namespaceID, key)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		return r.createPromise(goja.Undefined())
	})

	// list(options?)
	kv.Set("list", func(call goja.FunctionCall) goja.Value {
		prefix := ""
		limit := 1000

		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			opts := call.Arguments[0].ToObject(vm)

			if p := opts.Get("prefix"); p != nil && !goja.IsUndefined(p) {
				prefix = p.String()
			}
			if l := opts.Get("limit"); l != nil && !goja.IsUndefined(l) {
				limit = int(l.ToInteger())
			}
		}

		pairs, err := kvStore.List(context.Background(), namespaceID, prefix, limit)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		keys := make([]map[string]interface{}, 0, len(pairs))
		for _, pair := range pairs {
			keyObj := map[string]interface{}{
				"name": pair.Key,
			}
			if pair.Expiration != nil {
				keyObj["expiration"] = pair.Expiration.Unix()
			}
			if pair.Metadata != nil {
				keyObj["metadata"] = pair.Metadata
			}
			keys = append(keys, keyObj)
		}

		result := vm.NewObject()
		result.Set("keys", keys)
		result.Set("list_complete", len(pairs) < limit)
		if len(pairs) > 0 && len(pairs) >= limit {
			result.Set("cursor", pairs[len(pairs)-1].Key)
		}

		return r.createPromise(result)
	})

	r.vm.Set(name, kv)
}

type kvPair struct {
	Key        string
	Value      []byte
	Metadata   map[string]string
	Expiration *time.Time
}

// setupR2Binding creates an R2 bucket binding.
func (r *Runtime) setupR2Binding(name, bucketID string) {
	vm := r.vm
	r2Store := r.store.R2()

	r2 := vm.NewObject()

	// get(key)
	r2.Set("get", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()

		data, obj, err := r2Store.GetObject(context.Background(), bucketID, key)
		if err != nil || obj == nil {
			return r.createPromise(goja.Null())
		}

		result := vm.NewObject()
		result.Set("key", obj.Key)
		result.Set("size", obj.Size)
		result.Set("etag", obj.ETag)
		result.Set("httpEtag", `"`+obj.ETag+`"`)
		result.Set("uploaded", obj.LastModified.Format(time.RFC3339))

		// Custom metadata
		if obj.Metadata != nil {
			customMeta := vm.NewObject()
			for k, v := range obj.Metadata {
				customMeta.Set(k, v)
			}
			result.Set("customMetadata", customMeta)
		}

		// Body methods
		result.Set("text", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue(string(data)))
		})
		result.Set("json", func(c goja.FunctionCall) goja.Value {
			var jsonData interface{}
			json.Unmarshal(data, &jsonData)
			return r.createPromise(vm.ToValue(jsonData))
		})
		result.Set("arrayBuffer", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue(data))
		})
		result.Set("body", data)

		return r.createPromise(result)
	})

	// put(key, value, options?)
	r2.Set("put", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		value := call.Argument(1)

		var valueBytes []byte
		switch v := value.Export().(type) {
		case string:
			valueBytes = []byte(v)
		case []byte:
			valueBytes = v
		default:
			jsonBytes, _ := json.Marshal(v)
			valueBytes = jsonBytes
		}

		var metadata map[string]string
		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) {
			opts := call.Arguments[2].ToObject(vm)
			if cm := opts.Get("customMetadata"); cm != nil && !goja.IsUndefined(cm) {
				metaObj := cm.ToObject(vm)
				metadata = make(map[string]string)
				for _, k := range metaObj.Keys() {
					v := metaObj.Get(k)
					if v != nil && !goja.IsUndefined(v) {
						metadata[k] = v.String()
					}
				}
			}
		}

		err := r2Store.PutObject(context.Background(), bucketID, key, valueBytes, metadata)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		// Return object metadata
		result := vm.NewObject()
		result.Set("key", key)
		result.Set("size", len(valueBytes))
		result.Set("uploaded", time.Now().Format(time.RFC3339))

		return r.createPromise(result)
	})

	// delete(key)
	r2.Set("delete", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()

		err := r2Store.DeleteObject(context.Background(), bucketID, key)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		return r.createPromise(goja.Undefined())
	})

	// list(options?)
	r2.Set("list", func(call goja.FunctionCall) goja.Value {
		prefix := ""
		delimiter := ""
		limit := 1000

		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			opts := call.Arguments[0].ToObject(vm)

			if p := opts.Get("prefix"); p != nil && !goja.IsUndefined(p) {
				prefix = p.String()
			}
			if d := opts.Get("delimiter"); d != nil && !goja.IsUndefined(d) {
				delimiter = d.String()
			}
			if l := opts.Get("limit"); l != nil && !goja.IsUndefined(l) {
				limit = int(l.ToInteger())
			}
		}

		objects, err := r2Store.ListObjects(context.Background(), bucketID, prefix, delimiter, limit)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		objectsList := make([]map[string]interface{}, 0, len(objects))
		for _, obj := range objects {
			objectsList = append(objectsList, map[string]interface{}{
				"key":      obj.Key,
				"size":     obj.Size,
				"etag":     obj.ETag,
				"uploaded": obj.LastModified.Format(time.RFC3339),
			})
		}

		result := vm.NewObject()
		result.Set("objects", objectsList)
		result.Set("truncated", len(objects) >= limit)

		return r.createPromise(result)
	})

	// head(key)
	r2.Set("head", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()

		_, obj, err := r2Store.GetObject(context.Background(), bucketID, key)
		if err != nil || obj == nil {
			return r.createPromise(goja.Null())
		}

		result := vm.NewObject()
		result.Set("key", obj.Key)
		result.Set("size", obj.Size)
		result.Set("etag", obj.ETag)
		result.Set("httpEtag", `"`+obj.ETag+`"`)
		result.Set("uploaded", obj.LastModified.Format(time.RFC3339))

		return r.createPromise(result)
	})

	r.vm.Set(name, r2)
}

// setupD1Binding creates a D1 database binding.
func (r *Runtime) setupD1Binding(name, databaseID string) {
	vm := r.vm
	d1Store := r.store.D1()

	d1 := vm.NewObject()

	// prepare(sql)
	d1.Set("prepare", func(call goja.FunctionCall) goja.Value {
		sql := call.Argument(0).String()

		stmt := vm.NewObject()
		var bindParams []interface{}

		// bind(...params)
		stmt.Set("bind", func(c goja.FunctionCall) goja.Value {
			for _, arg := range c.Arguments {
				bindParams = append(bindParams, arg.Export())
			}
			return stmt
		})

		// first(columnName?)
		stmt.Set("first", func(c goja.FunctionCall) goja.Value {
			results, err := d1Store.Query(context.Background(), databaseID, sql, bindParams)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			if len(results) == 0 {
				return r.createPromise(goja.Null())
			}

			row := results[0]
			if len(c.Arguments) > 0 && !goja.IsUndefined(c.Arguments[0]) {
				colName := c.Argument(0).String()
				if val, ok := row[colName]; ok {
					return r.createPromise(vm.ToValue(val))
				}
				return r.createPromise(goja.Null())
			}

			return r.createPromise(vm.ToValue(row))
		})

		// all()
		stmt.Set("all", func(c goja.FunctionCall) goja.Value {
			results, err := d1Store.Query(context.Background(), databaseID, sql, bindParams)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			result := vm.NewObject()
			result.Set("results", results)
			result.Set("success", true)
			result.Set("meta", map[string]interface{}{
				"rows_read":    len(results),
				"rows_written": 0,
			})

			return r.createPromise(result)
		})

		// run()
		stmt.Set("run", func(c goja.FunctionCall) goja.Value {
			rowsAffected, err := d1Store.Exec(context.Background(), databaseID, sql, bindParams)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			result := vm.NewObject()
			result.Set("success", true)
			result.Set("meta", map[string]interface{}{
				"rows_read":    0,
				"rows_written": rowsAffected,
				"changes":      rowsAffected,
			})

			return r.createPromise(result)
		})

		// raw(options?)
		stmt.Set("raw", func(c goja.FunctionCall) goja.Value {
			results, err := d1Store.Query(context.Background(), databaseID, sql, bindParams)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			// Convert to array of arrays
			var rows [][]interface{}
			for _, row := range results {
				var rowArr []interface{}
				for _, v := range row {
					rowArr = append(rowArr, v)
				}
				rows = append(rows, rowArr)
			}

			return r.createPromise(vm.ToValue(rows))
		})

		return stmt
	})

	// exec(sql)
	d1.Set("exec", func(call goja.FunctionCall) goja.Value {
		sql := call.Argument(0).String()

		rowsAffected, err := d1Store.Exec(context.Background(), databaseID, sql, nil)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		result := vm.NewObject()
		result.Set("success", true)
		result.Set("changes", rowsAffected)

		return r.createPromise(result)
	})

	// batch(statements)
	d1.Set("batch", func(call goja.FunctionCall) goja.Value {
		// For batch, we'd need to execute multiple statements
		// This is a simplified implementation
		stmts := call.Argument(0).Export().([]interface{})

		var results []map[string]interface{}
		for _, stmt := range stmts {
			if stmtObj, ok := stmt.(map[string]interface{}); ok {
				if sql, ok := stmtObj["sql"].(string); ok {
					res, err := d1Store.Query(context.Background(), databaseID, sql, nil)
					if err != nil {
						return r.createRejectedPromise(err.Error())
					}
					results = append(results, map[string]interface{}{
						"results": res,
						"success": true,
					})
				}
			}
		}

		return r.createPromise(vm.ToValue(results))
	})

	// dump() - returns database as bytes
	d1.Set("dump", func(call goja.FunctionCall) goja.Value {
		return r.createPromise(vm.ToValue([]byte{}))
	})

	r.vm.Set(name, d1)
}

// setupDOBinding creates a Durable Object namespace binding.
func (r *Runtime) setupDOBinding(name, namespaceID string) {
	vm := r.vm
	doStore := r.store.DurableObjects()

	doNS := vm.NewObject()

	// idFromName(name)
	doNS.Set("idFromName", func(call goja.FunctionCall) goja.Value {
		objName := call.Argument(0).String()

		id := vm.NewObject()
		id.Set("toString", func(c goja.FunctionCall) goja.Value {
			return vm.ToValue("do:" + namespaceID + ":" + objName)
		})
		id.Set("name", objName)
		id.Set("_namespaceID", namespaceID)
		id.Set("_objectName", objName)

		return id
	})

	// idFromString(hexId)
	doNS.Set("idFromString", func(call goja.FunctionCall) goja.Value {
		hexID := call.Argument(0).String()

		id := vm.NewObject()
		id.Set("toString", func(c goja.FunctionCall) goja.Value {
			return vm.ToValue(hexID)
		})
		id.Set("_namespaceID", namespaceID)
		id.Set("_objectID", hexID)

		return id
	})

	// newUniqueId()
	doNS.Set("newUniqueId", func(call goja.FunctionCall) goja.Value {
		uniqueID := generateUUID()

		id := vm.NewObject()
		id.Set("toString", func(c goja.FunctionCall) goja.Value {
			return vm.ToValue(uniqueID)
		})
		id.Set("_namespaceID", namespaceID)
		id.Set("_objectID", uniqueID)

		return id
	})

	// get(id)
	doNS.Set("get", func(call goja.FunctionCall) goja.Value {
		idObj := call.Argument(0).ToObject(vm)

		objectName := ""
		objectID := ""

		if n := idObj.Get("_objectName"); n != nil && !goja.IsUndefined(n) {
			objectName = n.String()
		}
		if i := idObj.Get("_objectID"); i != nil && !goja.IsUndefined(i) {
			objectID = i.String()
		}

		// Get or create the instance
		instance, err := doStore.GetOrCreateInstance(context.Background(), namespaceID, objectID, objectName)
		if err != nil {
			return goja.Null()
		}

		// Create stub
		stub := vm.NewObject()
		stub.Set("id", idObj)
		stub.Set("name", objectName)

		// fetch(request, options?)
		stub.Set("fetch", func(c goja.FunctionCall) goja.Value {
			// This would typically forward the request to the DO instance
			// For now, we return a simple response
			resp := vm.NewObject()
			resp.Set("status", 200)
			resp.Set("ok", true)
			resp.Set("_body", "")
			resp.Set("text", func(fc goja.FunctionCall) goja.Value {
				return r.createPromise(vm.ToValue(""))
			})
			resp.Set("json", func(fc goja.FunctionCall) goja.Value {
				return r.createPromise(vm.ToValue(map[string]interface{}{}))
			})

			return r.createPromise(resp)
		})

		// Storage
		storage := vm.NewObject()

		storage.Set("get", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			data, err := doStore.Get(context.Background(), instance.ID, key)
			if err != nil || data == nil {
				return r.createPromise(goja.Undefined())
			}

			var value interface{}
			json.Unmarshal(data, &value)
			return r.createPromise(vm.ToValue(value))
		})

		storage.Set("put", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			value := c.Argument(1).Export()

			data, _ := json.Marshal(value)
			err := doStore.Put(context.Background(), instance.ID, key, data)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			return r.createPromise(goja.Undefined())
		})

		storage.Set("delete", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			err := doStore.Delete(context.Background(), instance.ID, key)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}
			return r.createPromise(vm.ToValue(true))
		})

		storage.Set("deleteAll", func(c goja.FunctionCall) goja.Value {
			err := doStore.DeleteAll(context.Background(), instance.ID)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}
			return r.createPromise(goja.Undefined())
		})

		storage.Set("list", func(c goja.FunctionCall) goja.Value {
			entries, err := doStore.List(context.Background(), instance.ID, nil)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			result := vm.NewObject()
			for k, v := range entries {
				var value interface{}
				json.Unmarshal(v, &value)
				result.Set(k, value)
			}

			return r.createPromise(result)
		})

		stub.Set("storage", storage)

		return stub
	})

	r.vm.Set(name, doNS)
}

// setupQueueBinding creates a Queue binding.
func (r *Runtime) setupQueueBinding(name, queueID string) {
	vm := r.vm
	queueStore := r.store.Queues()

	queue := vm.NewObject()

	// send(message, options?)
	queue.Set("send", func(call goja.FunctionCall) goja.Value {
		message := call.Argument(0).Export()

		body, _ := json.Marshal(message)

		contentType := "json"
		delaySeconds := 0

		// Parse options
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			opts := call.Arguments[1].ToObject(vm)
			if ct := opts.Get("contentType"); ct != nil && !goja.IsUndefined(ct) {
				contentType = ct.String()
			}
			if delay := opts.Get("delaySeconds"); delay != nil && !goja.IsUndefined(delay) {
				delaySeconds = int(delay.ToInteger())
			}
		}

		now := time.Now()
		visibleAt := now
		if delaySeconds > 0 {
			visibleAt = now.Add(time.Duration(delaySeconds) * time.Second)
		}

		storeMsg := &store.QueueMessage{
			ID:          generateUUID(),
			QueueID:     queueID,
			Body:        body,
			ContentType: contentType,
			Attempts:    0,
			CreatedAt:   now,
			VisibleAt:   visibleAt,
			ExpiresAt:   now.Add(4 * 24 * time.Hour), // 4 days default TTL
		}
		err := queueStore.SendMessage(context.Background(), queueID, storeMsg)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		return r.createPromise(goja.Undefined())
	})

	// sendBatch(messages)
	queue.Set("sendBatch", func(call goja.FunctionCall) goja.Value {
		messages := call.Argument(0).Export().([]interface{})

		now := time.Now()
		var storeMsgs []*store.QueueMessage
		for _, m := range messages {
			msgObj := m.(map[string]interface{})
			body := msgObj["body"]

			bodyBytes, _ := json.Marshal(body)

			contentType := "json"
			if ct, ok := msgObj["contentType"].(string); ok {
				contentType = ct
			}

			msg := &store.QueueMessage{
				ID:          generateUUID(),
				QueueID:     queueID,
				Body:        bodyBytes,
				ContentType: contentType,
				Attempts:    0,
				CreatedAt:   now,
				VisibleAt:   now,
				ExpiresAt:   now.Add(4 * 24 * time.Hour),
			}

			storeMsgs = append(storeMsgs, msg)
		}

		err := queueStore.SendBatch(context.Background(), queueID, storeMsgs)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		return r.createPromise(goja.Undefined())
	})

	r.vm.Set(name, queue)
}

// setupAIBinding creates an AI binding.
func (r *Runtime) setupAIBinding(name string) {
	vm := r.vm
	aiStore := r.store.AI()

	ai := vm.NewObject()

	// run(model, inputs, options?)
	ai.Set("run", func(call goja.FunctionCall) goja.Value {
		model := call.Argument(0).String()
		inputs := call.Argument(1).Export()

		inputsMap, ok := inputs.(map[string]interface{})
		if !ok {
			inputsMap = map[string]interface{}{"input": inputs}
		}

		// Check if it's embeddings
		if text, ok := inputsMap["text"]; ok {
			var texts []string
			switch v := text.(type) {
			case string:
				texts = []string{v}
			case []interface{}:
				for _, t := range v {
					texts = append(texts, t.(string))
				}
			}

			embeddings, err := aiStore.GenerateEmbeddings(context.Background(), model, texts)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			result := vm.NewObject()
			result.Set("shape", []int{len(embeddings), len(embeddings[0])})
			result.Set("data", embeddings)

			return r.createPromise(result)
		}

		// Text generation
		prompt := ""
		if p, ok := inputsMap["prompt"].(string); ok {
			prompt = p
		} else if msgs, ok := inputsMap["messages"].([]interface{}); ok {
			for _, m := range msgs {
				if msgMap, ok := m.(map[string]interface{}); ok {
					if content, ok := msgMap["content"].(string); ok {
						prompt += content + "\n"
					}
				}
			}
		}

		var opts map[string]interface{}
		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) {
			opts = call.Arguments[2].Export().(map[string]interface{})
		}

		text, err := aiStore.GenerateText(context.Background(), model, prompt, opts)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		result := vm.NewObject()
		result.Set("response", text)

		return r.createPromise(result)
	})

	r.vm.Set(name, ai)
}
