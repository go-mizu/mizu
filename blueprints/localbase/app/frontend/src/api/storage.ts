import { api } from './client';
import type { Bucket, StorageObject } from '../types';

export interface CreateBucketRequest {
  name: string;
  id?: string;
  public?: boolean;
  file_size_limit?: number;
  allowed_mime_types?: string[];
}

export interface UpdateBucketRequest {
  public?: boolean;
  file_size_limit?: number;
  allowed_mime_types?: string[];
}

export interface ListObjectsRequest {
  prefix?: string;
  limit?: number;
  offset?: number;
  search?: string;
  sortBy?: {
    column: string;
    order: 'asc' | 'desc';
  };
}

export interface SignedUrlResponse {
  signedURL: string;
}

export const storageApi = {
  // Bucket operations
  listBuckets: (): Promise<Bucket[]> => {
    return api.get<Bucket[]>('/storage/v1/bucket');
  },

  getBucket: (id: string): Promise<Bucket> => {
    return api.get<Bucket>(`/storage/v1/bucket/${id}`);
  },

  createBucket: (data: CreateBucketRequest): Promise<Bucket> => {
    return api.post<Bucket>('/storage/v1/bucket', data);
  },

  updateBucket: (id: string, data: UpdateBucketRequest): Promise<Bucket> => {
    return api.put<Bucket>(`/storage/v1/bucket/${id}`, data);
  },

  deleteBucket: (id: string): Promise<void> => {
    return api.delete(`/storage/v1/bucket/${id}`);
  },

  emptyBucket: (id: string): Promise<void> => {
    return api.post(`/storage/v1/bucket/${id}/empty`);
  },

  // Object operations
  listObjects: (bucket: string, options: ListObjectsRequest = {}): Promise<StorageObject[]> => {
    const body: Record<string, unknown> = {
      prefix: options.prefix || '',
      limit: options.limit || 100,
      offset: options.offset || 0,
    };
    // Only include sortBy if explicitly provided
    if (options.sortBy) {
      body.sortBy = options.sortBy;
    }
    // Only include search if explicitly provided
    if (options.search) {
      body.search = options.search;
    }
    return api.post<StorageObject[]>(`/storage/v1/object/list/${bucket}`, body);
  },

  uploadObject: async (bucket: string, path: string, file: File): Promise<any> => {
    return api.uploadFile(`/storage/v1/object/${bucket}/${path}`, file);
  },

  downloadObjectUrl: (bucket: string, path: string): string => {
    return api.getAuthenticatedUrl(`/storage/v1/object/${bucket}/${path}`);
  },

  getPublicUrl: (bucket: string, path: string): string => {
    return `/storage/v1/object/public/${bucket}/${path}`;
  },

  deleteObject: (bucket: string, path: string): Promise<void> => {
    return api.delete(`/storage/v1/object/${bucket}/${path}`);
  },

  deleteObjects: (bucket: string, _paths: string[]): Promise<void> => {
    return api.delete(`/storage/v1/object/${bucket}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
  },

  moveObject: (bucketId: string, sourcePath: string, destPath: string): Promise<void> => {
    return api.post('/storage/v1/object/move', {
      bucketId,
      sourceKey: sourcePath,
      destinationKey: destPath,
    });
  },

  copyObject: (
    sourceBucket: string,
    sourcePath: string,
    destBucket: string,
    destPath: string
  ): Promise<void> => {
    return api.post('/storage/v1/object/copy', {
      sourceKey: `${sourceBucket}/${sourcePath}`,
      destinationKey: `${destBucket}/${destPath}`,
    });
  },

  createSignedUrl: (bucket: string, path: string, expiresIn = 3600): Promise<SignedUrlResponse> => {
    return api.post<SignedUrlResponse>(`/storage/v1/object/sign/${bucket}/${path}`, {
      expiresIn,
    });
  },

  getObjectInfo: (bucket: string, path: string): Promise<StorageObject> => {
    return api.get<StorageObject>(`/storage/v1/object/info/${bucket}/${path}`);
  },

  // Rename object (change path within same bucket)
  renameObject: (bucketId: string, oldPath: string, newPath: string): Promise<void> => {
    return api.post('/storage/v1/object/rename', {
      bucketId,
      oldPath,
      newPath,
    });
  },

  // Delete folder recursively
  deleteFolder: (bucket: string, path: string): Promise<{ deleted: number; files: string[] }> => {
    return api.delete(`/storage/v1/object/folder/${bucket}/${path}`);
  },

  // Get bucket by name (Supabase compatibility)
  getBucketByName: (name: string): Promise<Bucket> => {
    return api.get<Bucket>(`/storage/v1/bucket/name/${name}`);
  },

  // Create upload signed URL
  createUploadSignedURL: (bucket: string, path: string): Promise<{ url: string; token: string }> => {
    return api.post(`/storage/v1/object/upload/sign/${bucket}/${path}`);
  },
};
