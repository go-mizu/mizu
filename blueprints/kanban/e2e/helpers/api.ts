interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
}

interface Workspace {
  id: string;
  slug: string;
  name: string;
}

interface Project {
  id: string;
  key: string;
  name: string;
  description?: string;
}

interface Issue {
  id: string;
  key: string;
  title: string;
  description?: string;
  column_id: string;
  priority?: string;
}

interface CreateIssueData {
  title: string;
  description?: string;
  column_id?: string;
  priority?: string;
}

export class ApiHelper {
  private baseUrl: string;
  private token: string | null = null;

  constructor(baseUrl: string = 'http://localhost:8080') {
    this.baseUrl = baseUrl;
  }

  private async request<T>(method: string, path: string, data?: unknown): Promise<T> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers,
      body: data ? JSON.stringify(data) : undefined,
    });

    const json = await response.json();

    if (!response.ok) {
      throw new Error(json.error || 'Request failed');
    }

    return json.data || json;
  }

  async login(email: string, password: string): Promise<{ user: User; token: string }> {
    const response = await this.request<{ user: User; session: { id: string } }>('POST', '/api/v1/auth/login', {
      email,
      password,
    });
    this.token = response.session.id;
    return { user: response.user, token: response.session.id };
  }

  async register(data: {
    username: string;
    email: string;
    display_name: string;
    password: string;
  }): Promise<User> {
    const response = await this.request<{ user: User }>('POST', '/api/v1/auth/register', data);
    return response.user;
  }

  async getCurrentUser(): Promise<User> {
    return this.request<User>('GET', '/api/v1/auth/me');
  }

  async getWorkspaces(): Promise<Workspace[]> {
    return this.request<Workspace[]>('GET', '/api/v1/workspaces');
  }

  async getTeams(workspaceId: string): Promise<{ id: string; name: string }[]> {
    return this.request<{ id: string; name: string }[]>('GET', `/api/v1/workspaces/${workspaceId}/teams`);
  }

  async getProjects(teamId: string): Promise<Project[]> {
    return this.request<Project[]>('GET', `/api/v1/teams/${teamId}/projects`);
  }

  async createIssue(projectId: string, data: CreateIssueData): Promise<Issue> {
    return this.request<Issue>('POST', `/api/v1/projects/${projectId}/issues`, data);
  }

  async getIssues(projectId: string): Promise<Issue[]> {
    return this.request<Issue[]>('GET', `/api/v1/projects/${projectId}/issues`);
  }

  async getIssue(key: string): Promise<Issue> {
    return this.request<Issue>('GET', `/api/v1/issues/${key}`);
  }

  async updateIssue(key: string, data: Partial<Issue>): Promise<Issue> {
    return this.request<Issue>('PATCH', `/api/v1/issues/${key}`, data);
  }

  async deleteIssue(key: string): Promise<void> {
    await this.request<void>('DELETE', `/api/v1/issues/${key}`);
  }

  async moveIssue(key: string, columnId: string, position?: number): Promise<void> {
    await this.request<void>('POST', `/api/v1/issues/${key}/move`, {
      column_id: columnId,
      position: position ?? 0,
    });
  }

  async getCycles(teamId: string): Promise<{ id: string; name: string; status: string }[]> {
    return this.request<{ id: string; name: string; status: string }[]>('GET', `/api/v1/teams/${teamId}/cycles`);
  }

  async createCycle(teamId: string, data: { name: string; start_date: string; end_date: string }): Promise<{ id: string; name: string }> {
    return this.request<{ id: string; name: string }>('POST', `/api/v1/teams/${teamId}/cycles`, data);
  }
}
