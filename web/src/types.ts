export interface Tag {
  id: number;
  name: string;
  slug: string;
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
  views: number;
  created_at: string;
  updated_at: string;
  tags?: Tag[];
  revisions?: Revision[];
  attachments?: Attachment[];
}

export interface CreatePageInput {
  title: string;
  content: string;
  summary?: string;
  comment?: string;
  tags?: string[];
}

export interface UpdatePageInput {
  title?: string;
  content: string;
  summary?: string;
  comment?: string;
  tags?: string[];
}
