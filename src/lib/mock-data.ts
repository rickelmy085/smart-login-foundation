export type QueryStatus = "concluida" | "em_andamento" | "arquivada";

export type HistoryItem = {
  id: string;
  title: string;
  preview: string;
  topic: string;
  status: QueryStatus;
  createdAt: string; // ISO
  sources: number;
};

export const historyItems: HistoryItem[] = [
  {
    id: "q-1024",
    title: "Política de reembolso de despesas de viagem",
    preview: "Quais são os limites diários para hospedagem e alimentação em viagens nacionais?",
    topic: "RH & Benefícios",
    status: "concluida",
    createdAt: "2026-09-04T18:42:00Z",
    sources: 3,
  },
  {
    id: "q-1023",
    title: "Prazo de fechamento contábil mensal",
    preview: "Até que dia útil os lançamentos devem ser enviados para o fechamento?",
    topic: "Financeiro",
    status: "concluida",
    createdAt: "2026-09-04T14:10:00Z",
    sources: 2,
  },
  {
    id: "q-1022",
    title: "Checklist de onboarding de novos colaboradores",
    preview: "Resumo dos passos obrigatórios da primeira semana.",
    topic: "RH & Benefícios",
    status: "em_andamento",
    createdAt: "2026-09-03T11:25:00Z",
    sources: 5,
  },
  {
    id: "q-1021",
    title: "Padrão de nomenclatura de projetos",
    preview: "Como nomear repositórios e pastas conforme o guia de governança?",
    topic: "TI & Governança",
    status: "concluida",
    createdAt: "2026-09-02T09:05:00Z",
    sources: 1,
  },
  {
    id: "q-1020",
    title: "Procedimento para solicitação de acesso a sistemas",
    preview: "Fluxo de aprovação e SLA para novos acessos.",
    topic: "TI & Governança",
    status: "concluida",
    createdAt: "2026-09-01T16:48:00Z",
    sources: 4,
  },
  {
    id: "q-1019",
    title: "Regras de comunicação com clientes",
    preview: "Tom de voz e canais oficiais recomendados.",
    topic: "Comercial",
    status: "arquivada",
    createdAt: "2026-08-28T13:30:00Z",
    sources: 2,
  },
  {
    id: "q-1018",
    title: "Calendário de feriados e pontes 2026",
    preview: "Quais datas são consideradas ponto facultativo?",
    topic: "RH & Benefícios",
    status: "concluida",
    createdAt: "2026-08-26T10:12:00Z",
    sources: 1,
  },
];

export const suggestedPrompts = [
  "Resuma a política de home office em 5 pontos",
  "Quais documentos preciso para solicitar férias?",
  "Explique o fluxo de aprovação de compras acima de R$ 10 mil",
  "Como abrir um chamado para o suporte de TI?",
];

export const weeklyActivity = [
  { day: "Seg", consultas: 4 },
  { day: "Ter", consultas: 7 },
  { day: "Qua", consultas: 5 },
  { day: "Qui", consultas: 9 },
  { day: "Sex", consultas: 6 },
  { day: "Sáb", consultas: 1 },
  { day: "Dom", consultas: 0 },
];

export const topicColors: Record<string, string> = {
  "RH & Benefícios": "bg-chart-2/15 text-chart-2",
  Financeiro: "bg-chart-4/20 text-warning-foreground dark:text-chart-4",
  "TI & Governança": "bg-chart-1/15 text-chart-1",
  Comercial: "bg-chart-3/15 text-chart-3",
};

export const statusLabels: Record<QueryStatus, string> = {
  concluida: "Concluída",
  em_andamento: "Em andamento",
  arquivada: "Arquivada",
};
