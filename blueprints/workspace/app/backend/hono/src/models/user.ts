import { z } from 'zod';

export const UserSettingsSchema = z.object({
  theme: z.enum(['light', 'dark', 'system']).default('system'),
  timezone: z.string().default('UTC'),
  dateFormat: z.string().default('MM/DD/YYYY'),
  startOfWeek: z.number().min(0).max(6).default(0),
  emailDigest: z.boolean().default(true),
  desktopNotify: z.boolean().default(true),
});

export const UserSchema = z.object({
  id: z.string(),
  email: z.string().email(),
  name: z.string().min(1).max(255),
  avatarUrl: z.string().url().optional().nullable(),
  passwordHash: z.string(),
  settings: UserSettingsSchema,
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const CreateUserSchema = z.object({
  email: z.string().email(),
  name: z.string().min(1).max(255),
  password: z.string().min(8).max(128),
});

export const UpdateUserSchema = z.object({
  name: z.string().min(1).max(255).optional(),
  avatarUrl: z.string().url().optional().nullable(),
  settings: UserSettingsSchema.partial().optional(),
});

export const LoginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
});

export type User = z.infer<typeof UserSchema>;
export type UserSettings = z.infer<typeof UserSettingsSchema>;
export type CreateUser = z.infer<typeof CreateUserSchema>;
export type UpdateUser = z.infer<typeof UpdateUserSchema>;
export type LoginInput = z.infer<typeof LoginSchema>;

// Session
export const SessionSchema = z.object({
  id: z.string(),
  userId: z.string(),
  expiresAt: z.string(),
  createdAt: z.string(),
});

export type Session = z.infer<typeof SessionSchema>;

// Public user (without password hash)
export function toPublicUser(user: User): Omit<User, 'passwordHash'> {
  const { passwordHash: _, ...publicUser } = user;
  return publicUser;
}

export const defaultUserSettings: UserSettings = {
  theme: 'system',
  timezone: 'UTC',
  dateFormat: 'MM/DD/YYYY',
  startOfWeek: 0,
  emailDigest: true,
  desktopNotify: true,
};
