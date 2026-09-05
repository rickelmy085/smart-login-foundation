# Smart Login Foundation

Construa a Etapa 1 do MVP ABIS conforme especificado: design system global com tema claro/escuro persistido, componentes reutilizáveis e acessíveis, tela de login corporativa split screen responsiva e autenticação simulada (RE 123456, senha demo123), redirecionando após sucesso para uma rota futura vazia. Não implemente Dashboard, sidebar, chat, IA, histórico, RAG ou integrações externas.

This project was built with [Lovable](https://lovable.dev).

## O que tem no projeto

A aplicação é o MVP inicial do **ABIS — Agentic Banking Intelligence System**, um assistente inteligente corporativo para o contexto bancário, e já inclui:

- **Login corporativo** (`/`): tela split screen responsiva com login simulado usando RE e senha, lembrança de sessão e redirecionamento automático para a área logada.
- **Dashboard** (`/app/`): visão geral com saudação por horário, indicadores semanais, gráfico de atividade e sugestões de perguntas.
- **ABIS** (`/app/copilot`): interface inicial do assistente com exemplos de perguntas e atalhos visuais. O envio de mensagens ainda é simulado para esta etapa.
- **Histórico** (`/app/historico`): lista de consultas anteriores agrupadas por dia, com filtro por status e busca.
- **Perfil** (`/app/perfil`): dados do colaborador e preferências, incluindo tema claro/escuro/sistema e toggles de notificações.
- **Design system**: tema claro/escuro persistido em `localStorage`, tipografia com Manrope e Sora, utilitários visuais customizados e componentes acessíveis baseados em shadcn/ui.
- **Stack principal**: React 19, TypeScript, TanStack Router, TanStack Query, Tailwind CSS v4, Vite + TanStack Start/Nitro, Radix UI, `lucide-react` e `sonner`.

## Build with Lovable

Continue developing this project in the [Lovable editor](https://lovable.dev/projects/41f60123-1769-4b98-a12f-cfc14e9147ce).

- **Ship faster**: describe what you want to build and Lovable handles the code.
- **Stay in sync**: every change made in Lovable is committed straight to this repository.
- **Full ownership**: this code is yours. Push to `main` on GitHub and your changes sync back into Lovable, ready for your next prompt.

## Backend (Go)

O backend do ABIS fica na pasta `backend/` e expõe a API de autenticação usada pelo frontend.

### Estrutura

- `backend/main.go`: entrada do servidor HTTP.
- `backend/internal/config`: carrega variáveis de ambiente e configura porta, banco e JWT.
- `backend/internal/database`: conexão com SQLite e migrations.
- `backend/internal/models`: contratos de request/response e entidades.
- `backend/internal/repository`: acesso a dados de funcionários e sessões.
- `backend/internal/service`: regras de login, validação de senha e JWT.
- `backend/internal/handler`: rotas HTTP.
- `backend/internal/middleware`: injeção de segredo JWT e proteção de rotas.

### Endpoints

- `POST /api/login`: autentica RE e senha e retorna token JWT.
- `GET /api/me`: retorna dados do funcionário autenticado.
- `POST /api/logout`: encerra a sessão.

### Como rodar

```sh
cd backend
go mod tidy
go run .
```

Por padrão o servidor sobe em `http://localhost:8081`.

### Variáveis de ambiente

- `PORT`: porta do servidor.
- `DATABASE_PATH`: caminho do SQLite.
- `JWT_SECRET`: segredo para assinar tokens.

### Usuário demo

- RE: `123456`
- Senha: `demo123`

## Development

Prefer working locally? You need Node.js and npm — [install with nvm](https://github.com/nvm-sh/nvm#installing-and-updating).

```sh
git clone <this-repository-url>
cd <repository-name>
npm i
npm run dev
```
