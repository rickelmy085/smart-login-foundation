// Shared types for the ABIS frontend
// This file consolidates types that were duplicated between chat.ts and workflow.ts

// ---------- Chat types ----------

export type Source = {
  documentId: string;
  title: string;
  source: string;
  chunkOrd: number;
  content: string;
  snippet: string;
  score: number;
};

export type ChatMessage = {
  id: string;
  role: "user" | "assistant";
  content: string;
  sources?: Source[];
  error?: string;
  documentRunId?: string;
  documentDocxUrl?: string;
  documentPdfUrl?: string;
};

export type ChatResponse = {
  answer: string;
  sources: Source[];
  documentRunId?: string;
  documentDocxUrl?: string;
  documentPdfUrl?: string;
};

// ---------- Workflow types ----------

export type TaskStatus =
  | "detected"
  | "collecting_data"
  | "validating"
  | "ready_to_generate"
  | "generating"
  | "generated"
  | "needs_review"
  | "completed"
  | "blocked";

export type Intent =
  | "knowledge_query"
  | "procedure_query"
  | "document_generation"
  | "form_completion"
  | "approval_check"
  | "requirement_check"
  | "workflow_execution"
  | "capability_query";

export type Requirement = {
  id: string;
  name: string;
  label: string;
  required: boolean;
  sourceDocument?: string;
  sourceSnippet?: string;
};

export type TemplateField = {
  fieldName: string;
  label: string;
  type: string;
  required: boolean;
  normativeDocument?: string;
};

export type Template = {
  id: string;
  name: string;
  description: string;
  documentType: string;
  version: string;
  fields: TemplateField[];
};

export type MissingField = {
  fieldName: string;
  label: string;
  required: boolean;
};

export type Task = {
  id: string;
  intent: Intent;
  procedure: string;
  status: TaskStatus;
  originalRequest: string;
  templateId: string;
  requirements: Requirement[];
  data: Record<string, string>;
  template?: Template;
  missingFields?: MissingField[];
  createdAt: string;
  updatedAt: string;
};

export type DocumentRun = {
  id: string;
  templateId: string;
  templateVersion: string;
  status: string;
  docxPath?: string;
  pdfPath?: string;
  createdAt: string;
};

// ---------- Document types ----------

export type DocumentSource = {
  normativeDocument: string;
  snippet: string;
  requirement: string;
};

export type DocumentRunDetail = {
  id: string;
  templateId: string;
  templateVersion: string;
  status: string;
  docxPath?: string;
  pdfPath?: string;
  createdAt: string;
};

// ---------- Response types ----------

export type TaskDetailResponse = {
  task: Task;
  requirements: Requirement[];
  data: Record<string, string>;
  template?: Template;
  missingFields?: MissingField[];
};

export type ProcessResult = {
  requirements: Requirement[];
  sources: unknown[];
  message?: string;
  readyToGenerate: boolean;
  template?: Template;
};

export type MessageResult = {
  answer: string;
  readyToGenerate: boolean;
  missingFields?: MissingField[];
};

export type ValidateResult = {
  answer: string;
  readyToGenerate: boolean;
  missingFields?: MissingField[];
};

export type DocumentRunResponse = {
  documentRunId: string;
  status: string;
  docxPath: string;
  pdfPath?: string;
};

export type DocumentListResponse = {
  documents: DocumentRun[];
};

export type DocumentSourcesResponse = {
  sources: DocumentSource[];
};

export type HistoryItem = {
  id: string;
  type: string;
  title: string;
  preview: string;
  status: string;
  createdAt: string;
  sources: number;
};

export type HistoryResponse = {
  history: HistoryItem[];
};