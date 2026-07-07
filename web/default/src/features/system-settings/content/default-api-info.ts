export const DEFAULT_API_INFO = [
  {
    url: 'https://router.atius.com.br/v1/chat/completions',
    route: 'OpenAI Compatible',
    description:
      'Chat Completions API - OpenAI-compatible endpoint for chat-style text generation',
    color: 'blue',
  },
  {
    url: 'https://router.atius.com.br/v1/responses',
    route: 'Responses',
    description:
      'Responses API - OpenAI-compatible endpoint for stateful and tool-ready responses',
    color: 'indigo',
  },
  {
    url: 'https://router.atius.com.br/v1/messages',
    route: 'Anthropic Compatible',
    description:
      'Messages API - Anthropic-compatible endpoint for Claude-format requests',
    color: 'orange',
  },
  {
    url: 'https://router.atius.com.br/v1/completions',
    route: 'Completions',
    description: 'Completions API - Legacy prompt-completion endpoint',
    color: 'green',
  },
  {
    url: 'https://router.atius.com.br/v1/embeddings',
    route: 'Embeddings',
    description: 'Embeddings API - Text embedding generation endpoint',
    color: 'purple',
  },
  {
    url: 'https://router.atius.com.br/v1/audio/speech',
    route: 'Text-to-Speech',
    description: 'TTS API - Text-to-Speech synthesis endpoint',
    color: 'pink',
  },
  {
    url: 'https://router.atius.com.br/v1/models',
    route: 'Models',
    description: 'Models API - List all available models',
    color: 'cyan',
  },
] as const
