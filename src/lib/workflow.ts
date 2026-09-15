import type {
  Task,
  TaskDetailResponse,
  Requirement,
  Template,
  TemplateField,
  MissingField,
  TaskStatus,
  Intent,
  ProcessResult,
  MessageResult,
  ValidateResult,
  DocumentRun,
  DocumentRunResponse,
  DocumentListResponse,
  DocumentSourcesResponse,
  HistoryResponse,
  TaskPriority,
} from "@/lib/types";
import {
  startTask,
  getTask,
  listTasks,
  processTask,
  sendTaskMessage,
  setTaskData,
  validateTask,
  generateDocument,
  getTaskSources,
  listDocuments,
  getDocumentSources,
  listHistory,
  updateTaskStatus,
  deleteTask,
} from "@/lib/api-client";

// Re-export types for backward compatibility
export type {
  Task,
  TaskDetailResponse,
  Requirement,
  Template,
  TemplateField,
  MissingField,
  TaskStatus,
  Intent,
  ProcessResult,
  MessageResult,
  ValidateResult,
  DocumentRun,
  DocumentRunResponse,
  DocumentListResponse,
  DocumentSourcesResponse,
  HistoryResponse,
  TaskPriority,
};

// Re-export API functions for backward compatibility
export {
  startTask,
  getTask,
  listTasks,
  processTask,
  sendTaskMessage,
  setTaskData,
  validateTask,
  generateDocument,
  getTaskSources,
  listDocuments,
  getDocumentSources,
  listHistory,
  updateTaskStatus,
  deleteTask,
};

// Backward compatibility aliases
export const createTask = startTask;
export const sendMessage = sendTaskMessage;
export const runTask = processTask;
