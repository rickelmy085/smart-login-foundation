import { apiFetch } from "@/lib/api";

// ---------- Types ----------

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
  | "workflow_execution";

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

// ---------- API ----------

export async function createTask(question: string): Promise<{
  taskId: string;
  status: string;
  message?: string;
}> {
  const res = await apiFetch("/api/tasks", {
    method: "POST",
    body: JSON.stringify({ question }),
  });
  const data = (await res.json()) as { taskId: string; status: string; message?: string };
  return data;
}

export async function getTask(taskId: string): Promise<Task> {
  const res = await apiFetch(`/api/tasks/${taskId}`, {
    method: "GET",
  });
  const data = (await res.json()) as Task;
  return data;
}

export async function listTasks(): Promise<{ tasks: Task[] }> {
  const res = await apiFetch("/api/tasks", {
    method: "GET",
  });
  const data = (await res.json()) as { tasks: Task[] };
  return data;
}

export async function processTask(taskId: string): Promise<{
  taskId: string;
  requirements: Requirement[];
  template?: Template;
  message?: string;
  readyToGenerate: boolean;
  sources: unknown[];
}> {
  const res = await apiFetch(`/api/tasks/${taskId}/process`, {
    method: "POST",
  });
  const data = (await res.json()) as {
    taskId: string;
    requirements: Requirement[];
    template?: Template;
    message?: string;
    readyToGenerate: boolean;
    sources: unknown[];
  };
  return data;
}

export async function sendMessage(
  taskId: string,
  message: string,
): Promise<{
  answer: string;
  taskStatus: string;
  readyToGenerate: boolean;
  missingFields?: MissingField[];
}> {
  const res = await apiFetch(`/api/tasks/${taskId}/message`, {
    method: "POST",
    body: JSON.stringify({ message }),
  });
  const data = (await res.json()) as {
    answer: string;
    taskStatus: string;
    readyToGenerate: boolean;
    missingFields?: MissingField[];
  };
  return data;
}

export async function setTaskData(
  taskId: string,
  fieldName: string,
  value: string,
): Promise<{ status: string }> {
  const res = await apiFetch(`/api/tasks/${taskId}/data`, {
    method: "POST",
    body: JSON.stringify({ fieldName, value }),
  });
  const data = (await res.json()) as { status: string };
  return data;
}

export async function validateTask(taskId: string): Promise<{
  answer: string;
  readyToGenerate: boolean;
  missingFields?: MissingField[];
}> {
  const res = await apiFetch(`/api/tasks/${taskId}/validate`, {
    method: "POST",
  });
  const data = (await res.json()) as {
    answer: string;
    readyToGenerate: boolean;
    missingFields?: MissingField[];
  };
  return data;
}

export async function generateDocument(taskId: string): Promise<{
  documentRunId: string;
  status: string;
  docxPath: string;
  pdfPath: string;
}> {
  const res = await apiFetch(`/api/tasks/${taskId}/generate`, {
    method: "POST",
  });
  const data = (await res.json()) as {
    documentRunId: string;
    status: string;
    docxPath: string;
    pdfPath: string;
  };
  return data;
}

export async function getTaskSources(taskId: string): Promise<{
  sources: Array<{
    title: string;
    snippet: string;
    chunkOrd: number;
    score: number;
    source: string;
  }>;
}> {
  const res = await apiFetch(`/api/tasks/${taskId}/sources`, {
    method: "GET",
  });
  const data = (await res.json()) as {
    sources: Array<{
      title: string;
      snippet: string;
      chunkOrd: number;
      score: number;
      source: string;
    }>;
  };
  return data;
}

export async function listDocuments(): Promise<{ documents: DocumentRun[] }> {
  const res = await apiFetch("/api/documents", {
    method: "GET",
  });
  const data = (await res.json()) as { documents: DocumentRun[] };
  return data;
}

export async function getDocumentSources(
  documentRunId: string,
): Promise<{
  sources: Array<{
    normativeDocument: string;
    snippet: string;
    requirement: string;
  }>;
}> {
  const res = await apiFetch(`/api/documents/${documentRunId}/sources`, {
    method: "GET",
  });
  const data = (await res.json()) as {
    sources: Array<{
      normativeDocument: string;
      snippet: string;
      requirement: string;
    }>;
  };
  return data;
}
