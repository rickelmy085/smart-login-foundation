import type {
  ChatMessage,
  ChatResponse,
  Task,
  TaskDetailResponse,
  ProcessResult,
  MessageResult,
  DocumentRunResponse,
  HistoryResponse,
  AgentResponse,
  AgentPlan,
  AgentToolResult,
  AgentTrace,
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
  agentProcess,
  agentResume,
  agentHumanInput,
  getAgentStatus,
  listAgentTools,
  getAgentToolSchema,
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
  AgentResponse,
  AgentPlan,
  AgentToolResult,
  AgentTrace,
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
  agentProcess,
  agentResume,
  agentHumanInput,
  getAgentStatus,
  listAgentTools,
  getAgentToolSchema,
};