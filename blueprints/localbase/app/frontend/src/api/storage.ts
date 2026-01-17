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
    return api.post<StorageObject[]>(`/storage/v1/object/list/${bucket}`, {
      prefix: options.prefix || '',
      limit: options.limit || 100,
      offset: options.offset || 0,
      sortBy: options.sortBy,
    });
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
};
