import type { Store } from '../store/types';
import type { Comment, CreateComment, UpdateComment, CommentWithAuthor } from '../models';
import { generateId } from '../utils/id';

export class CommentService {
  constructor(private store: Store) {}

  async create(input: CreateComment, userId: string): Promise<Comment> {
    return this.store.comments.create({
      id: generateId(),
      workspaceId: input.workspaceId,
      targetType: input.targetType,
      targetId: input.targetId,
      parentId: input.parentId,
      content: input.content,
      authorId: userId,
      isResolved: false,
    });
  }

  async getById(id: string): Promise<Comment | null> {
    return this.store.comments.getById(id);
  }

  async listByTarget(targetType: string, targetId: string): Promise<Comment[]> {
    return this.store.comments.listByTarget(targetType, targetId);
  }

  async listByPage(pageId: string): Promise<Comment[]> {
    return this.store.comments.listByPage(pageId);
  }

  async listWithAuthors(targetType: string, targetId: string): Promise<CommentWithAuthor[]> {
    const comments = await this.store.comments.listByTarget(targetType, targetId);
    const authorIds = [...new Set(comments.map((c) => c.authorId))];

    const authors = new Map<string, { id: string; name: string; avatarUrl?: string | null }>();
    for (const authorId of authorIds) {
      const user = await this.store.users.getById(authorId);
      if (user) {
        authors.set(authorId, { id: user.id, name: user.name, avatarUrl: user.avatarUrl });
      }
    }

    // Build threaded comments
    const commentMap = new Map<string, CommentWithAuthor>();
    const roots: CommentWithAuthor[] = [];

    for (const comment of comments) {
      const author = authors.get(comment.authorId) ?? { id: comment.authorId, name: 'Unknown' };
      commentMap.set(comment.id, { ...comment, author, replies: [] });
    }

    for (const comment of comments) {
      const node = commentMap.get(comment.id)!;
      if (comment.parentId) {
        const parent = commentMap.get(comment.parentId);
        if (parent) {
          parent.replies?.push(node);
        } else {
          roots.push(node);
        }
      } else {
        roots.push(node);
      }
    }

    return roots;
  }

  async update(id: string, data: UpdateComment, userId: string): Promise<Comment> {
    const comment = await this.store.comments.getById(id);
    if (!comment) {
      throw new Error('Comment not found');
    }

    if (comment.authorId !== userId) {
      throw new Error('Permission denied');
    }

    return this.store.comments.update(id, data);
  }

  async resolve(id: string): Promise<Comment> {
    return this.store.comments.update(id, { isResolved: true });
  }

  async unresolve(id: string): Promise<Comment> {
    return this.store.comments.update(id, { isResolved: false });
  }

  async delete(id: string, userId: string): Promise<void> {
    const comment = await this.store.comments.getById(id);
    if (!comment) {
      throw new Error('Comment not found');
    }

    if (comment.authorId !== userId) {
      throw new Error('Permission denied');
    }

    await this.store.comments.delete(id);
  }
}
