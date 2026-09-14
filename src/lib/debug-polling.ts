// Temporary debug utility for polling flow investigation
// Remove after investigation is complete

export const debugPolling = {
  log: (stage: string, data: any) => {
    const timestamp = new Date().toISOString();
    console.log(
      `[POLLING-DEBUG ${timestamp}] ${stage}:`,
      JSON.stringify(data, null, 2)
    );
  },

  error: (stage: string, error: any) => {
    const timestamp = new Date().toISOString();
    console.error(
      `[POLLING-ERROR ${timestamp}] ${stage}:`,
      error
    );
  },

  stages: {
    RESPONSE_RECEIVED: "1_RESPONSE_RECEIVED",
    PLAN_ID_SET: "2_PLAN_ID_SET",
    AGENT_STATE_SET: "3_AGENT_STATE_SET",
    USEEFFECT_RUNS: "4_USEEFFECT_RUNS",
    USEEFFECT_CONDITION_CHECK: "4A_USEEFFECT_CONDITION_CHECK",
    INTERVAL_CREATED: "4B_INTERVAL_CREATED",
    POLLING_ATTEMPT: "5_POLLING_ATTEMPT",
    STATUS_RESPONSE: "6_STATUS_RESPONSE",
    STATUS_CHECK: "7_STATUS_CHECK",
    APPEND_CALLED: "8_APPEND_CALLED",
    MESSAGES_SET: "9_MESSAGES_SET",
  },
};
