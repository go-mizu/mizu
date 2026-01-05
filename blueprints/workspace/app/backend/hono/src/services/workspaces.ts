import type { Store } from '../store/types';
import type { Workspace, Member, CreateWorkspace, UpdateWorkspace, AddMember, WorkspaceSettings } from '../models';
import { generateId } from '../utils/id';

export class WorkspaceService {
  constructor(private store: Store) {}

  async create(input: CreateWorkspace, userId: string): Promise<Workspace> {
    // Check if slug is taken
    const existing = await this.store.workspaces.getBySlug(input.slug);
    if (existing) {
      throw new Error('Workspace slug already taken');
    }

    // Create workspace
    const workspace = await this.store.workspaces.create({
      id: generateId(),
      name: input.name,
      slug: input.slug,
      icon: input.icon,
      plan: 'free',
      settings: defaultWorkspaceSettings(),
      ownerId: userId,
    });

    // Add creator as owner member
    await this.store.members.create({
      id: generateId(),
      workspaceId: workspace.id,
      userId,
      role: 'owner',
    });

    return workspace;
  }

  async getById(id: string, userId: string): Promise<Workspace | null> {
    const workspace = await this.store.workspaces.getById(id);
    if (!workspace) return null;

    // Check membership
    const member = await this.store.members.getByWorkspaceAndUser(id, userId);
    if (!member) return null;

    return workspace;
  }

  async getBySlug(slug: string, userId: string): Promise<Workspace | null> {
    const workspace = await this.store.workspaces.getBySlug(slug);
    if (!workspace) return null;

    // Check membership
    const member = await this.store.members.getByWorkspaceAndUser(workspace.id, userId);
    if (!member) return null;

    return workspace;
  }

  async listByUser(userId: string): Promise<Workspace[]> {
    return this.store.workspaces.listByUser(userId);
  }

  async update(id: string, data: UpdateWorkspace, userId: string): Promise<Workspace> {
    // Check membership and permission
    const member = await this.store.members.getByWorkspaceAndUser(id, userId);
    if (!member || !['owner', 'admin'].includes(member.role)) {
      throw new Error('Permission denied');
    }

    // Check slug uniqueness if changing
    if (data.slug) {
      const existing = await this.store.workspaces.getBySlug(data.slug);
      if (existing && existing.id !== id) {
        throw new Error('Workspace slug already taken');
      }
    }

    return this.store.workspaces.update(id, data);
  }

  async delete(id: string, userId: string): Promise<void> {
    // Check ownership
    const workspace = await this.store.workspaces.getById(id);
    if (!workspace || workspace.ownerId !== userId) {
      throw new Error('Permission denied');
    }

    await this.store.workspaces.delete(id);
  }

  async getMembers(workspaceId: string, userId: string): Promise<Member[]> {
    // Check membership
    const member = await this.store.members.getByWorkspaceAndUser(workspaceId, userId);
    if (!member) {
      throw new Error('Permission denied');
    }

    return this.store.members.listByWorkspace(workspaceId);
  }

  async addMember(workspaceId: string, input: AddMember, userId: string): Promise<Member> {
    // Check membership and permission
    const member = await this.store.members.getByWorkspaceAndUser(workspaceId, userId);
    if (!member || !['owner', 'admin'].includes(member.role)) {
      throw new Error('Permission denied');
    }

    // Check if user is already a member
    const existing = await this.store.members.getByWorkspaceAndUser(workspaceId, input.userId);
    if (existing) {
      throw new Error('User is already a member');
    }

    return this.store.members.create({
      id: generateId(),
      workspaceId,
      userId: input.userId,
      role: input.role ?? 'member',
    });
  }

  async removeMember(workspaceId: string, memberId: string, userId: string): Promise<void> {
    // Check membership and permission
    const member = await this.store.members.getByWorkspaceAndUser(workspaceId, userId);
    if (!member || !['owner', 'admin'].includes(member.role)) {
      throw new Error('Permission denied');
    }

    // Cannot remove owner
    const targetMember = await this.store.members.getById(memberId);
    if (targetMember?.role === 'owner') {
      throw new Error('Cannot remove workspace owner');
    }

    await this.store.members.delete(memberId);
  }

  async checkMembership(workspaceId: string, userId: string): Promise<Member | null> {
    return this.store.members.getByWorkspaceAndUser(workspaceId, userId);
  }
}

function defaultWorkspaceSettings(): WorkspaceSettings {
  return {
    allowPublicPages: false,
    allowGuestInvites: false,
    defaultPermission: 'read',
    allowedDomains: [],
    exportEnabled: true,
  };
}
