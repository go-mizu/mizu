import { z } from 'zod';

export const ShareTypeSchema = z.enum(['user', 'link', 'public', 'domain']);
export const SharePermissionSchema = z.enum(['read', 'comment', 'edit', 'full_access']);

export const ShareSchema = z.object({
  id: z.string(),
  pageId: z.string(),
  type: ShareTypeSchema,
  permission: SharePermissionSchema.default('read'),
  userId: z.string().optional().nullable(),
  token: z.string().optional().nullable(),
  password: z.string().optional().nullable(),
  expiresAt: z.string().optional().nullable(),
  domain: z.string().optional().nullable(),
  createdBy: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const CreateShareSchema = z.object({
  pageId: z.string(),
  type: ShareTypeSchema,
  permission: SharePermissionSchema.optional(),
  userId: z.string().optional(),
  password: z.string().optional(),
  expiresAt: z.string().optional(),
  domain: z.string().optional(),
});

export const UpdateShareSchema = z.object({
  permission: SharePermissionSchema.optional(),
  password: z.string().optional().nullable(),
  expiresAt: z.string().optional().nullable(),
});

export type Share = z.infer<typeof ShareSchema>;
export type ShareType = z.infer<typeof ShareTypeSchema>;
export type SharePermission = z.infer<typeof SharePermissionSchema>;
export type CreateShare = z.infer<typeof CreateShareSchema>;
export type UpdateShare = z.infer<typeof UpdateShareSchema>;

// Favorite
export const FavoriteSchema = z.object({
  id: z.string(),
  userId: z.string(),
  pageId: z.string(),
  position: z.number(),
  createdAt: z.string(),
});

export const CreateFavoriteSchema = z.object({
  pageId: z.string(),
});

export type Favorite = z.infer<typeof FavoriteSchema>;
export type CreateFavorite = z.infer<typeof CreateFavoriteSchema>;

// Activity
export const ActivitySchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  userId: z.string(),
  action: z.string(),
  targetType: z.string(),
  targetId: z.string(),
  metadata: z.record(z.unknown()).default({}),
  createdAt: z.string(),
});

export type Activity = z.infer<typeof ActivitySchema>;

// Notification
export const NotificationSchema = z.object({
  id: z.string(),
  userId: z.string(),
  type: z.string(),
  title: z.string(),
  message: z.string().optional().nullable(),
  metadata: z.record(z.unknown()).default({}),
  isRead: z.boolean().default(false),
  createdAt: z.string(),
});

export type Notification = z.infer<typeof NotificationSchema>;

// Synced Block
export const SyncedBlockSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  sourceBlockId: z.string(),
  content: z.unknown(),
  createdBy: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const CreateSyncedBlockSchema = z.object({
  workspaceId: z.string(),
  sourceBlockId: z.string(),
  content: z.unknown(),
});

export const UpdateSyncedBlockSchema = z.object({
  content: z.unknown(),
});

export type SyncedBlock = z.infer<typeof SyncedBlockSchema>;
export type CreateSyncedBlock = z.infer<typeof CreateSyncedBlockSchema>;
export type UpdateSyncedBlock = z.infer<typeof UpdateSyncedBlockSchema>;
