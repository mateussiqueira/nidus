export interface StackRunClientOptions {
  apiUrl?: string;
  token?: string;
}

export interface Project {
  id: string;
  name: string;
  [key: string]: unknown;
}

export interface Deployment {
  id: string;
  projectId: string;
  status: string;
  [key: string]: unknown;
}

export interface Domain {
  id: string;
  domain: string;
  [key: string]: unknown;
}

export interface Database {
  id: string;
  name: string;
  [key: string]: unknown;
}

export interface Env {
  key: string;
  value: string;
  [key: string]: unknown;
}

export interface AuthResponse {
  token: string;
  user: Record<string, unknown>;
}

export class StackRunClient {
  #apiUrl: string;
  #token: string | null;

  constructor(options: StackRunClientOptions = {}) {
    this.#apiUrl = options.apiUrl ?? "https://api.stackrun.vercel.app";
    this.#token = options.token ?? null;
  }

  setToken(token: string): void {
    this.#token = token;
  }

  async #request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    if (this.#token) {
      headers["Authorization"] = `Bearer ${this.#token}`;
    }

    const response = await fetch(`${this.#apiUrl}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const errorBody = await response.text();
      throw new Error(
        `StackRun API error ${response.status}: ${errorBody || response.statusText}`
      );
    }

    return response.json() as Promise<T>;
  }

  projects = {
    list: (): Promise<Project[]> => {
      return this.#request<Project[]>("GET", "/api/projects");
    },

    get: (id: string): Promise<Project> => {
      return this.#request<Project>("GET", `/api/projects/${id}`);
    },

    create: (data: Record<string, unknown>): Promise<Project> => {
      return this.#request<Project>("POST", "/api/projects", data);
    },

    deploy: (id: string, branch?: string): Promise<Deployment> => {
      return this.#request<Deployment>(
        "POST",
        `/api/projects/${id}/deploy`,
        branch ? { branch } : undefined
      );
    },

    envs: (id: string): Promise<Env[]> => {
      return this.#request<Env[]>("GET", `/api/projects/${id}/envs`);
    },

    envSet: (id: string, key: string, value: string): Promise<Env> => {
      return this.#request<Env>("POST", `/api/projects/${id}/envs`, {
        key,
        value,
      });
    },
  };

  deployments = {
    list: (projectId: string): Promise<Deployment[]> => {
      return this.#request<Deployment[]>(
        "GET",
        `/api/projects/${projectId}/deployments`
      );
    },

    logs: (projectId: string, deploymentId: string): Promise<string> => {
      return this.#request<string>(
        "GET",
        `/api/projects/${projectId}/deployments/${deploymentId}/logs`
      );
    },
  };

  domains = {
    list: (projectId: string): Promise<Domain[]> => {
      return this.#request<Domain[]>(
        "GET",
        `/api/projects/${projectId}/domains`
      );
    },

    add: (projectId: string, domain: string): Promise<Domain> => {
      return this.#request<Domain>(
        "POST",
        `/api/projects/${projectId}/domains`,
        { domain }
      );
    },
  };

  databases = {
    list: (): Promise<Database[]> => {
      return this.#request<Database[]>("GET", "/api/databases");
    },

    create: (name: string, projectId: string): Promise<Database> => {
      return this.#request<Database>("POST", "/api/databases", {
        name,
        projectId,
      });
    },
  };

  auth = {
    login: (email: string, password: string): Promise<AuthResponse> => {
      return this.#request<AuthResponse>("POST", "/api/auth/login", {
        email,
        password,
      });
    },

    register: (
      email: string,
      password: string,
      name: string
    ): Promise<AuthResponse> => {
      return this.#request<AuthResponse>("POST", "/api/auth/register", {
        email,
        password,
        name,
      });
    },
  };
}
