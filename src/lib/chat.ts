import type {
  ChatMessage,
  ChatResponse,
  Task,
  TaskDetailResponse,
  ProcessResult,
  MessageResult,
  DocumentRunResponse,
  HistoryResponse,
} from "@/lib/types";
import {
  sendQuestion,
  generateDocumentFromChat,
  startTask,
  getTask,
  runTask,
  sendTaskMessage,
  generateDocument,
  listHistory,
} from "@/lib/api-client";

// Re-export types for backward compatibility
export type {
  ChatMessage,
  ChatResponse,
  Task,
  TaskDetailResponse,
  ProcessResult,
  MessageResult,
  DocumentRunResponse,
  HistoryResponse,
};

// Re-export API functions for backward compatibility
export {
  sendQuestion,
  generateDocumentFromChat,
  startTask,
  getTask,
  runTask,
  sendTaskMessage,
  generateDocument,
  listHistory,
};