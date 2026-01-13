package runtime

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dop251/goja"

	"github.com/go-mizu/blueprints/localflare/store"
)

// decodeHex decodes a hex string to bytes, returning nil on error.
func decodeHex(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

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

// setupR2Binding creates an R2 bucket binding with full Cloudflare Workers API compatibility.
func (r *Runtime) setupR2Binding(name, bucketID string) {
	vm := r.vm
	r2Store := r.store.R2()

	r2 := vm.NewObject()

	// Helper to create R2Object JavaScript object from store.R2Object
	createR2Object := func(obj *store.R2Object) goja.Value {
		result := vm.NewObject()
		result.Set("key", obj.Key)
		result.Set("version", obj.Version)
		result.Set("size", obj.Size)
		result.Set("etag", obj.ETag)
		result.Set("httpEtag", `"`+obj.ETag+`"`)
		result.Set("storageClass", obj.StorageClass)

		// Create Date object for uploaded
		dateStr := obj.Uploaded.Format(time.RFC3339)
		dateJS, _ := vm.RunString(fmt.Sprintf("new Date('%s')", dateStr))
		result.Set("uploaded", dateJS)

		// HTTP metadata
		if obj.HTTPMetadata != nil {
			httpMeta := vm.NewObject()
			if obj.HTTPMetadata.ContentType != "" {
				httpMeta.Set("contentType", obj.HTTPMetadata.ContentType)
			}
			if obj.HTTPMetadata.ContentLanguage != "" {
				httpMeta.Set("contentLanguage", obj.HTTPMetadata.ContentLanguage)
			}
			if obj.HTTPMetadata.ContentDisposition != "" {
				httpMeta.Set("contentDisposition", obj.HTTPMetadata.ContentDisposition)
			}
			if obj.HTTPMetadata.ContentEncoding != "" {
				httpMeta.Set("contentEncoding", obj.HTTPMetadata.ContentEncoding)
			}
			if obj.HTTPMetadata.CacheControl != "" {
				httpMeta.Set("cacheControl", obj.HTTPMetadata.CacheControl)
			}
			if obj.HTTPMetadata.CacheExpiry != nil {
				expiryStr := obj.HTTPMetadata.CacheExpiry.Format(time.RFC3339)
				expiryJS, _ := vm.RunString(fmt.Sprintf("new Date('%s')", expiryStr))
				httpMeta.Set("cacheExpiry", expiryJS)
			}
			result.Set("httpMetadata", httpMeta)
		}

		// Custom metadata
		if obj.CustomMetadata != nil {
			customMeta := vm.NewObject()
			for k, v := range obj.CustomMetadata {
				customMeta.Set(k, v)
			}
			result.Set("customMetadata", customMeta)
		}

		// Range (if partial read)
		if obj.Range != nil {
			rangeObj := vm.NewObject()
			if obj.Range.Offset != nil {
				rangeObj.Set("offset", *obj.Range.Offset)
			}
			if obj.Range.Length != nil {
				rangeObj.Set("length", *obj.Range.Length)
			}
			result.Set("range", rangeObj)
		}

		// Checksums
		if obj.Checksums != nil {
			checksums := vm.NewObject()
			if obj.Checksums.MD5 != nil {
				checksums.Set("md5", fmt.Sprintf("%x", obj.Checksums.MD5))
			}
			if obj.Checksums.SHA1 != nil {
				checksums.Set("sha1", fmt.Sprintf("%x", obj.Checksums.SHA1))
			}
			if obj.Checksums.SHA256 != nil {
				checksums.Set("sha256", fmt.Sprintf("%x", obj.Checksums.SHA256))
			}
			if obj.Checksums.SHA384 != nil {
				checksums.Set("sha384", fmt.Sprintf("%x", obj.Checksums.SHA384))
			}
			if obj.Checksums.SHA512 != nil {
				checksums.Set("sha512", fmt.Sprintf("%x", obj.Checksums.SHA512))
			}
			result.Set("checksums", checksums)
		}

		// writeHttpMetadata(headers) method
		result.Set("writeHttpMetadata", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) == 0 {
				return goja.Undefined()
			}
			headers := call.Arguments[0].ToObject(vm)
			setFunc := headers.Get("set")
			if setFunc != nil && !goja.IsUndefined(setFunc) {
				if fn, ok := goja.AssertFunction(setFunc); ok {
					if obj.HTTPMetadata != nil {
						if obj.HTTPMetadata.ContentType != "" {
							fn(headers, vm.ToValue("content-type"), vm.ToValue(obj.HTTPMetadata.ContentType))
						}
						if obj.HTTPMetadata.ContentLanguage != "" {
							fn(headers, vm.ToValue("content-language"), vm.ToValue(obj.HTTPMetadata.ContentLanguage))
						}
						if obj.HTTPMetadata.ContentDisposition != "" {
							fn(headers, vm.ToValue("content-disposition"), vm.ToValue(obj.HTTPMetadata.ContentDisposition))
						}
						if obj.HTTPMetadata.ContentEncoding != "" {
							fn(headers, vm.ToValue("content-encoding"), vm.ToValue(obj.HTTPMetadata.ContentEncoding))
						}
						if obj.HTTPMetadata.CacheControl != "" {
							fn(headers, vm.ToValue("cache-control"), vm.ToValue(obj.HTTPMetadata.CacheControl))
						}
					}
				}
			}
			return goja.Undefined()
		})

		return result
	}

	// Helper to create R2ObjectBody (extends R2Object with body methods)
	createR2ObjectBody := func(obj *store.R2Object, data []byte) goja.Value {
		result := createR2Object(obj).ToObject(vm)
		bodyUsed := false

		// body property (ReadableStream-like, simplified as bytes)
		result.Set("body", data)
		result.Set("bodyUsed", false)

		// text() method
		result.Set("text", func(c goja.FunctionCall) goja.Value {
			bodyUsed = true
			result.Set("bodyUsed", true)
			return r.createPromise(vm.ToValue(string(data)))
		})

		// json() method
		result.Set("json", func(c goja.FunctionCall) goja.Value {
			bodyUsed = true
			result.Set("bodyUsed", true)
			var jsonData interface{}
			json.Unmarshal(data, &jsonData)
			return r.createPromise(vm.ToValue(jsonData))
		})

		// arrayBuffer() method
		result.Set("arrayBuffer", func(c goja.FunctionCall) goja.Value {
			bodyUsed = true
			result.Set("bodyUsed", true)
			return r.createPromise(vm.ToValue(data))
		})

		// blob() method
		result.Set("blob", func(c goja.FunctionCall) goja.Value {
			bodyUsed = true
			result.Set("bodyUsed", true)
			contentType := "application/octet-stream"
			if obj.HTTPMetadata != nil && obj.HTTPMetadata.ContentType != "" {
				contentType = obj.HTTPMetadata.ContentType
			}
			blobObj := vm.NewObject()
			blobObj.Set("size", len(data))
			blobObj.Set("type", contentType)
			blobObj.Set("_data", data)
			blobObj.Set("text", func(call goja.FunctionCall) goja.Value {
				return r.createPromise(vm.ToValue(string(data)))
			})
			blobObj.Set("arrayBuffer", func(call goja.FunctionCall) goja.Value {
				return r.createPromise(vm.ToValue(data))
			})
			return r.createPromise(blobObj)
		})

		_ = bodyUsed // Suppress unused warning
		return result
	}

	// get(key, options?) - Returns R2ObjectBody | R2Object | null
	r2.Set("get", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()

		// Parse options
		var opts *store.R2GetOptions
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			optsArg := call.Arguments[1].ToObject(vm)
			opts = &store.R2GetOptions{}

			// onlyIf conditional
			if onlyIf := optsArg.Get("onlyIf"); onlyIf != nil && !goja.IsUndefined(onlyIf) {
				onlyIfObj := onlyIf.ToObject(vm)
				opts.OnlyIf = &store.R2Conditional{}
				if em := onlyIfObj.Get("etagMatches"); em != nil && !goja.IsUndefined(em) {
					opts.OnlyIf.EtagMatches = em.String()
				}
				if edm := onlyIfObj.Get("etagDoesNotMatch"); edm != nil && !goja.IsUndefined(edm) {
					opts.OnlyIf.EtagDoesNotMatch = edm.String()
				}
				if ub := onlyIfObj.Get("uploadedBefore"); ub != nil && !goja.IsUndefined(ub) {
					// JavaScript Date objects export to time.Time directly
					if t, ok := ub.Export().(time.Time); ok {
						opts.OnlyIf.UploadedBefore = &t
					} else if t, err := time.Parse(time.RFC3339, ub.String()); err == nil {
						opts.OnlyIf.UploadedBefore = &t
					}
				}
				if ua := onlyIfObj.Get("uploadedAfter"); ua != nil && !goja.IsUndefined(ua) {
					// JavaScript Date objects export to time.Time directly
					if t, ok := ua.Export().(time.Time); ok {
						opts.OnlyIf.UploadedAfter = &t
					} else if t, err := time.Parse(time.RFC3339, ua.String()); err == nil {
						opts.OnlyIf.UploadedAfter = &t
					}
				}
			}

			// range for partial reads
			if rangeOpt := optsArg.Get("range"); rangeOpt != nil && !goja.IsUndefined(rangeOpt) {
				rangeObj := rangeOpt.ToObject(vm)
				opts.Range = &store.R2Range{}
				if offset := rangeObj.Get("offset"); offset != nil && !goja.IsUndefined(offset) {
					o := offset.ToInteger()
					opts.Range.Offset = &o
				}
				if length := rangeObj.Get("length"); length != nil && !goja.IsUndefined(length) {
					l := length.ToInteger()
					opts.Range.Length = &l
				}
				if suffix := rangeObj.Get("suffix"); suffix != nil && !goja.IsUndefined(suffix) {
					s := suffix.ToInteger()
					opts.Range.Suffix = &s
				}
			}
		}

		data, obj, err := r2Store.GetObject(context.Background(), bucketID, key, opts)
		if err != nil {
			return r.createPromise(goja.Null())
		}
		if obj == nil {
			return r.createPromise(goja.Null())
		}

		// If data is nil but obj exists, conditional failed - return metadata only
		if data == nil {
			return r.createPromise(createR2Object(obj))
		}

		return r.createPromise(createR2ObjectBody(obj, data))
	})

	// put(key, value, options?) - Returns R2Object | null
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

		// Parse options
		var opts *store.R2PutOptions
		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) {
			optsArg := call.Arguments[2].ToObject(vm)
			opts = &store.R2PutOptions{}

			// httpMetadata
			if hm := optsArg.Get("httpMetadata"); hm != nil && !goja.IsUndefined(hm) {
				hmObj := hm.ToObject(vm)
				opts.HTTPMetadata = &store.R2HTTPMetadata{}
				if ct := hmObj.Get("contentType"); ct != nil && !goja.IsUndefined(ct) {
					opts.HTTPMetadata.ContentType = ct.String()
				}
				if cl := hmObj.Get("contentLanguage"); cl != nil && !goja.IsUndefined(cl) {
					opts.HTTPMetadata.ContentLanguage = cl.String()
				}
				if cd := hmObj.Get("contentDisposition"); cd != nil && !goja.IsUndefined(cd) {
					opts.HTTPMetadata.ContentDisposition = cd.String()
				}
				if ce := hmObj.Get("contentEncoding"); ce != nil && !goja.IsUndefined(ce) {
					opts.HTTPMetadata.ContentEncoding = ce.String()
				}
				if cc := hmObj.Get("cacheControl"); cc != nil && !goja.IsUndefined(cc) {
					opts.HTTPMetadata.CacheControl = cc.String()
				}
			}

			// customMetadata
			if cm := optsArg.Get("customMetadata"); cm != nil && !goja.IsUndefined(cm) {
				metaObj := cm.ToObject(vm)
				opts.CustomMetadata = make(map[string]string)
				for _, k := range metaObj.Keys() {
					v := metaObj.Get(k)
					if v != nil && !goja.IsUndefined(v) {
						opts.CustomMetadata[k] = v.String()
					}
				}
			}

			// storageClass
			if sc := optsArg.Get("storageClass"); sc != nil && !goja.IsUndefined(sc) {
				opts.StorageClass = sc.String()
			}

			// onlyIf conditional
			if onlyIf := optsArg.Get("onlyIf"); onlyIf != nil && !goja.IsUndefined(onlyIf) {
				onlyIfObj := onlyIf.ToObject(vm)
				opts.OnlyIf = &store.R2Conditional{}
				if em := onlyIfObj.Get("etagMatches"); em != nil && !goja.IsUndefined(em) {
					opts.OnlyIf.EtagMatches = em.String()
				}
				if edm := onlyIfObj.Get("etagDoesNotMatch"); edm != nil && !goja.IsUndefined(edm) {
					opts.OnlyIf.EtagDoesNotMatch = edm.String()
				}
			}

			// Checksums (md5, sha1, sha256, sha384, sha512)
			if md5Opt := optsArg.Get("md5"); md5Opt != nil && !goja.IsUndefined(md5Opt) {
				if bytes, err := decodeHex(md5Opt.String()); err == nil {
					opts.MD5 = bytes
				}
			}
			if sha1Opt := optsArg.Get("sha1"); sha1Opt != nil && !goja.IsUndefined(sha1Opt) {
				if bytes, err := decodeHex(sha1Opt.String()); err == nil {
					opts.SHA1 = bytes
				}
			}
			if sha256Opt := optsArg.Get("sha256"); sha256Opt != nil && !goja.IsUndefined(sha256Opt) {
				if bytes, err := decodeHex(sha256Opt.String()); err == nil {
					opts.SHA256 = bytes
				}
			}
			if sha384Opt := optsArg.Get("sha384"); sha384Opt != nil && !goja.IsUndefined(sha384Opt) {
				if bytes, err := decodeHex(sha384Opt.String()); err == nil {
					opts.SHA384 = bytes
				}
			}
			if sha512Opt := optsArg.Get("sha512"); sha512Opt != nil && !goja.IsUndefined(sha512Opt) {
				if bytes, err := decodeHex(sha512Opt.String()); err == nil {
					opts.SHA512 = bytes
				}
			}
		}

		obj, err := r2Store.PutObject(context.Background(), bucketID, key, valueBytes, opts)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		return r.createPromise(createR2Object(obj))
	})

	// delete(key | keys[]) - Batch delete up to 1000 keys
	r2.Set("delete", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)

		// Check if it's an array
		if arr, ok := arg.Export().([]interface{}); ok {
			keys := make([]string, 0, len(arr))
			for _, k := range arr {
				keys = append(keys, fmt.Sprintf("%v", k))
			}
			_, err := r2Store.DeleteObjects(context.Background(), bucketID, keys)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}
			return r.createPromise(goja.Undefined())
		}

		// Single key
		key := arg.String()
		err := r2Store.DeleteObject(context.Background(), bucketID, key)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}
		return r.createPromise(goja.Undefined())
	})

	// list(options?) - Returns R2Objects
	r2.Set("list", func(call goja.FunctionCall) goja.Value {
		opts := &store.R2ListOptions{Limit: 1000}

		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			optsArg := call.Arguments[0].ToObject(vm)

			if p := optsArg.Get("prefix"); p != nil && !goja.IsUndefined(p) {
				opts.Prefix = p.String()
			}
			if d := optsArg.Get("delimiter"); d != nil && !goja.IsUndefined(d) {
				opts.Delimiter = d.String()
			}
			if l := optsArg.Get("limit"); l != nil && !goja.IsUndefined(l) {
				opts.Limit = int(l.ToInteger())
			}
			if c := optsArg.Get("cursor"); c != nil && !goja.IsUndefined(c) {
				opts.Cursor = c.String()
			}
			if inc := optsArg.Get("include"); inc != nil && !goja.IsUndefined(inc) {
				if incArr, ok := inc.Export().([]interface{}); ok {
					for _, i := range incArr {
						opts.Include = append(opts.Include, fmt.Sprintf("%v", i))
					}
				}
			}
		}

		listResult, err := r2Store.ListObjects(context.Background(), bucketID, opts)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		objectsList := make([]interface{}, 0, len(listResult.Objects))
		for _, obj := range listResult.Objects {
			objectsList = append(objectsList, createR2Object(obj).Export())
		}

		result := vm.NewObject()
		result.Set("objects", objectsList)
		result.Set("truncated", listResult.Truncated)
		if listResult.Cursor != "" {
			result.Set("cursor", listResult.Cursor)
		}
		// Always set delimitedPrefixes as an array (empty if no prefixes)
		if len(listResult.DelimitedPrefixes) > 0 {
			result.Set("delimitedPrefixes", listResult.DelimitedPrefixes)
		} else {
			result.Set("delimitedPrefixes", []string{})
		}

		return r.createPromise(result)
	})

	// head(key) - Returns R2Object | null (metadata only, no body)
	r2.Set("head", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()

		obj, err := r2Store.HeadObject(context.Background(), bucketID, key)
		if err != nil || obj == nil {
			return r.createPromise(goja.Null())
		}

		return r.createPromise(createR2Object(obj))
	})

	// createMultipartUpload(key, options?) - Returns R2MultipartUpload
	r2.Set("createMultipartUpload", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()

		var opts *store.R2PutOptions
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			optsArg := call.Arguments[1].ToObject(vm)
			opts = &store.R2PutOptions{}

			if hm := optsArg.Get("httpMetadata"); hm != nil && !goja.IsUndefined(hm) {
				hmObj := hm.ToObject(vm)
				opts.HTTPMetadata = &store.R2HTTPMetadata{}
				if ct := hmObj.Get("contentType"); ct != nil && !goja.IsUndefined(ct) {
					opts.HTTPMetadata.ContentType = ct.String()
				}
			}
			if cm := optsArg.Get("customMetadata"); cm != nil && !goja.IsUndefined(cm) {
				metaObj := cm.ToObject(vm)
				opts.CustomMetadata = make(map[string]string)
				for _, k := range metaObj.Keys() {
					v := metaObj.Get(k)
					if v != nil && !goja.IsUndefined(v) {
						opts.CustomMetadata[k] = v.String()
					}
				}
			}
		}

		mpu, err := r2Store.CreateMultipartUpload(context.Background(), bucketID, key, opts)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		return r.createPromise(r.createMultipartUploadObject(bucketID, mpu))
	})

	// resumeMultipartUpload(key, uploadId) - Returns R2MultipartUpload (synchronous)
	r2.Set("resumeMultipartUpload", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		uploadID := call.Argument(1).String()

		mpu := &store.R2MultipartUpload{
			Key:      key,
			UploadID: uploadID,
		}
		return r.createMultipartUploadObject(bucketID, mpu)
	})

	r.vm.Set(name, r2)
}

// createMultipartUploadObject creates a JavaScript R2MultipartUpload object
func (r *Runtime) createMultipartUploadObject(bucketID string, mpu *store.R2MultipartUpload) goja.Value {
	vm := r.vm
	r2Store := r.store.R2()

	mpuObj := vm.NewObject()
	mpuObj.Set("key", mpu.Key)
	mpuObj.Set("uploadId", mpu.UploadID)

	// uploadPart(partNumber, value)
	mpuObj.Set("uploadPart", func(call goja.FunctionCall) goja.Value {
		partNumber := int(call.Argument(0).ToInteger())
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

		part, err := r2Store.UploadPart(context.Background(), bucketID, mpu.Key, mpu.UploadID, partNumber, valueBytes)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		partObj := vm.NewObject()
		partObj.Set("partNumber", part.PartNumber)
		partObj.Set("etag", part.ETag)

		return r.createPromise(partObj)
	})

	// abort()
	mpuObj.Set("abort", func(call goja.FunctionCall) goja.Value {
		err := r2Store.AbortMultipartUpload(context.Background(), bucketID, mpu.Key, mpu.UploadID)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}
		return r.createPromise(goja.Undefined())
	})

	// complete(uploadedParts)
	mpuObj.Set("complete", func(call goja.FunctionCall) goja.Value {
		partsArg := call.Argument(0)
		partsArr, ok := partsArg.Export().([]interface{})
		if !ok {
			return r.createRejectedPromise("uploadedParts must be an array")
		}

		parts := make([]*store.R2UploadedPart, 0, len(partsArr))
		for _, p := range partsArr {
			if pMap, ok := p.(map[string]interface{}); ok {
				part := &store.R2UploadedPart{}
				if pn, ok := pMap["partNumber"].(int64); ok {
					part.PartNumber = int(pn)
				} else if pn, ok := pMap["partNumber"].(float64); ok {
					part.PartNumber = int(pn)
				}
				if etag, ok := pMap["etag"].(string); ok {
					part.ETag = etag
				}
				parts = append(parts, part)
			}
		}

		obj, err := r2Store.CompleteMultipartUpload(context.Background(), bucketID, mpu.Key, mpu.UploadID, parts)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		result := vm.NewObject()
		result.Set("key", obj.Key)
		result.Set("version", obj.Version)
		result.Set("size", obj.Size)
		result.Set("etag", obj.ETag)
		result.Set("httpEtag", `"`+obj.ETag+`"`)
		result.Set("storageClass", obj.StorageClass)
		dateStr := obj.Uploaded.Format(time.RFC3339)
		dateJS, _ := vm.RunString(fmt.Sprintf("new Date('%s')", dateStr))
		result.Set("uploaded", dateJS)

		return r.createPromise(result)
	})

	return mpuObj
}

// setupD1Binding creates a D1 database binding with 100% Cloudflare D1 compatibility.
func (r *Runtime) setupD1Binding(name, databaseID string) {
	vm := r.vm
	d1Store := r.store.D1()

	d1 := vm.NewObject()

	// Helper to convert D1ResultMeta to JS object
	createMetaObject := func(meta store.D1ResultMeta) goja.Value {
		metaObj := vm.NewObject()
		metaObj.Set("served_by", meta.ServedBy)
		metaObj.Set("duration", meta.Duration)
		metaObj.Set("changes", meta.Changes)
		metaObj.Set("last_row_id", meta.LastRowID)
		metaObj.Set("changed_db", meta.ChangedDB)
		metaObj.Set("size_after", meta.SizeAfter)
		metaObj.Set("rows_read", meta.RowsRead)
		metaObj.Set("rows_written", meta.RowsWritten)
		return metaObj
	}

	// prepare(sql) - creates a prepared statement
	d1.Set("prepare", func(call goja.FunctionCall) goja.Value {
		sql := call.Argument(0).String()

		stmt := vm.NewObject()
		var bindParams []interface{}

		// Store SQL and params for batch() to access
		stmt.Set("_sql", sql)
		stmt.Set("_getParams", func(c goja.FunctionCall) goja.Value {
			return vm.ToValue(bindParams)
		})

		// bind(...params) - binds parameters to placeholders
		stmt.Set("bind", func(c goja.FunctionCall) goja.Value {
			for _, arg := range c.Arguments {
				bindParams = append(bindParams, arg.Export())
			}
			// Update stored params
			stmt.Set("_getParams", func(c goja.FunctionCall) goja.Value {
				return vm.ToValue(bindParams)
			})
			return stmt
		})

		// first(columnName?) - returns first row or specific column value
		stmt.Set("first", func(c goja.FunctionCall) goja.Value {
			result, err := d1Store.QueryWithMeta(context.Background(), databaseID, sql, bindParams)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			if len(result.Rows) == 0 {
				return r.createPromise(goja.Null())
			}

			row := result.Rows[0]
			if len(c.Arguments) > 0 && !goja.IsUndefined(c.Arguments[0]) && !goja.IsNull(c.Arguments[0]) {
				colName := c.Argument(0).String()
				if val, ok := row[colName]; ok {
					return r.createPromise(vm.ToValue(val))
				}
				return r.createPromise(goja.Null())
			}

			return r.createPromise(vm.ToValue(row))
		})

		// all() - returns all results with metadata
		stmt.Set("all", func(c goja.FunctionCall) goja.Value {
			result, err := d1Store.QueryWithMeta(context.Background(), databaseID, sql, bindParams)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			// Ensure results is never nil (Cloudflare returns empty array)
			rows := result.Rows
			if rows == nil {
				rows = []map[string]interface{}{}
			}

			resultObj := vm.NewObject()
			resultObj.Set("results", rows)
			resultObj.Set("success", true)
			resultObj.Set("meta", createMetaObject(result.Meta))

			return r.createPromise(resultObj)
		})

		// run() - executes statement and returns metadata
		stmt.Set("run", func(c goja.FunctionCall) goja.Value {
			result, err := d1Store.ExecWithMeta(context.Background(), databaseID, sql, bindParams)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			resultObj := vm.NewObject()
			resultObj.Set("results", []interface{}{})
			resultObj.Set("success", true)
			resultObj.Set("meta", createMetaObject(result.Meta))

			return r.createPromise(resultObj)
		})

		// raw(options?) - returns results as array of arrays
		stmt.Set("raw", func(c goja.FunctionCall) goja.Value {
			result, err := d1Store.QueryWithMeta(context.Background(), databaseID, sql, bindParams)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			// Check if columnNames option is set
			includeColumnNames := false
			if len(c.Arguments) > 0 && !goja.IsUndefined(c.Arguments[0]) && !goja.IsNull(c.Arguments[0]) {
				if opts, ok := c.Argument(0).Export().(map[string]interface{}); ok {
					if cn, ok := opts["columnNames"].(bool); ok {
						includeColumnNames = cn
					}
				}
			}

			var rows [][]interface{}

			// Include column names as first row if requested
			if includeColumnNames && len(result.Columns) > 0 {
				colRow := make([]interface{}, len(result.Columns))
				for i, col := range result.Columns {
					colRow[i] = col
				}
				rows = append(rows, colRow)
			}

			// Add data rows (using RawRows to preserve column order)
			rows = append(rows, result.RawRows...)

			return r.createPromise(vm.ToValue(rows))
		})

		return stmt
	})

	// exec(sql) - executes raw SQL (possibly multiple statements)
	d1.Set("exec", func(call goja.FunctionCall) goja.Value {
		sql := call.Argument(0).String()

		result, err := d1Store.ExecMulti(context.Background(), databaseID, sql)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		resultObj := vm.NewObject()
		resultObj.Set("count", result.Count)
		resultObj.Set("duration", result.Duration)

		return r.createPromise(resultObj)
	})

	// batch(statements) - executes multiple prepared statements
	d1.Set("batch", func(call goja.FunctionCall) goja.Value {
		stmtsVal := call.Argument(0)
		if goja.IsUndefined(stmtsVal) || goja.IsNull(stmtsVal) {
			return r.createRejectedPromise("batch() requires an array of statements")
		}

		stmtsObj := stmtsVal.ToObject(vm)
		length := stmtsObj.Get("length")
		if goja.IsUndefined(length) {
			return r.createRejectedPromise("batch() requires an array of statements")
		}

		numStmts := int(length.ToInteger())
		var results []goja.Value

		for i := 0; i < numStmts; i++ {
			stmtVal := stmtsObj.Get(fmt.Sprintf("%d", i))
			if goja.IsUndefined(stmtVal) || goja.IsNull(stmtVal) {
				continue
			}

			stmtObj := stmtVal.ToObject(vm)

			// Get SQL and params from the statement object
			sqlVal := stmtObj.Get("_sql")
			if goja.IsUndefined(sqlVal) {
				return r.createRejectedPromise(fmt.Sprintf("statement %d is not a valid prepared statement", i))
			}
			sql := sqlVal.String()

			// Get bound params
			var params []interface{}
			getParamsVal := stmtObj.Get("_getParams")
			if !goja.IsUndefined(getParamsVal) {
				if getParamsFunc, ok := goja.AssertFunction(getParamsVal); ok {
					paramsResult, err := getParamsFunc(goja.Undefined())
					if err == nil && !goja.IsUndefined(paramsResult) {
						if p, ok := paramsResult.Export().([]interface{}); ok {
							params = p
						}
					}
				}
			}

			// Determine if this is a SELECT or write operation
			sqlUpper := strings.ToUpper(strings.TrimSpace(sql))
			isSelect := strings.HasPrefix(sqlUpper, "SELECT") ||
				strings.HasPrefix(sqlUpper, "PRAGMA") ||
				strings.HasPrefix(sqlUpper, "EXPLAIN")

			var resultObj *goja.Object
			if isSelect {
				result, err := d1Store.QueryWithMeta(context.Background(), databaseID, sql, params)
				if err != nil {
					return r.createRejectedPromise(fmt.Sprintf("batch statement %d failed: %s", i, err.Error()))
				}

				rows := result.Rows
				if rows == nil {
					rows = []map[string]interface{}{}
				}

				resultObj = vm.NewObject()
				resultObj.Set("results", rows)
				resultObj.Set("success", true)
				resultObj.Set("meta", createMetaObject(result.Meta))
			} else {
				result, err := d1Store.ExecWithMeta(context.Background(), databaseID, sql, params)
				if err != nil {
					return r.createRejectedPromise(fmt.Sprintf("batch statement %d failed: %s", i, err.Error()))
				}

				resultObj = vm.NewObject()
				resultObj.Set("results", []interface{}{})
				resultObj.Set("success", true)
				resultObj.Set("meta", createMetaObject(result.Meta))
			}

			results = append(results, resultObj)
		}

		return r.createPromise(vm.ToValue(results))
	})

	// dump() - returns database as SQLite file bytes
	d1.Set("dump", func(call goja.FunctionCall) goja.Value {
		data, err := d1Store.Dump(context.Background(), databaseID)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		// Return as ArrayBuffer
		arrayBuffer := vm.NewArrayBuffer(data)
		return r.createPromise(vm.ToValue(arrayBuffer))
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

// setupVectorizeBinding creates a Vectorize index binding.
func (r *Runtime) setupVectorizeBinding(name, indexID string) {
	vm := r.vm
	vectorizeStore := r.store.Vectorize()

	vectorize := vm.NewObject()

	// query(vector, options?)
	vectorize.Set("query", func(call goja.FunctionCall) goja.Value {
		vectorVal := call.Argument(0)

		// Extract vector
		var vector []float32
		if arr, ok := vectorVal.Export().([]interface{}); ok {
			for _, v := range arr {
				switch n := v.(type) {
				case float64:
					vector = append(vector, float32(n))
				case int64:
					vector = append(vector, float32(n))
				}
			}
		}

		// Parse options
		opts := &store.VectorQueryOptions{
			TopK:           10,
			ReturnValues:   false,
			ReturnMetadata: "none",
		}

		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			optsArg := call.Arguments[1].ToObject(vm)

			if k := optsArg.Get("topK"); k != nil && !goja.IsUndefined(k) {
				opts.TopK = int(k.ToInteger())
			}
			if rv := optsArg.Get("returnValues"); rv != nil && !goja.IsUndefined(rv) {
				opts.ReturnValues = rv.ToBoolean()
			}
			if rm := optsArg.Get("returnMetadata"); rm != nil && !goja.IsUndefined(rm) {
				if rm.ToBoolean() {
					opts.ReturnMetadata = "all"
				}
			}
			if ns := optsArg.Get("namespace"); ns != nil && !goja.IsUndefined(ns) {
				opts.Namespace = ns.String()
			}
			if f := optsArg.Get("filter"); f != nil && !goja.IsUndefined(f) {
				opts.Filter = f.Export().(map[string]interface{})
			}
		}

		matches, err := vectorizeStore.Query(context.Background(), indexID, vector, opts)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		// Build result
		matchList := make([]map[string]interface{}, len(matches))
		for i, m := range matches {
			match := map[string]interface{}{
				"id":    m.ID,
				"score": m.Score,
			}
			if opts.ReturnValues && m.Values != nil {
				match["values"] = m.Values
			}
			if opts.ReturnMetadata != "none" && m.Metadata != nil {
				match["metadata"] = m.Metadata
			}
			matchList[i] = match
		}

		result := vm.NewObject()
		result.Set("matches", matchList)
		result.Set("count", len(matches))

		return r.createPromise(result)
	})

	// insert(vectors)
	vectorize.Set("insert", func(call goja.FunctionCall) goja.Value {
		vectorsVal := call.Argument(0)
		vectors := vectorsVal.Export().([]interface{})

		var storeVectors []*store.Vector
		var ids []string
		for _, v := range vectors {
			vecMap := v.(map[string]interface{})
			id := vecMap["id"].(string)
			ids = append(ids, id)

			vec := &store.Vector{
				ID: id,
			}

			if values, ok := vecMap["values"].([]interface{}); ok {
				for _, val := range values {
					switch n := val.(type) {
					case float64:
						vec.Values = append(vec.Values, float32(n))
					case int64:
						vec.Values = append(vec.Values, float32(n))
					}
				}
			}

			if metadata, ok := vecMap["metadata"].(map[string]interface{}); ok {
				vec.Metadata = metadata
			}

			if ns, ok := vecMap["namespace"].(string); ok {
				vec.Namespace = ns
			}

			storeVectors = append(storeVectors, vec)
		}

		err := vectorizeStore.Insert(context.Background(), indexID, storeVectors)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		result := vm.NewObject()
		result.Set("ids", ids)
		result.Set("count", len(ids))

		return r.createPromise(result)
	})

	// upsert(vectors)
	vectorize.Set("upsert", func(call goja.FunctionCall) goja.Value {
		vectorsVal := call.Argument(0)
		vectors := vectorsVal.Export().([]interface{})

		var storeVectors []*store.Vector
		var ids []string
		for _, v := range vectors {
			vecMap := v.(map[string]interface{})
			id := vecMap["id"].(string)
			ids = append(ids, id)

			vec := &store.Vector{
				ID: id,
			}

			if values, ok := vecMap["values"].([]interface{}); ok {
				for _, val := range values {
					switch n := val.(type) {
					case float64:
						vec.Values = append(vec.Values, float32(n))
					case int64:
						vec.Values = append(vec.Values, float32(n))
					}
				}
			}

			if metadata, ok := vecMap["metadata"].(map[string]interface{}); ok {
				vec.Metadata = metadata
			}

			storeVectors = append(storeVectors, vec)
		}

		err := vectorizeStore.Upsert(context.Background(), indexID, storeVectors)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		result := vm.NewObject()
		result.Set("ids", ids)
		result.Set("count", len(ids))

		return r.createPromise(result)
	})

	// deleteByIds(ids)
	vectorize.Set("deleteByIds", func(call goja.FunctionCall) goja.Value {
		idsVal := call.Argument(0)
		idsInterface := idsVal.Export().([]interface{})

		var ids []string
		for _, id := range idsInterface {
			ids = append(ids, id.(string))
		}

		err := vectorizeStore.DeleteByIDs(context.Background(), indexID, ids)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		result := vm.NewObject()
		result.Set("count", len(ids))

		return r.createPromise(result)
	})

	// getByIds(ids)
	vectorize.Set("getByIds", func(call goja.FunctionCall) goja.Value {
		idsVal := call.Argument(0)
		idsInterface := idsVal.Export().([]interface{})

		var ids []string
		for _, id := range idsInterface {
			ids = append(ids, id.(string))
		}

		vectors, err := vectorizeStore.GetByIDs(context.Background(), indexID, ids)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		result := make([]map[string]interface{}, len(vectors))
		for i, v := range vectors {
			result[i] = map[string]interface{}{
				"id":       v.ID,
				"values":   v.Values,
				"metadata": v.Metadata,
			}
		}

		return r.createPromise(vm.ToValue(result))
	})

	// describe()
	vectorize.Set("describe", func(call goja.FunctionCall) goja.Value {
		info, err := vectorizeStore.GetIndex(context.Background(), indexID)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		result := vm.NewObject()
		result.Set("name", info.Name)
		result.Set("dimensions", info.Dimensions)
		result.Set("metric", info.Metric)
		result.Set("vectorCount", info.VectorCount)

		return r.createPromise(result)
	})

	r.vm.Set(name, vectorize)
}

// setupHyperdriveBinding creates a Hyperdrive binding.
func (r *Runtime) setupHyperdriveBinding(name, configID string) {
	vm := r.vm
	hyperdriveStore := r.store.Hyperdrive()

	hyperdrive := vm.NewObject()

	// Get config info
	cfg, err := hyperdriveStore.GetConfig(context.Background(), configID)
	if err != nil {
		// Set empty values on error
		hyperdrive.Set("connectionString", "")
		hyperdrive.Set("host", "")
		hyperdrive.Set("port", 0)
		hyperdrive.Set("user", "")
		hyperdrive.Set("password", "")
		hyperdrive.Set("database", "")
	} else {
		// Build connection string from origin
		connStr := fmt.Sprintf("%s://%s:%s@%s:%d/%s",
			cfg.Origin.Scheme,
			cfg.Origin.User,
			cfg.Origin.Password,
			cfg.Origin.Host,
			cfg.Origin.Port,
			cfg.Origin.Database,
		)
		hyperdrive.Set("connectionString", connStr)
		hyperdrive.Set("host", cfg.Origin.Host)
		hyperdrive.Set("port", cfg.Origin.Port)
		hyperdrive.Set("user", cfg.Origin.User)
		hyperdrive.Set("password", cfg.Origin.Password)
		hyperdrive.Set("database", cfg.Origin.Database)
	}

	r.vm.Set(name, hyperdrive)
}

// setupAnalyticsEngineBinding creates an Analytics Engine binding.
func (r *Runtime) setupAnalyticsEngineBinding(name, datasetID string) {
	vm := r.vm
	analyticsStore := r.store.AnalyticsEngine()

	dataset := vm.NewObject()

	// writeDataPoint(event)
	dataset.Set("writeDataPoint", func(call goja.FunctionCall) goja.Value {
		eventObj := call.Argument(0).ToObject(vm)

		event := &store.AnalyticsEngineDataPoint{
			Dataset:   datasetID,
			Timestamp: time.Now(),
		}

		// Parse indexes (up to 1)
		if indexes := eventObj.Get("indexes"); indexes != nil && !goja.IsUndefined(indexes) {
			if arr, ok := indexes.Export().([]interface{}); ok {
				for _, idx := range arr {
					event.Indexes = append(event.Indexes, idx.(string))
				}
			}
		}

		// Parse doubles (up to 20)
		if doubles := eventObj.Get("doubles"); doubles != nil && !goja.IsUndefined(doubles) {
			if arr, ok := doubles.Export().([]interface{}); ok {
				for _, d := range arr {
					switch n := d.(type) {
					case float64:
						event.Doubles = append(event.Doubles, n)
					case int64:
						event.Doubles = append(event.Doubles, float64(n))
					}
				}
			}
		}

		// Parse blobs (up to 20)
		if blobs := eventObj.Get("blobs"); blobs != nil && !goja.IsUndefined(blobs) {
			if arr, ok := blobs.Export().([]interface{}); ok {
				for _, b := range arr {
					switch v := b.(type) {
					case string:
						event.Blobs = append(event.Blobs, []byte(v))
					case []byte:
						event.Blobs = append(event.Blobs, v)
					}
				}
			}
		}

		// Write asynchronously (non-blocking)
		go analyticsStore.WriteDataPoint(context.Background(), event)

		return goja.Undefined()
	})

	r.vm.Set(name, dataset)
}

// setupServiceBinding creates a Service binding (Fetcher).
func (r *Runtime) setupServiceBinding(name, targetWorker string) {
	vm := r.vm

	service := vm.NewObject()

	// fetch(request, init?)
	service.Set("fetch", func(call goja.FunctionCall) goja.Value {
		// Get request URL
		urlArg := call.Argument(0)
		var url string
		method := "GET"
		var bodyData []byte
		headers := make(map[string]string)

		// Handle Request object or URL string
		if urlArg.ExportType().Kind().String() == "string" {
			url = urlArg.String()
		} else {
			reqObj := urlArg.ToObject(vm)
			if u := reqObj.Get("url"); u != nil && !goja.IsUndefined(u) {
				url = u.String()
			}
			if m := reqObj.Get("method"); m != nil && !goja.IsUndefined(m) {
				method = m.String()
			}
		}

		// Parse init options
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			init := call.Arguments[1].ToObject(vm)
			if m := init.Get("method"); m != nil && !goja.IsUndefined(m) {
				method = m.String()
			}
			if h := init.Get("headers"); h != nil && !goja.IsUndefined(h) {
				hObj := h.ToObject(vm)
				for _, key := range hObj.Keys() {
					val := hObj.Get(key)
					if val != nil && !goja.IsUndefined(val) {
						headers[key] = val.String()
					}
				}
			}
			if b := init.Get("body"); b != nil && !goja.IsUndefined(b) {
				bodyData = []byte(b.String())
			}
		}

		// Execute the target worker
		executor := r.store.Workers()
		worker, err := executor.GetByName(context.Background(), targetWorker)
		if err != nil {
			return r.createRejectedPromise(fmt.Sprintf("service binding error: %v", err))
		}

		// Create a new request
		var reqBody io.Reader
		if len(bodyData) > 0 {
			reqBody = strings.NewReader(string(bodyData))
		}

		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		// Execute in a new runtime (to avoid deadlock)
		newRuntime := New(Config{Store: r.store, Bindings: worker.Bindings})
		defer newRuntime.Close()

		resp, err := newRuntime.Execute(context.Background(), worker.Script, req)
		if err != nil {
			return r.createRejectedPromise(err.Error())
		}

		// Build response object
		respObj := vm.NewObject()
		respObj.Set("status", resp.Status)
		respObj.Set("ok", resp.Status >= 200 && resp.Status < 300)
		respObj.Set("_body", string(resp.Body))

		// Headers
		respHeaders := vm.NewObject()
		headerData := make(map[string]string)
		for k, v := range resp.Headers {
			if len(v) > 0 {
				headerData[strings.ToLower(k)] = v[0]
			}
		}
		respHeaders.Set("get", func(c goja.FunctionCall) goja.Value {
			key := strings.ToLower(c.Argument(0).String())
			if val, ok := headerData[key]; ok {
				return vm.ToValue(val)
			}
			return goja.Null()
		})
		respObj.Set("headers", respHeaders)

		// Body methods
		respObj.Set("text", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue(string(resp.Body)))
		})
		respObj.Set("json", func(c goja.FunctionCall) goja.Value {
			var data interface{}
			json.Unmarshal(resp.Body, &data)
			return r.createPromise(vm.ToValue(data))
		})
		respObj.Set("arrayBuffer", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue(resp.Body))
		})

		return r.createPromise(respObj)
	})

	r.vm.Set(name, service)
}

// In-memory cache entry for Workers Cache API
type workersCacheEntry struct {
	Body    []byte
	Status  int
	Headers map[string]string
}

// Global in-memory cache storage for Workers Cache API
var workersCaches = make(map[string]map[string]*workersCacheEntry)

// setupCacheBinding creates a Cache API binding with in-memory storage.
func (r *Runtime) setupCacheBinding() {
	vm := r.vm

	// Cache storage
	caches := vm.NewObject()

	// Create default cache
	defaultCache := r.createCache("default")
	caches.Set("default", defaultCache)

	// caches.open(name)
	caches.Set("open", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		cache := r.createCache(name)
		return r.createPromise(cache)
	})

	vm.Set("caches", caches)
}

func (r *Runtime) createCache(cacheName string) goja.Value {
	vm := r.vm

	// Ensure cache namespace exists
	if workersCaches[cacheName] == nil {
		workersCaches[cacheName] = make(map[string]*workersCacheEntry)
	}
	cacheData := workersCaches[cacheName]

	cache := vm.NewObject()

	// put(request, response)
	cache.Set("put", func(call goja.FunctionCall) goja.Value {
		reqArg := call.Argument(0)
		respArg := call.Argument(1)

		// Get cache key from request URL
		var cacheKey string
		if reqArg.ExportType().Kind().String() == "string" {
			cacheKey = reqArg.String()
		} else {
			reqObj := reqArg.ToObject(vm)
			if u := reqObj.Get("url"); u != nil && !goja.IsUndefined(u) {
				cacheKey = u.String()
			}
		}

		// Get response body
		respObj := respArg.ToObject(vm)
		var body []byte
		if b := respObj.Get("_body"); b != nil && !goja.IsUndefined(b) {
			body = []byte(b.String())
		}

		status := 200
		if s := respObj.Get("status"); s != nil && !goja.IsUndefined(s) {
			status = int(s.ToInteger())
		}

		// Get headers
		headers := make(map[string]string)
		if h := respObj.Get("headers"); h != nil && !goja.IsUndefined(h) {
			hObj := h.ToObject(vm)
			for _, key := range hObj.Keys() {
				val := hObj.Get(key)
				if val != nil && !goja.IsUndefined(val) {
					headers[key] = val.String()
				}
			}
		}

		// Store in cache
		cacheData[cacheKey] = &workersCacheEntry{
			Body:    body,
			Status:  status,
			Headers: headers,
		}

		return r.createPromise(goja.Undefined())
	})

	// match(request, options?)
	cache.Set("match", func(call goja.FunctionCall) goja.Value {
		reqArg := call.Argument(0)

		// Get cache key from request URL
		var cacheKey string
		if reqArg.ExportType().Kind().String() == "string" {
			cacheKey = reqArg.String()
		} else {
			reqObj := reqArg.ToObject(vm)
			if u := reqObj.Get("url"); u != nil && !goja.IsUndefined(u) {
				cacheKey = u.String()
			}
		}

		// Get from cache
		entry, ok := cacheData[cacheKey]
		if !ok || entry == nil {
			return r.createPromise(goja.Undefined())
		}

		// Build response object
		respObj := vm.NewObject()
		respObj.Set("status", entry.Status)
		respObj.Set("ok", entry.Status >= 200 && entry.Status < 300)
		respObj.Set("_body", string(entry.Body))

		// Headers
		respHeaders := vm.NewObject()
		for k, v := range entry.Headers {
			respHeaders.Set(strings.ToLower(k), v)
		}
		respHeaders.Set("get", func(c goja.FunctionCall) goja.Value {
			key := strings.ToLower(c.Argument(0).String())
			if val, ok := entry.Headers[key]; ok {
				return vm.ToValue(val)
			}
			return goja.Null()
		})
		respObj.Set("headers", respHeaders)

		// Body methods
		respObj.Set("text", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue(string(entry.Body)))
		})
		respObj.Set("json", func(c goja.FunctionCall) goja.Value {
			var data interface{}
			json.Unmarshal(entry.Body, &data)
			return r.createPromise(vm.ToValue(data))
		})

		return r.createPromise(respObj)
	})

	// delete(request, options?)
	cache.Set("delete", func(call goja.FunctionCall) goja.Value {
		reqArg := call.Argument(0)

		// Get cache key from request URL
		var cacheKey string
		if reqArg.ExportType().Kind().String() == "string" {
			cacheKey = reqArg.String()
		} else {
			reqObj := reqArg.ToObject(vm)
			if u := reqObj.Get("url"); u != nil && !goja.IsUndefined(u) {
				cacheKey = u.String()
			}
		}

		_, ok := cacheData[cacheKey]
		if ok {
			delete(cacheData, cacheKey)
		}

		return r.createPromise(vm.ToValue(ok))
	})

	return cache
}
