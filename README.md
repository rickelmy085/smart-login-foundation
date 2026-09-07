# Smart Login Foundation

**ABIS** — *Agentic Banking Intelligence System*: um assistente inteligente corporativo para o contexto bancário da Organização Bradesco.

Este repositório contém o MVP completo do ABIS, incluindo autenticação, dashboard, chat com RAG (busca semântica em normativos), geração de documentos e histórico de interações.

> Construído com [Lovable](https://lovable.dev).

## Índice

- [Funcionalidades](#o-que-tem-no-projeto)
- [Tecnologias](#stack-principal)
- [Backend (Go)](#backend-go)
- [Frontend (React + TypeScript)](#frontend-react--typescript)
- [Como rodar](#como-rodar)
- [Usuário demo](#usuário-demo)
- [Desenvolvimento](#desenvolvimento)

## O que tem no projeto

A aplicação ABIS já inclui:

- **Login corporativo** (`/`): tela split screen responsiva com login simulado usando RE e senha, lembrança de sessão e redirecionamento automático para a área logada.
- **Dashboard** (`/app/`): visão geral com saudação por horário, indicadores semanais, gráfico de atividade e sugestões de perguntas. Exibe interações recentes do usuário (chat e tarefas).
- **ABIS Copilot** (`/app/copilot`): chat inteligente com renderização de markdown (negrito, listas, blocos de código com destaque de sintaxe), fontes normativas citadas, e geração automática de documentos (DOCX e PDF) quando o usuário solicita.
- **Histórico** (`/app/historico`): lista de consultas, tarefas e documentos anteriores agrupados por dia, com filtro por status e busca.
- **Tarefas** (`/app/tarefas`): fluxo de trabalho completo para criação, coleta de dados, validação e geração de documentos a partir de templates.
- **Perfil** (`/app/perfil`): dados do colaborador e preferências, incluindo tema claro/escuro/sistema e toggles de notificações.
- **Design system**: tema claro/escuro persistido em `localStorage`, tipografia com Manrope e Sora, utilitários visuais customizados e componentes acessíveis baseados em shadcn/ui.
- **Chat com RAG**: o pipeline de chat usa busca FTS5 (BM25) em normativos internos + LLM (Groq) + fallback de busca web. Mensagens do usuário e respostas do assistente são persistidas no banco e aparecem no histórico.
- **Geração de documentos**: quando o usuário pede um documento (ex: "documento de empréstimo de 10 mil"), o sistema extrai dados reais do funcionário logado, preenche um template e gera arquivos DOCX e PDF para download.
- **Persistência de histórico de chat**: mensagens individais são salvas na tabela `chat_messages` e aparecem no histórico unificado via `history_view`.

## Stack principal

- **Frontend**: React 19, TypeScript, TanStack Router, TanStack Query, Tailwind CSS v4, Vite + TanStack Start/Nitro, Radix UI, `lucide-react`, `react-markdown`, `remark-gfm`, `rehype-highlight`, `sonner`.
- **Backend**: Go, HTTP nativo (`net/http`), SQLite (pure-Go via `modernc.org/sqlite`), JWT para autenticação, Groq API para LLM.
- **RAG**: FTS5 com BM25, extração de texto de PDFs, chunking, re-ranking e expansão de termos com sinônimos do domínio bancário.

## Backend (Go)

O backend do ABIS fica na pasta `backend/` e expõe a API usada pelo frontend.

### Estrutura

- `backend/main.go`: entrada do servidor HTTP, registro de rotas e wiring de dependências.
- `backend/internal/config`: carrega variáveis de ambiente e configura porta, banco, JWT e Groq.
- `backend/internal/database`: conexão com SQLite, migrations e seeds.
- `backend/internal/models`: contratos de request/response e entidades (Employee, Session, Task, DocumentRun, etc.).
- `backend/internal/repository`: acesso a dados (employees, sessions, chat_messages, workflow_tasks, document_runs, templates).
- `backend/internal/service`: regras de negócio (auth, workflow, RAG).
- `backend/internal/handler`: rotas HTTP (auth, chat, knowledge, workflow).
- `backend/internal/middleware`: injeção de segredo JWT e proteção de rotas.
- `backend/internal/document`: geração de arquivos DOCX e PDF.
- `backend/internal/knowledge`: pipeline de ingestão e busca de normativos (scanner, extractor, chunker, FTS5).
- `backend/internal/groq`: cliente HTTP mínimo para a API Groq.
- `backend/internal/chat`: pipeline RAG (busca → prompt → LLM → resposta).
- `backend/internal/web`: cliente de busca web (fallback).

### Endpoints

#### Autenticação
- `POST /api/login`: autentica RE e senha, retorna token JWT.
- `GET /api/me`: retorna dados do funcionário autenticado.
- `POST /api/logout`: encerra a sessão.

#### Chat (RAG)
- `POST /api/chat`: recebe uma pergunta, executa o pipeline RAG (busca FTS5 + LLM) e devolve a resposta com fontes. Se a pergunta contém palavras como "documento", gera automaticamente um DOCX/PDF.
- `POST /api/chat/generate-document`: gera um documento sob demanda a partir de uma pergunta.
- `GET /api/search`: busca normativos via FTS5 (BM25).

#### Workflow (tarefas e documentos)
- `POST /api/tasks`: cria uma nova tarefa a partir de uma pergunta.
- `GET /api/tasks/{id}`: retorna detalhes da tarefa.
- `POST /api/tasks/{id}/process`: inicia o processamento da tarefa.
- `POST /api/tasks/{id}/message`: envia uma mensagem para coleta de dados da tarefa.
- `POST /api/tasks/{id}/data`: define dados da tarefa.
- `POST /api/tasks/{id}/validate`: valida a tarefa.
- `POST /api/tasks/{id}/generate`: gera o documento da tarefa.
- `GET /api/tasks/{id}/sources`: retorna fontes usadas na tarefa.
- `GET /api/tasks`: lista todas as tarefas do usuário.
- `GET /api/documents`: lista documentos gerados.
- `GET /api/documents/{id}/sources`: retorna fontes do documento.
- `GET /api/documents/{id}/docx`: download do DOCX.
- `GET /api/documents/{id}/pdf`: download do PDF.
- `GET /api/history`: retorna o histórico unificado (chat, tarefas e documentos) do usuário.

### Como rodar

```sh
cd backend
go mod tidy
go run .
```

Por padrão o servidor sobe em `http://localhost:8081`.

### Variáveis de ambiente

- `PORT`: porta do servidor (padrão: `8081`).
- `DATABASE_PATH`: caminho do SQLite (padrão: `data/abis.db`).
- `JWT_SECRET`: segredo para assinar tokens JWT.
- `GROQ_API_KEY`: chave da API Groq (para usar o LLM).
- `GROQ_MODEL`: modelo da Groq (padrão: `llama-3.3-70b-versatile`).

### Usuário demo

- RE: `123456`
- Nome: `José Alberto`
- Senha: `demo123`
- Email: `jose.alberto@bradesco.com.br`
- Cargo: `Analista`

## Frontend (React + TypeScript)

### Como rodar em desenvolvimento

```sh
git clone <this-repository-url>
cd <repository-name>
npm install
npm run dev
```

O frontend sobe em `http://localhost:8080` (ou porta indicada pelo Vite).

### Comandos úteis

| Comando          | Descrição                                      |
| ---------------- | ---------------------------------------------- |
| `npm run dev`    | Inicia o servidor de desenvolvimento Vite.     |
| `npm run build`  | Gera a build de produção (Nitro).              |
| `npm run lint`   | Executa o ESLint.                              |
| `npm run format` | Formata o código com Prettier.                 |

## Desenvolvimento

Preferindo trabalhar localmente? Você precisa do Node.js e npm — [instale com nvm](https://github.com/nvm-sh/nvm#installing-and-updating).

### Conectando frontend e backend

1. Inicie o backend: `cd backend && go run .`
2. Inicie o frontend: `npm run dev`
3. Acesse `http://localhost:8080`, faça login com RE `123456` e senha `demo123`.

> Se o backend estiver em outra porta, configure `VITE_API_URL` no `.env` do frontend (ex: `VITE_API_URL=http://localhost:8081`).

### Estrutura de diretórios (frontend)

```
src/
├── routes/                  # Páginas (TanStack Router)
│   ├── index.tsx            # Login
│   ├── app.tsx              # Shell do app (guarda de sessão, sidebar, header)
│   ├── app.index.tsx        # Dashboard
│   ├── app.copilot.tsx      # Chat ABIS
│   ├── app.historico.tsx    # Histórico
│   ├── app.perfil.tsx       # Perfil
│   ├── app.tarefas.tsx      # Lista de tarefas
│   └── app.tarefas.$id.tsx  # Detalhe da tarefa
├── components/              # Componentes UI reutilizáveis
├── lib/                     # API client, auth, tipos
└── styles.css               # Design tokens e estilos globais
```

### Estrutura de diretórios (backend)

```
backend/
├── main.go                  # Entry point + wiring
├── internal/
│   ├── config/              # Configuração (env, porta, JWT, Groq)
│   ├── database/            # SQLite + migrations + seeds
│   ├── models/              # Structs de dados
│   ├── repository/          # Acesso aDados (SQL)
│   ├── service/             # Regras de negócio
│   ├── handler/             # Handlers HTTP
│   ├── middleware/          # Auth JWT
│   ├── chat/                # Pipeline RAG
│   ├── knowledge/           # Ingestão e busca de normativos
│   ├── document/            # Geração DOCX/PDF
│   ├── groq/                # Cliente Groq
│   └── web/                 # Busca web fallback
├── data/                    # SQLite, documentos gerados
└── storage/
    ├── templates/           # Templates de documentos (.txt com {{placeholders}})
    └── normativos/          # PDFs de políticas internas
```

### Banco de dados

O ABIS usa SQLite (via `modernc.org/sqlite`, driver pure-Go sem CGO). As tabelas principais incluem:

- `employees` / `sessions`: autenticação e usuários.
- `documents` / `chunks` / `chunks_fts`: base de conhecimento (normativos).
- `chat_messages`: mensagens de chat persistidas (usuário + assistente).
- `workflow_tasks` / `workflow_data` / `workflow_requirements`: tarefas e coleta de dados.
- `document_templates` / `template_fields`: templates de documentos.
- `document_runs` / `document_sources`: execuções e fontes de documentos gerados.
- `history_view`: *view* unificada de histórico (chat + tarefas + documentos).

As migrations são idempotentes e rodam automaticamente na inicialização do backend.
