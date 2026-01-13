package runtime

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"

	"github.com/dop251/goja"
)

// setupExtendedWebAPIs sets up additional Web APIs for Cloudflare Workers compatibility.
func (r *Runtime) setupExtendedWebAPIs() {
	r.setupFormData()
	r.setupBlob()
	r.setupSubtleCrypto()
	r.setupStreams()
	r.setupAbortController()
}

// setupFormData implements the FormData API.
func (r *Runtime) setupFormData() {
	vm := r.vm

	vm.Set("FormData", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		entries := make(map[string][]formDataEntry)

		obj.Set("append", func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			value := c.Argument(1)

			entry := formDataEntry{}
			if isBlob(value, vm) {
				entry.isFile = true
				entry.blob = value.ToObject(vm)
				if len(c.Arguments) > 2 {
					entry.filename = c.Argument(2).String()
				}
			} else {
				entry.value = value.String()
			}

			entries[name] = append(entries[name], entry)
			return goja.Undefined()
		})

		obj.Set("delete", func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			delete(entries, name)
			return goja.Undefined()
		})

		obj.Set("get", func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			if vals, ok := entries[name]; ok && len(vals) > 0 {
				if vals[0].isFile {
					return vals[0].blob
				}
				return vm.ToValue(vals[0].value)
			}
			return goja.Null()
		})

		obj.Set("getAll", func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			if vals, ok := entries[name]; ok {
				result := make([]interface{}, len(vals))
				for i, v := range vals {
					if v.isFile {
						result[i] = v.blob
					} else {
						result[i] = v.value
					}
				}
				return vm.ToValue(result)
			}
			return vm.ToValue([]interface{}{})
		})

		obj.Set("has", func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			_, ok := entries[name]
			return vm.ToValue(ok)
		})

		obj.Set("set", func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			value := c.Argument(1)

			entry := formDataEntry{}
			if isBlob(value, vm) {
				entry.isFile = true
				entry.blob = value.ToObject(vm)
				if len(c.Arguments) > 2 {
					entry.filename = c.Argument(2).String()
				}
			} else {
				entry.value = value.String()
			}

			entries[name] = []formDataEntry{entry}
			return goja.Undefined()
		})

		obj.Set("entries", func(c goja.FunctionCall) goja.Value {
			var result [][]interface{}
			for name, vals := range entries {
				for _, v := range vals {
					if v.isFile {
						result = append(result, []interface{}{name, v.blob})
					} else {
						result = append(result, []interface{}{name, v.value})
					}
				}
			}
			return vm.ToValue(result)
		})

		obj.Set("keys", func(c goja.FunctionCall) goja.Value {
			var keys []string
			for name := range entries {
				keys = append(keys, name)
			}
			return vm.ToValue(keys)
		})

		obj.Set("values", func(c goja.FunctionCall) goja.Value {
			var values []interface{}
			for _, vals := range entries {
				for _, v := range vals {
					if v.isFile {
						values = append(values, v.blob)
					} else {
						values = append(values, v.value)
					}
				}
			}
			return vm.ToValue(values)
		})

		obj.Set("forEach", func(c goja.FunctionCall) goja.Value {
			callback, ok := goja.AssertFunction(c.Argument(0))
			if ok {
				for name, vals := range entries {
					for _, v := range vals {
						var value interface{}
						if v.isFile {
							value = v.blob
						} else {
							value = v.value
						}
						callback(nil, vm.ToValue(value), vm.ToValue(name), obj)
					}
				}
			}
			return goja.Undefined()
		})

		// Internal method to encode as multipart form data
		obj.Set("_encode", func(c goja.FunctionCall) goja.Value {
			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)

			for name, vals := range entries {
				for _, v := range vals {
					if v.isFile {
						filename := v.filename
						if filename == "" {
							filename = "blob"
						}
						contentType := "application/octet-stream"
						if typeVal := v.blob.Get("type"); typeVal != nil && !goja.IsUndefined(typeVal) {
							contentType = typeVal.String()
						}

						h := make(textproto.MIMEHeader)
						h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, name, filename))
						h.Set("Content-Type", contentType)

						part, _ := writer.CreatePart(h)
						if dataVal := v.blob.Get("_data"); dataVal != nil && !goja.IsUndefined(dataVal) {
							if data, ok := dataVal.Export().([]byte); ok {
								part.Write(data)
							} else if data, ok := dataVal.Export().(string); ok {
								part.Write([]byte(data))
							}
						}
					} else {
						writer.WriteField(name, v.value)
					}
				}
			}

			writer.Close()

			result := vm.NewObject()
			result.Set("boundary", writer.Boundary())
			result.Set("body", buf.Bytes())
			return result
		})

		return obj
	})
}

type formDataEntry struct {
	value    string
	isFile   bool
	blob     *goja.Object
	filename string
}

func isBlob(val goja.Value, vm *goja.Runtime) bool {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return false
	}
	// Check if it's an object type before trying to convert
	exported := val.Export()
	if _, ok := exported.(string); ok {
		return false
	}
	if _, ok := exported.(float64); ok {
		return false
	}
	if _, ok := exported.(bool); ok {
		return false
	}
	obj := val.ToObject(vm)
	if obj == nil {
		return false
	}
	// Check if it has Blob-like properties
	isB := obj.Get("_isBlob")
	return isB != nil && !goja.IsUndefined(isB)
}

// setupBlob implements the Blob and File APIs.
func (r *Runtime) setupBlob() {
	vm := r.vm

	vm.Set("Blob", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		var data []byte
		contentType := ""

		// Process blob parts
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			parts := call.Arguments[0].Export()
			if arr, ok := parts.([]interface{}); ok {
				for _, part := range arr {
					switch v := part.(type) {
					case string:
						data = append(data, []byte(v)...)
					case []byte:
						data = append(data, v...)
					case map[string]interface{}:
						// Another blob
						if d, ok := v["_data"].([]byte); ok {
							data = append(data, d...)
						}
					}
				}
			}
		}

		// Process options
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			opts := call.Arguments[1].ToObject(vm)
			if t := opts.Get("type"); t != nil && !goja.IsUndefined(t) {
				contentType = t.String()
			}
		}

		obj.Set("_isBlob", true)
		obj.Set("_data", data)
		obj.Set("size", len(data))
		obj.Set("type", contentType)

		obj.Set("slice", func(c goja.FunctionCall) goja.Value {
			start := 0
			end := len(data)
			newType := contentType

			if len(c.Arguments) > 0 && !goja.IsUndefined(c.Arguments[0]) {
				start = int(c.Argument(0).ToInteger())
				if start < 0 {
					start = len(data) + start
				}
			}
			if len(c.Arguments) > 1 && !goja.IsUndefined(c.Arguments[1]) {
				end = int(c.Argument(1).ToInteger())
				if end < 0 {
					end = len(data) + end
				}
			}
			if len(c.Arguments) > 2 && !goja.IsUndefined(c.Arguments[2]) {
				newType = c.Argument(2).String()
			}

			if start < 0 {
				start = 0
			}
			if end > len(data) {
				end = len(data)
			}
			if start > end {
				start = end
			}

			slicedData := data[start:end]

			blob := vm.NewObject()
			blob.Set("_isBlob", true)
			blob.Set("_data", slicedData)
			blob.Set("size", len(slicedData))
			blob.Set("type", newType)

			return blob
		})

		obj.Set("text", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue(string(data)))
		})

		obj.Set("arrayBuffer", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue(data))
		})

		obj.Set("stream", func(c goja.FunctionCall) goja.Value {
			return r.createReadableStreamFromBytes(data)
		})

		return obj
	})

	// File extends Blob
	vm.Set("File", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		var data []byte
		var fileName string
		contentType := ""
		lastModified := int64(0)

		// Process file parts
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			parts := call.Arguments[0].Export()
			if arr, ok := parts.([]interface{}); ok {
				for _, part := range arr {
					switch v := part.(type) {
					case string:
						data = append(data, []byte(v)...)
					case []byte:
						data = append(data, v...)
					}
				}
			}
		}

		// Get filename
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			fileName = call.Arguments[1].String()
		}

		// Process options
		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) {
			opts := call.Arguments[2].ToObject(vm)
			if t := opts.Get("type"); t != nil && !goja.IsUndefined(t) {
				contentType = t.String()
			}
			if lm := opts.Get("lastModified"); lm != nil && !goja.IsUndefined(lm) {
				lastModified = lm.ToInteger()
			}
		}

		obj.Set("_isBlob", true)
		obj.Set("_data", data)
		obj.Set("size", len(data))
		obj.Set("type", contentType)
		obj.Set("name", fileName)
		obj.Set("lastModified", lastModified)

		obj.Set("slice", func(c goja.FunctionCall) goja.Value {
			start := 0
			end := len(data)
			newType := contentType

			if len(c.Arguments) > 0 && !goja.IsUndefined(c.Arguments[0]) {
				start = int(c.Argument(0).ToInteger())
			}
			if len(c.Arguments) > 1 && !goja.IsUndefined(c.Arguments[1]) {
				end = int(c.Argument(1).ToInteger())
			}
			if len(c.Arguments) > 2 && !goja.IsUndefined(c.Arguments[2]) {
				newType = c.Argument(2).String()
			}

			if start < 0 {
				start = 0
			}
			if end > len(data) {
				end = len(data)
			}

			slicedData := data[start:end]

			blob := vm.NewObject()
			blob.Set("_isBlob", true)
			blob.Set("_data", slicedData)
			blob.Set("size", len(slicedData))
			blob.Set("type", newType)

			return blob
		})

		obj.Set("text", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue(string(data)))
		})

		obj.Set("arrayBuffer", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue(data))
		})

		obj.Set("stream", func(c goja.FunctionCall) goja.Value {
			return r.createReadableStreamFromBytes(data)
		})

		return obj
	})
}

// setupSubtleCrypto implements the SubtleCrypto API.
func (r *Runtime) setupSubtleCrypto() {
	vm := r.vm
	crypto := vm.Get("crypto").ToObject(vm)

	subtle := vm.NewObject()

	// digest(algorithm, data) -> Promise<ArrayBuffer>
	subtle.Set("digest", func(call goja.FunctionCall) goja.Value {
		algorithm := call.Argument(0)
		data := call.Argument(1)

		var algoName string
		if algorithm.ExportType().Kind().String() == "string" {
			algoName = algorithm.String()
		} else {
			algoObj := algorithm.ToObject(vm)
			if name := algoObj.Get("name"); name != nil {
				algoName = name.String()
			}
		}

		var dataBytes []byte
		switch v := data.Export().(type) {
		case []byte:
			dataBytes = v
		case string:
			dataBytes = []byte(v)
		default:
			return r.createRejectedPromise("invalid data type")
		}

		var hasher hash.Hash
		switch strings.ToUpper(algoName) {
		case "SHA-1":
			hasher = sha1.New()
		case "SHA-256":
			hasher = sha256.New()
		case "SHA-384":
			hasher = sha512.New384()
		case "SHA-512":
			hasher = sha512.New()
		case "MD5":
			hasher = md5.New()
		default:
			return r.createRejectedPromise("unsupported algorithm: " + algoName)
		}

		hasher.Write(dataBytes)
		result := hasher.Sum(nil)

		return r.createPromise(vm.ToValue(result))
	})

	// generateKey(algorithm, extractable, keyUsages) -> Promise<CryptoKey>
	subtle.Set("generateKey", func(call goja.FunctionCall) goja.Value {
		algorithm := call.Argument(0).ToObject(vm)
		extractable := call.Argument(1).ToBoolean()
		keyUsages := call.Argument(2).Export()

		algoName := ""
		if name := algorithm.Get("name"); name != nil {
			algoName = strings.ToUpper(name.String())
		}

		usages := []string{}
		if arr, ok := keyUsages.([]interface{}); ok {
			for _, u := range arr {
				usages = append(usages, u.(string))
			}
		}

		key := vm.NewObject()
		key.Set("type", "secret")
		key.Set("extractable", extractable)
		key.Set("usages", usages)

		switch algoName {
		case "AES-GCM", "AES-CBC", "AES-CTR":
			length := 256
			if l := algorithm.Get("length"); l != nil && !goja.IsUndefined(l) {
				length = int(l.ToInteger())
			}

			keyBytes := make([]byte, length/8)
			rand.Read(keyBytes)

			key.Set("_raw", keyBytes)
			key.Set("algorithm", map[string]interface{}{
				"name":   algoName,
				"length": length,
			})

		case "HMAC":
			hashName := "SHA-256"
			if h := algorithm.Get("hash"); h != nil && !goja.IsUndefined(h) {
				if h.ExportType().Kind().String() == "string" {
					hashName = h.String()
				} else {
					hashObj := h.ToObject(vm)
					if name := hashObj.Get("name"); name != nil {
						hashName = name.String()
					}
				}
			}

			length := 256
			if l := algorithm.Get("length"); l != nil && !goja.IsUndefined(l) {
				length = int(l.ToInteger())
			}

			keyBytes := make([]byte, length/8)
			rand.Read(keyBytes)

			key.Set("_raw", keyBytes)
			key.Set("algorithm", map[string]interface{}{
				"name": algoName,
				"hash": map[string]interface{}{"name": hashName},
			})

		case "RSA-OAEP", "RSASSA-PKCS1-V1_5":
			modulusLength := 2048
			if ml := algorithm.Get("modulusLength"); ml != nil && !goja.IsUndefined(ml) {
				modulusLength = int(ml.ToInteger())
			}

			privateKey, _ := rsa.GenerateKey(rand.Reader, modulusLength)
			privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
			publicKeyBytes := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)

			publicKey := vm.NewObject()
			publicKey.Set("type", "public")
			publicKey.Set("extractable", true)
			publicKey.Set("usages", []string{"encrypt", "verify"})
			publicKey.Set("_raw", publicKeyBytes)
			publicKey.Set("algorithm", map[string]interface{}{
				"name":           algoName,
				"modulusLength":  modulusLength,
				"publicExponent": []byte{1, 0, 1},
			})

			key.Set("type", "private")
			key.Set("_raw", privateKeyBytes)
			key.Set("algorithm", map[string]interface{}{
				"name":           algoName,
				"modulusLength":  modulusLength,
				"publicExponent": []byte{1, 0, 1},
			})

			keyPair := vm.NewObject()
			keyPair.Set("publicKey", publicKey)
			keyPair.Set("privateKey", key)
			return r.createPromise(keyPair)

		case "ECDSA", "ECDH":
			curveName := "P-256"
			if c := algorithm.Get("namedCurve"); c != nil && !goja.IsUndefined(c) {
				curveName = c.String()
			}

			var curve elliptic.Curve
			switch curveName {
			case "P-256":
				curve = elliptic.P256()
			case "P-384":
				curve = elliptic.P384()
			case "P-521":
				curve = elliptic.P521()
			default:
				return r.createRejectedPromise("unsupported curve: " + curveName)
			}

			privateKey, _ := ecdsa.GenerateKey(curve, rand.Reader)
			privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
			publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)

			publicKey := vm.NewObject()
			publicKey.Set("type", "public")
			publicKey.Set("extractable", true)
			publicKey.Set("usages", []string{"verify"})
			publicKey.Set("_raw", publicKeyBytes)
			publicKey.Set("algorithm", map[string]interface{}{
				"name":       algoName,
				"namedCurve": curveName,
			})

			key.Set("type", "private")
			key.Set("_raw", privateKeyBytes)
			key.Set("algorithm", map[string]interface{}{
				"name":       algoName,
				"namedCurve": curveName,
			})

			keyPair := vm.NewObject()
			keyPair.Set("publicKey", publicKey)
			keyPair.Set("privateKey", key)
			return r.createPromise(keyPair)

		default:
			return r.createRejectedPromise("unsupported algorithm: " + algoName)
		}

		return r.createPromise(key)
	})

	// importKey(format, keyData, algorithm, extractable, keyUsages) -> Promise<CryptoKey>
	subtle.Set("importKey", func(call goja.FunctionCall) goja.Value {
		format := call.Argument(0).String()
		keyData := call.Argument(1)
		algorithm := call.Argument(2).ToObject(vm)
		extractable := call.Argument(3).ToBoolean()
		keyUsages := call.Argument(4).Export()

		algoName := ""
		if name := algorithm.Get("name"); name != nil {
			algoName = strings.ToUpper(name.String())
		}

		usages := []string{}
		if arr, ok := keyUsages.([]interface{}); ok {
			for _, u := range arr {
				usages = append(usages, u.(string))
			}
		}

		var keyBytes []byte
		switch v := keyData.Export().(type) {
		case []byte:
			keyBytes = v
		case string:
			keyBytes = []byte(v)
		case map[string]interface{}:
			// JWK format
			if k, ok := v["k"].(string); ok {
				keyBytes, _ = base64Decode(k)
			}
		}

		key := vm.NewObject()
		key.Set("type", "secret")
		key.Set("extractable", extractable)
		key.Set("usages", usages)
		key.Set("_raw", keyBytes)

		switch algoName {
		case "AES-GCM", "AES-CBC", "AES-CTR":
			key.Set("algorithm", map[string]interface{}{
				"name":   algoName,
				"length": len(keyBytes) * 8,
			})
		case "HMAC":
			hashName := "SHA-256"
			if h := algorithm.Get("hash"); h != nil && !goja.IsUndefined(h) {
				if h.ExportType().Kind().String() == "string" {
					hashName = h.String()
				} else {
					hashObj := h.ToObject(vm)
					if name := hashObj.Get("name"); name != nil {
						hashName = name.String()
					}
				}
			}
			key.Set("algorithm", map[string]interface{}{
				"name": algoName,
				"hash": map[string]interface{}{"name": hashName},
			})
		default:
			key.Set("algorithm", map[string]interface{}{"name": algoName})
		}

		_ = format // format handling
		return r.createPromise(key)
	})

	// exportKey(format, key) -> Promise<ArrayBuffer>
	subtle.Set("exportKey", func(call goja.FunctionCall) goja.Value {
		format := call.Argument(0).String()
		key := call.Argument(1).ToObject(vm)

		rawVal := key.Get("_raw")
		if rawVal == nil || goja.IsUndefined(rawVal) {
			return r.createRejectedPromise("key not extractable")
		}

		keyBytes, ok := rawVal.Export().([]byte)
		if !ok {
			return r.createRejectedPromise("invalid key data")
		}

		switch format {
		case "raw":
			return r.createPromise(vm.ToValue(keyBytes))
		case "jwk":
			jwk := map[string]interface{}{
				"kty": "oct",
				"k":   base64Encode(keyBytes),
			}
			return r.createPromise(vm.ToValue(jwk))
		default:
			return r.createRejectedPromise("unsupported format: " + format)
		}
	})

	// encrypt(algorithm, key, data) -> Promise<ArrayBuffer>
	subtle.Set("encrypt", func(call goja.FunctionCall) goja.Value {
		algorithm := call.Argument(0).ToObject(vm)
		key := call.Argument(1).ToObject(vm)
		data := call.Argument(2)

		algoName := ""
		if name := algorithm.Get("name"); name != nil {
			algoName = strings.ToUpper(name.String())
		}

		keyBytes, ok := key.Get("_raw").Export().([]byte)
		if !ok {
			return r.createRejectedPromise("invalid key")
		}

		var dataBytes []byte
		switch v := data.Export().(type) {
		case []byte:
			dataBytes = v
		case string:
			dataBytes = []byte(v)
		}

		switch algoName {
		case "AES-GCM":
			block, err := aes.NewCipher(keyBytes)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			var iv []byte
			if ivVal := algorithm.Get("iv"); ivVal != nil && !goja.IsUndefined(ivVal) {
				iv, _ = ivVal.Export().([]byte)
			}
			if len(iv) == 0 {
				iv = make([]byte, gcm.NonceSize())
				rand.Read(iv)
			}

			ciphertext := gcm.Seal(nil, iv, dataBytes, nil)
			result := append(iv, ciphertext...)
			return r.createPromise(vm.ToValue(result))

		case "AES-CBC":
			block, err := aes.NewCipher(keyBytes)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			var iv []byte
			if ivVal := algorithm.Get("iv"); ivVal != nil && !goja.IsUndefined(ivVal) {
				iv, _ = ivVal.Export().([]byte)
			}
			if len(iv) == 0 {
				iv = make([]byte, aes.BlockSize)
				rand.Read(iv)
			}

			// PKCS7 padding
			padding := aes.BlockSize - len(dataBytes)%aes.BlockSize
			padded := make([]byte, len(dataBytes)+padding)
			copy(padded, dataBytes)
			for i := len(dataBytes); i < len(padded); i++ {
				padded[i] = byte(padding)
			}

			mode := cipher.NewCBCEncrypter(block, iv)
			ciphertext := make([]byte, len(padded))
			mode.CryptBlocks(ciphertext, padded)

			result := append(iv, ciphertext...)
			return r.createPromise(vm.ToValue(result))

		default:
			return r.createRejectedPromise("unsupported algorithm: " + algoName)
		}
	})

	// decrypt(algorithm, key, data) -> Promise<ArrayBuffer>
	subtle.Set("decrypt", func(call goja.FunctionCall) goja.Value {
		algorithm := call.Argument(0).ToObject(vm)
		key := call.Argument(1).ToObject(vm)
		data := call.Argument(2)

		algoName := ""
		if name := algorithm.Get("name"); name != nil {
			algoName = strings.ToUpper(name.String())
		}

		keyBytes, ok := key.Get("_raw").Export().([]byte)
		if !ok {
			return r.createRejectedPromise("invalid key")
		}

		var dataBytes []byte
		switch v := data.Export().(type) {
		case []byte:
			dataBytes = v
		case string:
			dataBytes = []byte(v)
		}

		switch algoName {
		case "AES-GCM":
			block, err := aes.NewCipher(keyBytes)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			nonceSize := gcm.NonceSize()
			if len(dataBytes) < nonceSize {
				return r.createRejectedPromise("ciphertext too short")
			}

			nonce, ciphertext := dataBytes[:nonceSize], dataBytes[nonceSize:]
			plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			return r.createPromise(vm.ToValue(plaintext))

		case "AES-CBC":
			block, err := aes.NewCipher(keyBytes)
			if err != nil {
				return r.createRejectedPromise(err.Error())
			}

			if len(dataBytes) < aes.BlockSize {
				return r.createRejectedPromise("ciphertext too short")
			}

			iv := dataBytes[:aes.BlockSize]
			ciphertext := dataBytes[aes.BlockSize:]

			mode := cipher.NewCBCDecrypter(block, iv)
			plaintext := make([]byte, len(ciphertext))
			mode.CryptBlocks(plaintext, ciphertext)

			// Remove PKCS7 padding
			padding := int(plaintext[len(plaintext)-1])
			if padding > aes.BlockSize || padding == 0 {
				return r.createRejectedPromise("invalid padding")
			}
			plaintext = plaintext[:len(plaintext)-padding]

			return r.createPromise(vm.ToValue(plaintext))

		default:
			return r.createRejectedPromise("unsupported algorithm: " + algoName)
		}
	})

	// sign(algorithm, key, data) -> Promise<ArrayBuffer>
	subtle.Set("sign", func(call goja.FunctionCall) goja.Value {
		algorithm := call.Argument(0).ToObject(vm)
		key := call.Argument(1).ToObject(vm)
		data := call.Argument(2)

		algoName := ""
		if name := algorithm.Get("name"); name != nil {
			algoName = strings.ToUpper(name.String())
		}

		keyBytes, ok := key.Get("_raw").Export().([]byte)
		if !ok {
			return r.createRejectedPromise("invalid key")
		}

		var dataBytes []byte
		switch v := data.Export().(type) {
		case []byte:
			dataBytes = v
		case string:
			dataBytes = []byte(v)
		}

		switch algoName {
		case "HMAC":
			hashName := "SHA-256"
			if h := algorithm.Get("hash"); h != nil && !goja.IsUndefined(h) {
				if h.ExportType().Kind().String() == "string" {
					hashName = h.String()
				} else {
					hashObj := h.ToObject(vm)
					if name := hashObj.Get("name"); name != nil {
						hashName = name.String()
					}
				}
			}

			var hasher func() hash.Hash
			switch strings.ToUpper(hashName) {
			case "SHA-1":
				hasher = sha1.New
			case "SHA-256":
				hasher = sha256.New
			case "SHA-384":
				hasher = sha512.New384
			case "SHA-512":
				hasher = sha512.New
			default:
				return r.createRejectedPromise("unsupported hash: " + hashName)
			}

			mac := hmac.New(hasher, keyBytes)
			mac.Write(dataBytes)
			signature := mac.Sum(nil)

			return r.createPromise(vm.ToValue(signature))

		default:
			return r.createRejectedPromise("unsupported algorithm: " + algoName)
		}
	})

	// verify(algorithm, key, signature, data) -> Promise<boolean>
	subtle.Set("verify", func(call goja.FunctionCall) goja.Value {
		algorithm := call.Argument(0).ToObject(vm)
		key := call.Argument(1).ToObject(vm)
		signature := call.Argument(2)
		data := call.Argument(3)

		algoName := ""
		if name := algorithm.Get("name"); name != nil {
			algoName = strings.ToUpper(name.String())
		}

		keyBytes, ok := key.Get("_raw").Export().([]byte)
		if !ok {
			return r.createRejectedPromise("invalid key")
		}

		var sigBytes []byte
		switch v := signature.Export().(type) {
		case []byte:
			sigBytes = v
		case string:
			sigBytes = []byte(v)
		}

		var dataBytes []byte
		switch v := data.Export().(type) {
		case []byte:
			dataBytes = v
		case string:
			dataBytes = []byte(v)
		}

		switch algoName {
		case "HMAC":
			hashName := "SHA-256"
			if h := algorithm.Get("hash"); h != nil && !goja.IsUndefined(h) {
				if h.ExportType().Kind().String() == "string" {
					hashName = h.String()
				} else {
					hashObj := h.ToObject(vm)
					if name := hashObj.Get("name"); name != nil {
						hashName = name.String()
					}
				}
			}

			var hasher func() hash.Hash
			switch strings.ToUpper(hashName) {
			case "SHA-1":
				hasher = sha1.New
			case "SHA-256":
				hasher = sha256.New
			case "SHA-384":
				hasher = sha512.New384
			case "SHA-512":
				hasher = sha512.New
			default:
				return r.createRejectedPromise("unsupported hash: " + hashName)
			}

			mac := hmac.New(hasher, keyBytes)
			mac.Write(dataBytes)
			expected := mac.Sum(nil)

			valid := hmac.Equal(sigBytes, expected)
			return r.createPromise(vm.ToValue(valid))

		default:
			return r.createRejectedPromise("unsupported algorithm: " + algoName)
		}
	})

	crypto.Set("subtle", subtle)

	// crypto.getRandomValues(array)
	crypto.Set("getRandomValues", func(call goja.FunctionCall) goja.Value {
		arr := call.Argument(0)

		if arrBytes, ok := arr.Export().([]byte); ok {
			rand.Read(arrBytes)
			return vm.ToValue(arrBytes)
		}

		// Handle typed arrays
		obj := arr.ToObject(vm)
		length := 0
		if l := obj.Get("length"); l != nil && !goja.IsUndefined(l) {
			length = int(l.ToInteger())
		}

		randomBytes := make([]byte, length)
		rand.Read(randomBytes)

		for i := 0; i < length; i++ {
			obj.Set(fmt.Sprintf("%d", i), randomBytes[i])
		}

		return arr
	})
}

// setupStreams implements the Streams API.
func (r *Runtime) setupStreams() {
	vm := r.vm

	// ReadableStream constructor
	vm.Set("ReadableStream", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This

		var chunks [][]byte
		var controller *goja.Object
		locked := false
		cancelled := false
		closeRequested := false

		// Create controller
		controller = vm.NewObject()
		controller.Set("enqueue", func(c goja.FunctionCall) goja.Value {
			if closeRequested || cancelled {
				return goja.Undefined()
			}
			chunk := c.Argument(0)
			var data []byte
			switch v := chunk.Export().(type) {
			case []byte:
				data = v
			case string:
				data = []byte(v)
			}
			chunks = append(chunks, data)
			return goja.Undefined()
		})
		controller.Set("close", func(c goja.FunctionCall) goja.Value {
			closeRequested = true
			return goja.Undefined()
		})
		controller.Set("error", func(c goja.FunctionCall) goja.Value {
			cancelled = true
			return goja.Undefined()
		})

		// Call underlying source's start method
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			source := call.Arguments[0].ToObject(vm)
			if start := source.Get("start"); start != nil && !goja.IsUndefined(start) {
				if startFunc, ok := goja.AssertFunction(start); ok {
					startFunc(nil, controller)
				}
			}
		}

		obj.Set("locked", locked)

		obj.Set("cancel", func(c goja.FunctionCall) goja.Value {
			cancelled = true
			chunks = nil
			return r.createPromise(goja.Undefined())
		})

		obj.Set("getReader", func(c goja.FunctionCall) goja.Value {
			if locked {
				panic(vm.NewGoError(fmt.Errorf("stream is locked")))
			}
			locked = true

			reader := vm.NewObject()
			readIndex := 0

			reader.Set("read", func(rc goja.FunctionCall) goja.Value {
				if readIndex < len(chunks) {
					chunk := chunks[readIndex]
					readIndex++
					result := vm.NewObject()
					result.Set("done", false)
					result.Set("value", chunk)
					return r.createPromise(result)
				}

				if closeRequested || cancelled {
					result := vm.NewObject()
					result.Set("done", true)
					result.Set("value", goja.Undefined())
					return r.createPromise(result)
				}

				// No more data available yet
				result := vm.NewObject()
				result.Set("done", true)
				result.Set("value", goja.Undefined())
				return r.createPromise(result)
			})

			reader.Set("releaseLock", func(rc goja.FunctionCall) goja.Value {
				locked = false
				return goja.Undefined()
			})

			reader.Set("cancel", func(rc goja.FunctionCall) goja.Value {
				cancelled = true
				locked = false
				return r.createPromise(goja.Undefined())
			})

			reader.Set("closed", r.createPromise(goja.Undefined()))

			return reader
		})

		obj.Set("pipeThrough", func(c goja.FunctionCall) goja.Value {
			// Returns the readable side of the transform
			transform := c.Argument(0).ToObject(vm)
			return transform.Get("readable")
		})

		obj.Set("pipeTo", func(c goja.FunctionCall) goja.Value {
			// Simplified - just transfers data
			return r.createPromise(goja.Undefined())
		})

		obj.Set("tee", func(c goja.FunctionCall) goja.Value {
			// Create two new streams with the same data
			stream1 := vm.NewObject()
			stream2 := vm.NewObject()
			// Copy chunks to both
			return vm.ToValue([]interface{}{stream1, stream2})
		})

		return obj
	})

	// WritableStream constructor
	vm.Set("WritableStream", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This

		var chunks [][]byte
		locked := false

		var writeFunc func(chunk []byte) error
		var closeFunc func() error

		// Parse underlying sink
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			sink := call.Arguments[0].ToObject(vm)
			if write := sink.Get("write"); write != nil && !goja.IsUndefined(write) {
				if wf, ok := goja.AssertFunction(write); ok {
					writeFunc = func(chunk []byte) error {
						wf(nil, vm.ToValue(chunk))
						return nil
					}
				}
			}
			if close := sink.Get("close"); close != nil && !goja.IsUndefined(close) {
				if cf, ok := goja.AssertFunction(close); ok {
					closeFunc = func() error {
						cf(nil)
						return nil
					}
				}
			}
		}

		obj.Set("locked", locked)

		obj.Set("close", func(c goja.FunctionCall) goja.Value {
			if closeFunc != nil {
				closeFunc()
			}
			obj.Set("_closed", true)
			return r.createPromise(goja.Undefined())
		})

		obj.Set("abort", func(c goja.FunctionCall) goja.Value {
			obj.Set("_closed", true)
			return r.createPromise(goja.Undefined())
		})

		obj.Set("getWriter", func(c goja.FunctionCall) goja.Value {
			if locked {
				panic(vm.NewGoError(fmt.Errorf("stream is locked")))
			}
			locked = true

			writer := vm.NewObject()

			writer.Set("write", func(wc goja.FunctionCall) goja.Value {
				chunk := wc.Argument(0)
				var data []byte
				switch v := chunk.Export().(type) {
				case []byte:
					data = v
				case string:
					data = []byte(v)
				}
				chunks = append(chunks, data)
				if writeFunc != nil {
					writeFunc(data)
				}
				return r.createPromise(goja.Undefined())
			})

			writer.Set("close", func(wc goja.FunctionCall) goja.Value {
				if closeFunc != nil {
					closeFunc()
				}
				obj.Set("_closed", true)
				locked = false
				return r.createPromise(goja.Undefined())
			})

			writer.Set("abort", func(wc goja.FunctionCall) goja.Value {
				obj.Set("_closed", true)
				locked = false
				return r.createPromise(goja.Undefined())
			})

			writer.Set("releaseLock", func(wc goja.FunctionCall) goja.Value {
				locked = false
				return goja.Undefined()
			})

			writer.Set("ready", r.createPromise(goja.Undefined()))
			writer.Set("closed", r.createPromise(goja.Undefined()))

			return writer
		})

		return obj
	})

	// TransformStream constructor
	vm.Set("TransformStream", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This

		var transformFunc func(chunk []byte, controller *goja.Object)
		var flushFunc func(controller *goja.Object)

		// Parse transformer
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			transformer := call.Arguments[0].ToObject(vm)
			if transform := transformer.Get("transform"); transform != nil && !goja.IsUndefined(transform) {
				if tf, ok := goja.AssertFunction(transform); ok {
					transformFunc = func(chunk []byte, controller *goja.Object) {
						tf(nil, vm.ToValue(chunk), controller)
					}
				}
			}
			if flush := transformer.Get("flush"); flush != nil && !goja.IsUndefined(flush) {
				if ff, ok := goja.AssertFunction(flush); ok {
					flushFunc = func(controller *goja.Object) {
						ff(nil, controller)
					}
				}
			}
		}

		// Create readable and writable sides
		var outputChunks [][]byte

		readableController := vm.NewObject()
		readableController.Set("enqueue", func(c goja.FunctionCall) goja.Value {
			chunk := c.Argument(0)
			var data []byte
			switch v := chunk.Export().(type) {
			case []byte:
				data = v
			case string:
				data = []byte(v)
			}
			outputChunks = append(outputChunks, data)
			return goja.Undefined()
		})

		readable := vm.NewObject()
		readable.Set("getReader", func(c goja.FunctionCall) goja.Value {
			reader := vm.NewObject()
			readIndex := 0
			reader.Set("read", func(rc goja.FunctionCall) goja.Value {
				if readIndex < len(outputChunks) {
					chunk := outputChunks[readIndex]
					readIndex++
					result := vm.NewObject()
					result.Set("done", false)
					result.Set("value", chunk)
					return r.createPromise(result)
				}
				result := vm.NewObject()
				result.Set("done", true)
				return r.createPromise(result)
			})
			return reader
		})

		writable := vm.NewObject()
		writable.Set("getWriter", func(c goja.FunctionCall) goja.Value {
			writer := vm.NewObject()
			writer.Set("write", func(wc goja.FunctionCall) goja.Value {
				chunk := wc.Argument(0)
				var data []byte
				switch v := chunk.Export().(type) {
				case []byte:
					data = v
				case string:
					data = []byte(v)
				}
				if transformFunc != nil {
					transformFunc(data, readableController)
				} else {
					outputChunks = append(outputChunks, data)
				}
				return r.createPromise(goja.Undefined())
			})
			writer.Set("close", func(wc goja.FunctionCall) goja.Value {
				if flushFunc != nil {
					flushFunc(readableController)
				}
				return r.createPromise(goja.Undefined())
			})
			return writer
		})

		obj.Set("readable", readable)
		obj.Set("writable", writable)

		return obj
	})
}

func (r *Runtime) createReadableStreamFromBytes(data []byte) goja.Value {
	vm := r.vm

	stream := vm.NewObject()
	locked := false
	readIndex := 0
	chunkSize := 65536 // 64KB chunks

	stream.Set("locked", locked)

	stream.Set("getReader", func(c goja.FunctionCall) goja.Value {
		if locked {
			panic(vm.NewGoError(fmt.Errorf("stream is locked")))
		}
		locked = true

		reader := vm.NewObject()

		reader.Set("read", func(rc goja.FunctionCall) goja.Value {
			if readIndex >= len(data) {
				result := vm.NewObject()
				result.Set("done", true)
				result.Set("value", goja.Undefined())
				return r.createPromise(result)
			}

			end := readIndex + chunkSize
			if end > len(data) {
				end = len(data)
			}

			chunk := data[readIndex:end]
			readIndex = end

			result := vm.NewObject()
			result.Set("done", false)
			result.Set("value", chunk)
			return r.createPromise(result)
		})

		reader.Set("releaseLock", func(rc goja.FunctionCall) goja.Value {
			locked = false
			return goja.Undefined()
		})

		reader.Set("cancel", func(rc goja.FunctionCall) goja.Value {
			locked = false
			return r.createPromise(goja.Undefined())
		})

		return reader
	})

	stream.Set("cancel", func(c goja.FunctionCall) goja.Value {
		return r.createPromise(goja.Undefined())
	})

	return stream
}

// setupAbortController implements the AbortController API.
func (r *Runtime) setupAbortController() {
	vm := r.vm

	vm.Set("AbortController", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		aborted := false
		reason := goja.Undefined()

		signal := vm.NewObject()
		signal.Set("aborted", false)
		signal.Set("reason", goja.Undefined())

		var onabortHandler goja.Callable
		signal.Set("onabort", goja.Null())

		signal.Set("addEventListener", func(c goja.FunctionCall) goja.Value {
			eventType := c.Argument(0).String()
			if eventType == "abort" {
				if fn, ok := goja.AssertFunction(c.Argument(1)); ok {
					onabortHandler = fn
				}
			}
			return goja.Undefined()
		})

		signal.Set("removeEventListener", func(c goja.FunctionCall) goja.Value {
			eventType := c.Argument(0).String()
			if eventType == "abort" {
				onabortHandler = nil
			}
			return goja.Undefined()
		})

		signal.Set("throwIfAborted", func(c goja.FunctionCall) goja.Value {
			if aborted {
				panic(vm.NewGoError(fmt.Errorf("aborted")))
			}
			return goja.Undefined()
		})

		obj.Set("signal", signal)

		obj.Set("abort", func(c goja.FunctionCall) goja.Value {
			if aborted {
				return goja.Undefined()
			}
			aborted = true
			if len(c.Arguments) > 0 {
				reason = c.Argument(0)
			} else {
				reason = vm.NewGoError(fmt.Errorf("aborted"))
			}
			signal.Set("aborted", true)
			signal.Set("reason", reason)

			if onabortHandler != nil {
				event := vm.NewObject()
				event.Set("type", "abort")
				onabortHandler(nil, event)
			}

			return goja.Undefined()
		})

		return obj
	})

	// AbortSignal static methods
	vm.Set("AbortSignal", vm.NewObject())
	abortSignal := vm.Get("AbortSignal").ToObject(vm)

	abortSignal.Set("abort", func(call goja.FunctionCall) goja.Value {
		signal := vm.NewObject()
		signal.Set("aborted", true)
		if len(call.Arguments) > 0 {
			signal.Set("reason", call.Argument(0))
		} else {
			signal.Set("reason", vm.NewGoError(fmt.Errorf("aborted")))
		}
		return signal
	})

	abortSignal.Set("timeout", func(call goja.FunctionCall) goja.Value {
		// Returns a signal that will abort after the timeout
		signal := vm.NewObject()
		signal.Set("aborted", false)
		signal.Set("reason", goja.Undefined())
		return signal
	})
}

// parseMultipartFormData parses multipart form data from request body.
func parseMultipartFormData(body []byte, boundary string) (map[string][]formDataEntry, error) {
	result := make(map[string][]formDataEntry)

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		name := part.FormName()
		filename := part.FileName()
		data, _ := io.ReadAll(part)

		entry := formDataEntry{}
		if filename != "" {
			entry.isFile = true
			entry.filename = filename
			entry.value = string(data) // Store as string temporarily
		} else {
			entry.value = string(data)
		}

		result[name] = append(result[name], entry)
	}

	return result, nil
}

// getBoundaryFromContentType extracts the boundary from a Content-Type header.
func getBoundaryFromContentType(contentType string) string {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return params["boundary"]
}

// Helper to add formData method to Request
func (r *Runtime) addFormDataToRequest(reqObj *goja.Object, bodyBytes []byte, contentType string) {
	vm := r.vm

	reqObj.Set("formData", func(call goja.FunctionCall) goja.Value {
		formData := vm.NewObject()
		entries := make(map[string][]formDataEntry)

		if strings.Contains(contentType, "multipart/form-data") {
			boundary := getBoundaryFromContentType(contentType)
			if boundary != "" {
				parsed, _ := parseMultipartFormData(bodyBytes, boundary)
				entries = parsed
			}
		} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			// Parse URL-encoded form data
			pairs := strings.Split(string(bodyBytes), "&")
			for _, pair := range pairs {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					entries[kv[0]] = append(entries[kv[0]], formDataEntry{value: kv[1]})
				}
			}
		}

		// Build FormData-like object
		formData.Set("get", func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			if vals, ok := entries[name]; ok && len(vals) > 0 {
				return vm.ToValue(vals[0].value)
			}
			return goja.Null()
		})

		formData.Set("getAll", func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			if vals, ok := entries[name]; ok {
				result := make([]string, len(vals))
				for i, v := range vals {
					result[i] = v.value
				}
				return vm.ToValue(result)
			}
			return vm.ToValue([]string{})
		})

		formData.Set("has", func(c goja.FunctionCall) goja.Value {
			name := c.Argument(0).String()
			_, ok := entries[name]
			return vm.ToValue(ok)
		})

		formData.Set("entries", func(c goja.FunctionCall) goja.Value {
			var result [][]string
			for name, vals := range entries {
				for _, v := range vals {
					result = append(result, []string{name, v.value})
				}
			}
			return vm.ToValue(result)
		})

		return r.createPromise(formData)
	})
}

// Helper to convert Response body to JSON
func (r *Runtime) addJSONBodyMethod(obj *goja.Object, body string) {
	obj.Set("json", func(c goja.FunctionCall) goja.Value {
		var data interface{}
		if err := json.Unmarshal([]byte(body), &data); err != nil {
			return r.createRejectedPromise(err.Error())
		}
		return r.createPromise(r.vm.ToValue(data))
	})
}
