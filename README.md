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

## Development

Prefer working locally? You need Node.js and npm — [install with nvm](https://github.com/nvm-sh/nvm#installing-and-updating).

```sh
git clone <this-repository-url>
cd <repository-name>
npm i
npm run dev
```
