import type { Store } from '../store/types';
import type { Share, CreateShare, UpdateShare } from '../models';
import { generateId, generateToken } from '../utils/id';

export class ShareService {
  constructor(private store: Store) {}

  async create(input: CreateShare, userId: string): Promise<Share> {
    const share: Omit<Share, 'createdAt' | 'updatedAt'> = {
      id: generateId(),
      pageId: input.pageId,
      type: input.type,
      permission: input.permission ?? 'read',
      userId: input.userId,
      password: input.password,
      expiresAt: input.expiresAt,
      domain: input.domain,
      createdBy: userId,
    };

    // Generate token for link shares
    if (input.type === 'link') {
      share.token = generateToken();
    }

    return this.store.shares.create(share);
  }

  async getById(id: string): Promise<Share | null> {
    return this.store.shares.getById(id);
  }

  async getByToken(token: string): Promise<Share | null> {
    const share = await this.store.shares.getByToken(token);
    if (!share) return null;

    // Check expiration
    if (share.expiresAt && new Date(share.expiresAt) < new Date()) {
      return null;
    }

    return share;
  }

  async listByPage(pageId: string): Promise<Share[]> {
    return this.store.shares.listByPage(pageId);
  }

  async update(id: string, data: UpdateShare): Promise<Share> {
    return this.store.shares.update(id, data);
  }

  async delete(id: string): Promise<void> {
    await this.store.shares.delete(id);
  }

  async checkAccess(pageId: string, userId: string): Promise<Share['permission'] | null> {
    const shares = await this.store.shares.listByPage(pageId);

    // Check for user share
    const userShare = shares.find((s) => s.type === 'user' && s.userId === userId);
    if (userShare) return userShare.permission;

    // Check for public share
    const publicShare = shares.find((s) => s.type === 'public');
    if (publicShare) return publicShare.permission;

    return null;
  }

  async validateLinkAccess(token: string, password?: string): Promise<{ share: Share; pageId: string } | { error: 'not_found' | 'expired' | 'password_required' } > {
    const share = await this.store.shares.getByToken(token);
    if (!share) return { error: 'not_found' };

    // Check expiration
    if (share.expiresAt && new Date(share.expiresAt) < new Date()) {
      return { error: 'expired' };
    }

    // Check password if required
    if (share.password && share.password !== password) {
      return { error: 'password_required' };
    }

    return { share, pageId: share.pageId };
  }
}
