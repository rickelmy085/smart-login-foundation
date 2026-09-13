// Consolidated API functions for the ABIS frontend
// This consolidates the duplicated API calls from chat.ts and workflow.ts

import { apiFetch } from "@/lib/api";
import type {
  ChatMessage,
  ChatResponse,
  Task,
  Requirement,
  Template,
  MissingField,
  TaskDetailResponse,
  ProcessResult,
  MessageResult,
  ValidateResult,
  DocumentRunResponse,
  DocumentListResponse,
  DocumentSourcesResponse,
  HistoryResponse,
} from "@/lib/types";

// ---------- Task/Workflow API ----------

export async function startTask(question: string): Promise<{
  taskId: string;
  status: string;
  message?: string;
}> {
  const res = await apiFetch("/api/tasks", {
    method: "POST",
    body: JSON.stringify({ question }),
  });
  return (await res.json()) as { taskId: string; status: string; message?: string };
}

export async function getTask(taskId: string): Promise<TaskDetailResponse> {
  const res = await apiFetch(`/api/tasks/${taskId}`, { method: "GET" });
  return (await res.json()) as TaskDetailResponse;
}

export async function listTasks(): Promise<{ tasks: Task[] }> {
  const res = await apiFetch("/api/tasks", { method: "GET" });
  return (await res.json()) as { tasks: Task[] };
}

export async function processTask(taskId: string): Promise<ProcessResult> {
  const res = await apiFetch(`/api/tasks/${taskId}/process`, { method: "POST" });
  return (await res.json()) as ProcessResult;
}

export async function sendTaskMessage(
  taskId: string,
  message: string
): Promise<MessageResult> {
  const res = await apiFetch(`/api/tasks/${taskId}/message`, {
    method: "POST",
    body: JSON.stringify({ message }),
  });
  return (await res.json()) as MessageResult;
}

export async function setTaskData(
  taskId: string,
  fieldName: string,
  value: string
): Promise<{ status: string }> {
  const res = await apiFetch(`/api/tasks/${taskId}/data`, {
    method: "POST",
    body: JSON.stringify({ fieldName, value }),
  });
  return (await res.json()) as { status: string };
}

export async function validateTask(taskId: string): Promise<ValidateResult> {
  const res = await apiFetch(`/api/tasks/${taskId}/validate`, {
    method: "POST",
  });
  return (await res.json()) as ValidateResult;
}

export async function generateDocument(taskId: string): Promise<DocumentRunResponse> {
  const res = await apiFetch(`/api/tasks/${taskId}/generate`, { method: "POST" });
  return (await res.json()) as DocumentRunResponse;
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
  const res = await apiFetch(`/api/tasks/${taskId}/sources`, { method: "GET" });
  return (await res.json()) as {
    sources: Array<{
      title: string;
      snippet: string;
      chunkOrd: number;
      score: number;
      source: string;
    }>;
  };
}

// ---------- Document API ----------

export async function listDocuments(): Promise<DocumentListResponse> {
  const res = await apiFetch("/api/documents", { method: "GET" });
  return (await res.json()) as DocumentListResponse;
}

export async function getDocumentSources(
  documentRunId: string
): Promise<DocumentSourcesResponse> {
  const res = await apiFetch(`/api/documents/${documentRunId}/sources`, {
    method: "GET",
  });
  return (await res.json()) as DocumentSourcesResponse;
}

export async function downloadDocument(
  path: string,
  filename: string
): Promise<void> {
  const token = getToken();
  const headers: Record<string, string> = {};
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    headers,
    credentials: "omit",
  });

  if (res.status === 401) {
    clearToken();
    throw new Error("unauthorized");
  }

  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || `request failed with status ${res.status}`);
  }

  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

// ---------- Chat API ----------

export async function sendQuestion(
  question: string,
  allowWebSearch = false
): Promise<ChatResponse> {
  const res = await apiFetch("/api/chat", {
    method: "POST",
    body: JSON.stringify({ question, allowWebSearch }),
  });
  return (await res.json()) as ChatResponse;
}

export async function generateDocumentFromChat(question: string): Promise<{
  documentRunId: string;
  docxUrl: string;
  pdfUrl: string;
  message: string;
}> {
  const res = await apiFetch("/api/chat/generate-document", {
    method: "POST",
    body: JSON.stringify({ question }),
  });
  return (await res.json()) as {
    documentRunId: string;
    docxUrl: string;
    pdfUrl: string;
    message: string;
  };
}

// ---------- History API ----------

export async function listHistory(): Promise<HistoryResponse> {
  const res = await apiFetch("/api/history", { method: "GET" });
  return (await res.json()) as HistoryResponse;
}

// ---------- Auth API ----------

export async function login(re: string, password: string, remember = false): Promise<{
  token: string;
  session: { id: string; employeeId: string; expiresAt: string; createdAt: string };
  employee: { id: string; re: string; name: string; role: string; email: string };
}> {
  const res = await apiFetch("/api/login", {
    method: "POST",
    body: JSON.stringify({ re, password, remember }),
  });
  return (await res.json()) as {
    token: string;
    session: { id: string; employeeId: string; expiresAt: string; createdAt: string };
    employee: { id: string; re: string; name: string; role: string; email: string };
  };
}

export async function me(): Promise<{ employee: { id: string; re: string; name: string; role: string; email: string } }> {
  const res = await apiFetch("/api/me", { method: "GET" });
  return (await res.json()) as { employee: { id: string; re: string; name: string; role: string; email: string } };
}

export async function logout(): Promise<{ message: string }> {
  const res = await apiFetch("/api/logout", { method: "POST" });
  return (await res.json()) as { message: string };
}

// ---------- Health API ----------

export async function healthCheck(): Promise<{ status: string }> {
  const res = await apiFetch("/api/health", { method: "GET" });
  return (await res.json()) as { status: string };
}

// Backward compatibility aliases
export const createTask = startTask;
export const sendMessage = sendTaskMessage;
export const runTask = processTask;