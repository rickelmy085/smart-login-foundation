import { apiFetch } from "@/lib/api";

// ---------- types ----------

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
};

export type ChatResponse = {
  answer: string;
  sources: Source[];
};

// ---------- Workflow API ----------

export type Task = {
  id: string;
  intent: string;
  procedure: string;
  status: string;
  originalRequest: string;
  templateId: string;
  requirements: Array<{
    id: string;
    name: string;
    label: string;
    required: boolean;
    sourceDocument?: string;
    sourceSnippet?: string;
  }>;
  data: Record<string, string>;
  missingFields?: Array<{
    fieldName: string;
    label: string;
    required: boolean;
  }>;
  template?: {
    id: string;
    name: string;
    fields: Array<{
      fieldName: string;
      label: string;
      type: string;
      required: boolean;
    }>;
  };
  createdAt: string;
  updatedAt: string;
};

export async function startTask(question: string): Promise<{ taskId: string; status: string; message?: string }> {
  const res = await apiFetch("/api/tasks", {
    method: "POST",
    body: JSON.stringify({ question }),
  });
  const data = (await res.json()) as { taskId: string; status: string; message?: string };
  return data;
}

export async function getTask(taskId: string): Promise<Task> {
  const res = await apiFetch(`/api/tasks/${taskId}`, { method: "GET" });
  return (await res.json()) as Task;
}

export async function runTask(taskId: string): Promise<{
  requirements: Task["requirements"];
  sources: unknown[];
  message?: string;
  readyToGenerate: boolean;
  template?: Task["template"];
}> {
  const res = await apiFetch(`/api/tasks/${taskId}/process`, { method: "POST" });
  const data = (await res.json()) as {
    requirements: Task["requirements"];
    sources: unknown[];
    message?: string;
    readyToGenerate: boolean;
    template?: Task["template"];
  };
  return data;
}

export async function sendTaskMessage(taskId: string, message: string): Promise<{
  answer: string;
  readyToGenerate: boolean;
  missingFields?: Task["missingFields"];
}> {
  const res = await apiFetch(`/api/tasks/${taskId}/message`, {
    method: "POST",
    body: JSON.stringify({ message }),
  });
  const data = (await res.json()) as {
    answer: string;
    readyToGenerate: boolean;
    missingFields?: Task["missingFields"];
  };
  return data;
}

export async function generateDocument(taskId: string): Promise<{
  documentRunId: string;
  status: string;
  docxPath: string;
}> {
  const res = await apiFetch(`/api/tasks/${taskId}/generate`, { method: "POST" });
  const data = (await res.json()) as { documentRunId: string; status: string; docxPath: string };
  return data;
}

// ---------- API ----------

export async function sendQuestion(question: string): Promise<ChatResponse> {
  const res = await apiFetch("/api/chat", {
    method: "POST",
    body: JSON.stringify({ question }),
  });

  const data = (await res.json()) as ChatResponse;
  return data;
}
