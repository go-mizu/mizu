import { describe, it, expect, vi, beforeEach } from 'vitest'
import { api } from '../../src/api/client'

describe('ApiClient', () => {
  beforeEach(() => {
    vi.mocked(global.fetch).mockReset()
  })

  describe('GET requests', () => {
    it('should make a GET request with correct headers', async () => {
      const mockResponse = { id: '123', title: 'Test Page' }
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        text: async () => JSON.stringify(mockResponse),
      } as Response)

      const result = await api.get('/pages/123')

      expect(global.fetch).toHaveBeenCalledWith('/api/v1/pages/123', {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
      })
      expect(result).toEqual(mockResponse)
    })

    it('should handle empty responses', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        text: async () => '',
      } as Response)

      const result = await api.get('/pages/123')

      expect(result).toEqual({})
    })
  })

  describe('POST requests', () => {
    it('should make a POST request with body', async () => {
      const requestData = { title: 'New Page' }
      const mockResponse = { id: '456', title: 'New Page' }
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        text: async () => JSON.stringify(mockResponse),
      } as Response)

      const result = await api.post('/pages', requestData)

      expect(global.fetch).toHaveBeenCalledWith('/api/v1/pages', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify(requestData),
      })
      expect(result).toEqual(mockResponse)
    })

    it('should allow POST without body', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        text: async () => JSON.stringify({}),
      } as Response)

      await api.post('/pages/123/duplicate')

      expect(global.fetch).toHaveBeenCalledWith('/api/v1/pages/123/duplicate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
      })
    })
  })

  describe('PUT requests', () => {
    it('should make a PUT request with body', async () => {
      const requestData = { blocks: [] }
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        text: async () => JSON.stringify({}),
      } as Response)

      await api.put('/pages/123/blocks', requestData)

      expect(global.fetch).toHaveBeenCalledWith('/api/v1/pages/123/blocks', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify(requestData),
      })
    })
  })

  describe('PATCH requests', () => {
    it('should make a PATCH request with body', async () => {
      const requestData = { title: 'Updated Title' }
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        text: async () => JSON.stringify({}),
      } as Response)

      await api.patch('/pages/123', requestData)

      expect(global.fetch).toHaveBeenCalledWith('/api/v1/pages/123', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify(requestData),
      })
    })
  })

  describe('DELETE requests', () => {
    it('should make a DELETE request', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        text: async () => '',
      } as Response)

      await api.delete('/pages/123')

      expect(global.fetch).toHaveBeenCalledWith('/api/v1/pages/123', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
      })
    })
  })

  describe('Error handling', () => {
    it('should throw an error with message from response', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        json: async () => ({ error: 'Not found' }),
        statusText: 'Not Found',
      } as Response)

      await expect(api.get('/pages/999')).rejects.toThrow('Not found')
    })

    it('should use statusText when no error message in response', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        json: async () => {
          throw new Error('Invalid JSON')
        },
        statusText: 'Internal Server Error',
      } as Response)

      await expect(api.get('/pages/error')).rejects.toThrow('Internal Server Error')
    })

    it('should use default message when no error info available', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        json: async () => {
          throw new Error('Invalid JSON')
        },
        statusText: '',
      } as Response)

      await expect(api.get('/pages/error')).rejects.toThrow('Request failed')
    })
  })

  describe('File upload', () => {
    it('should upload a file with FormData', async () => {
      const mockFile = new File(['test content'], 'test.txt', { type: 'text/plain' })
      const mockResponse = { id: 'file-123', url: '/uploads/test.txt' }

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response)

      const result = await api.upload(mockFile)

      expect(global.fetch).toHaveBeenCalledWith('/api/v1/media/upload', {
        method: 'POST',
        body: expect.any(FormData),
        credentials: 'same-origin',
      })
      expect(result).toEqual(mockResponse)
    })

    it('should use custom upload path', async () => {
      const mockFile = new File(['test'], 'image.png', { type: 'image/png' })
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: '1', url: '/images/1.png' }),
      } as Response)

      await api.upload(mockFile, '/images/upload')

      expect(global.fetch).toHaveBeenCalledWith('/api/v1/images/upload', expect.anything())
    })

    it('should throw error on upload failure', async () => {
      const mockFile = new File(['test'], 'test.txt', { type: 'text/plain' })
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
      } as Response)

      await expect(api.upload(mockFile)).rejects.toThrow('Upload failed')
    })
  })
})
