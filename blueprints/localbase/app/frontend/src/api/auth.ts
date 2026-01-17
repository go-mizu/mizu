import { api } from './client';
import type { User } from '../types';

export interface CreateUserRequest {
  email: string;
  password: string;
  phone?: string;
  user_metadata?: Record<string, any>;
  app_metadata?: Record<string, any>;
  email_confirm?: boolean;
}

export interface UpdateUserRequest {
  email?: string;
  phone?: string;
  password?: string;
  user_metadata?: Record<string, any>;
  app_metadata?: Record<string, any>;
  ban_duration?: string;
}

export interface ListUsersResponse {
  users: User[];
  aud: string;
}

export const authApi = {
  // List all users (admin)
  listUsers: async (page = 1, perPage = 50): Promise<{ users: User[]; total: number }> => {
    const response = await api.get<ListUsersResponse>(
      `/auth/v1/admin/users?page=${page}&per_page=${perPage}`
    );
    // The API returns { users: [...], aud: "..." }
    return {
      users: response.users || [],
      total: response.users?.length || 0,
    };
  },

  // Get user by ID (admin)
  getUser: (id: string): Promise<User> => {
    return api.get<User>(`/auth/v1/admin/users/${id}`);
  },

  // Create user (admin)
  createUser: (data: CreateUserRequest): Promise<User> => {
    return api.post<User>('/auth/v1/admin/users', data);
  },

  // Update user (admin)
  updateUser: (id: string, data: UpdateUserRequest): Promise<User> => {
    return api.put<User>(`/auth/v1/admin/users/${id}`, data);
  },

  // Delete user (admin)
  deleteUser: (id: string): Promise<void> => {
    return api.delete(`/auth/v1/admin/users/${id}`);
  },
};
