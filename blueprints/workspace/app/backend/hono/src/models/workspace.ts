import { z } from 'zod';

export const WorkspaceSettingsSchema = z.object({
  allowPublicPages: z.boolean().default(false),
  allowGuestInvites: z.boolean().default(false),
  defaultPermission: z.string().default('read'),
  allowedDomains: z.array(z.string()).default([]),
  exportEnabled: z.boolean().default(true),
});

export const WorkspacePlanSchema = z.enum(['free', 'pro', 'team', 'enterprise']);

export const WorkspaceSchema = z.object({
  id: z.string(),
  name: z.string().min(1).max(255),
  slug: z.string().min(1).max(64),
  icon: z.string().optional().nullable(),
  domain: z.string().optional().nullable(),
  plan: WorkspacePlanSchema.default('free'),
  settings: WorkspaceSettingsSchema,
  ownerId: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const CreateWorkspaceSchema = z.object({
  name: z.string().min(1).max(255),
  slug: z.string().min(1).max(64).regex(/^[a-z0-9-]+$/),
  icon: z.string().optional(),
});

export const UpdateWorkspaceSchema = z.object({
  name: z.string().min(1).max(255).optional(),
  slug: z.string().min(1).max(64).regex(/^[a-z0-9-]+$/).optional(),
  icon: z.string().optional().nullable(),
  domain: z.string().optional().nullable(),
  settings: WorkspaceSettingsSchema.partial().optional(),
});

export type Workspace = z.infer<typeof WorkspaceSchema>;
export type WorkspaceSettings = z.infer<typeof WorkspaceSettingsSchema>;
export type WorkspacePlan = z.infer<typeof WorkspacePlanSchema>;
export type CreateWorkspace = z.infer<typeof CreateWorkspaceSchema>;
export type UpdateWorkspace = z.infer<typeof UpdateWorkspaceSchema>;

// Member
export const MemberRoleSchema = z.enum(['owner', 'admin', 'member', 'guest']);

export const MemberSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  userId: z.string(),
  role: MemberRoleSchema.default('member'),
  createdAt: z.string(),
});

export const AddMemberSchema = z.object({
  userId: z.string(),
  role: MemberRoleSchema.optional(),
});

export type Member = z.infer<typeof MemberSchema>;
export type MemberRole = z.infer<typeof MemberRoleSchema>;
export type AddMember = z.infer<typeof AddMemberSchema>;

export const defaultWorkspaceSettings: WorkspaceSettings = {
  allowPublicPages: false,
  allowGuestInvites: false,
  defaultPermission: 'read',
  allowedDomains: [],
  exportEnabled: true,
};
