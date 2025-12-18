export type ReposGetRequest = { owner: string; repo: string }

export type UserSummary = {
  login: string
  id: number
  avatar_url: string
  html_url: string
  type: "User" | "Organization"
}

export type Repo = {
  id: number
  node_id: string
  name: string
  full_name: string
  private: boolean
  html_url: string
  description: string | null
  fork: boolean
  owner: UserSummary
  stargazers_count: number
  forks_count: number
  open_issues_count: number
  default_branch: string
}

export type IssuesCreateRequest = {
  owner: string
  repo: string
  title: string
  body?: string
  assignees?: string[]
  labels?: string[]
}

export type PullRequestRef = {
  url: string
  html_url: string
  diff_url: string
  patch_url: string
}

export type Issue = {
  id: number
  number: number
  title: string
  state: "open" | "closed"
  locked: boolean
  user: UserSummary
  body: string | null
  comments: number
  html_url: string
  pull_request?: PullRequestRef
}
