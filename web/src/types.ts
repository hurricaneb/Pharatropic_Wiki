export interface Tag {
  id: number;
  name: string;
  slug: string;
}

export interface User {
  id: number;
  username: string;
  email: string;
  role: string;
  must_change_password?: boolean;
  created_at: string;
}

export interface UserApiKey {
  id: number;
  user_id: number;
  name: string;
  key?: string;
  prefix: string;
  active: boolean;
  expires_at?: string | null;
  last_used_at?: string | null;
  created_at: string;
}

export interface Revision {
  id: number;
  page_id: number;
  title: string;
  content: string;
  comment: string;
  author: string;
  created_at: string;
}

export interface Attachment {
  id: number;
  page_id: number;
  filename: string;
  original_name: string;
  file_path: string;
  mime_type: string;
  file_size: number;
  created_at: string;
}

export interface Page {
  id: number;
  slug: string;
  title: string;
  summary: string;
  content: string;
  is_public: boolean;
  views: number;
  parent_id?: number | null;
  created_at: string;
  updated_at: string;
  tags?: Tag[];
  revisions?: Revision[];
  attachments?: Attachment[];
  parent?: Page | null;
  children?: Page[];
}

export interface CreatePageInput {
  title: string;
  content: string;
  summary?: string;
  comment?: string;
  is_public?: boolean;
  tags?: string[];
  parent_slug?: string;
}

export interface UpdatePageInput {
  title?: string;
  content: string;
  summary?: string;
  comment?: string;
  is_public?: boolean;
  tags?: string[];
  parent_slug?: string;
}
