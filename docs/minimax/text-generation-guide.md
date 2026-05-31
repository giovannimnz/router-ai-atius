> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/guides/text-generation",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# Text Generation

> MiniMax text models, supporting multilingual programming, Agent workflows and complex task scenarios.

<Note>
  Subscribe to [Token Plan](https://platform.minimax.io/subscribe/token-plan) to use MiniMax models of all modalities at ultra-low prices!
</Note>

## Model Overview

MiniMax offers multiple text models to meet different scenario requirements. **MiniMax-M2.7** achieves or sets new SOTA benchmarks in programming, tool calling and search, office productivity and other scenarios, while **MiniMax-M2** is built for efficient coding and Agent workflows.

### Supported Models

| Model Name             | Context Window | Description                                                                                                                                   |
| :--------------------- | :------------- | :-------------------------------------------------------------------------------------------------------------------------------------------- |
| MiniMax-M2.7           | 204,800        | **Beginning the journey of recursive self-improvement** (output speed approximately 60 tps)                                                   |
| MiniMax-M2.7-highspeed | 204,800        | **M2.7 Highspeed: Same performance, faster and more agile (output speed approximately 100 tps)**                                              |
| MiniMax-M2.5           | 204,800        | **Peak Performance. Ultimate Value. Master the Complex (output speed approximately 60 tps)**                                                  |
| MiniMax-M2.5-highspeed | 204,800        | **M2.5 highspeed: Same performance, faster and more agile (output speed approximately 100 tps)**                                              |
| MiniMax-M2.1           | 204,800        | **Powerful Multi-Language Programming Capabilities with Comprehensively Enhanced Programming Experience (output speed approximately 60 tps)** |
| MiniMax-M2.1-highspeed | 204,800        | **Faster and More Agile (output speed approximately 100 tps)**                                                                                |
| MiniMax-M2             | 204,800        | **Agentic capabilities, Advanced reasoning**                                                                                                  |

<Note>
  For details on how tps (Tokens Per Second) is calculated, please refer to [FAQ > About APIs](/faq/about-apis#q-how-is-tps-tokens-per-second-calculated-for-text-models).
</Note>

### **MiniMax M2.7** Key Highlights

<AccordionGroup>
  <Accordion title="Top real-world engineering">
    M2.7 delivers outstanding performance in real-world software engineering, including end-to-end full project delivery, log analysis and bug troubleshooting, code security, machine learning, and more. On the SWE-Pro benchmark, M2.7 scored 56.22%, nearly approaching Opus's best level. This capability also extends to end-to-end full project delivery scenarios (VIBE-Pro 55.6%) and deep understanding of complex engineering systems on Terminal Bench 2 (57.0%).
  </Accordion>

  <Accordion title="Professional office delivery">
    In the professional office domain, we have enhanced the model's expertise and task delivery capabilities across various fields. Its ELO score on GDPval-AA is 1495, the highest among open-source models. M2.7 shows significantly improved ability for complex editing in the Office suite — Excel, PPT, and Word — and can better handle multi-round revisions and high-fidelity editing. M2.7 is capable of interacting with complex environments: across 40 complex skills (each exceeding 2,000 tokens), it still maintains a 97% skill adherence rate.
  </Accordion>

  <Accordion title="Character-rich interaction">
    M2.7 possesses excellent character consistency and emotional intelligence, opening up more room for product innovation.
  </Accordion>
</AccordionGroup>

<Note>
  For more model details, please refer to [MiniMax M2.7](https://www.minimax.io/news/minimax-m27-en)
</Note>

***

## URL Configuration

Before calling MiniMax models, prepare the following:

| Field                                          | Value                                                                                |
| :--------------------------------------------- | :----------------------------------------------------------------------------------- |
| `base_url` (Anthropic-compatible, recommended) | `https://api.minimax.io/anthropic`                                                   |
| `base_url` (OpenAI-compatible)                 | `https://api.minimax.io/v1`                                                          |
| `api_key`                                      | [Get Token Plan API Key](https://platform.minimax.io/user-center/payment/token-plan) |
| `model`                                        | See [Supported Models](#supported-models) above                                      |

***

## Calling Example

MiniMax accepts both Anthropic-style and OpenAI-style request formats. The two examples below are equivalent non-streaming calls; flip `stream` to `true` to switch to streaming responses.

### Anthropic-Compatible (Recommended)

Supports thinking blocks, interleaved thinking, and other advanced features — this is the default path.

<CodeGroup>
  ```bash curl theme={null}
  curl https://api.minimax.io/anthropic/v1/messages \
    -H "Authorization: Bearer <MINIMAX_API_KEY>" \
    -H "Content-Type: application/json" \
    -d '{
      "model": "MiniMax-M2.7",
      "max_tokens": 1000,
      "messages": [
        {"role": "user", "content": "Hi, how are you?"}
      ]
    }'
  ```

  ```python Python theme={null}
  # Please install the Anthropic SDK first: `pip install anthropic`
  import anthropic

  client = anthropic.Anthropic(
      base_url="https://api.minimax.io/anthropic",
      api_key="<MINIMAX_API_KEY>",
  )

  message = client.messages.create(
      model="MiniMax-M2.7",
      max_tokens=1000,
      messages=[
          {"role": "user", "content": "Hi, how are you?"}
      ],
  )

  for block in message.content:
      if block.type == "thinking":
          print(f"Thinking:\n{block.thinking}\n")
      elif block.type == "text":
          print(f"Text:\n{block.text}\n")
  ```

  ```javascript Node.js theme={null}
  // Please install the Anthropic SDK first: `npm install @anthropic-ai/sdk`
  import Anthropic from "@anthropic-ai/sdk";

  const client = new Anthropic({
    baseURL: "https://api.minimax.io/anthropic",
    apiKey: "<MINIMAX_API_KEY>",
  });

  const message = await client.messages.create({
    model: "MiniMax-M2.7",
    max_tokens: 1000,
    messages: [
      { role: "user", content: "Hi, how are you?" },
    ],
  });

  for (const block of message.content) {
    if (block.type === "thinking") {
      console.log(`Thinking:\n${block.thinking}\n`);
    } else if (block.type === "text") {
      console.log(`Text:\n${block.text}\n`);
    }
  }
  ```
</CodeGroup>

### OpenAI-Compatible

Already wired up to the OpenAI SDK? Swap `base_url` and `model` for the values below and you can keep using your existing client without migrating to a new SDK.

<CodeGroup>
  ```bash curl theme={null}
  curl https://api.minimax.io/v1/chat/completions \
    -H "Authorization: Bearer <MINIMAX_API_KEY>" \
    -H "Content-Type: application/json" \
    -d '{
      "model": "MiniMax-M2.7",
      "messages": [
        {"role": "user", "content": "Hi, how are you?"}
      ]
    }'
  ```

  ```python Python theme={null}
  # Please install the OpenAI SDK first: `pip install openai`
  from openai import OpenAI

  client = OpenAI(
      base_url="https://api.minimax.io/v1",
      api_key="<MINIMAX_API_KEY>",
  )

  response = client.chat.completions.create(
      model="MiniMax-M2.7",
      messages=[
          {"role": "user", "content": "Hi, how are you?"},
      ],
  )

  print(response.choices[0].message.content)
  ```

  ```javascript Node.js theme={null}
  // Please install the OpenAI SDK first: `npm install openai`
  import OpenAI from "openai";

  const client = new OpenAI({
    baseURL: "https://api.minimax.io/v1",
    apiKey: "<MINIMAX_API_KEY>",
  });

  const response = await client.chat.completions.create({
    model: "MiniMax-M2.7",
    messages: [
      { role: "user", content: "Hi, how are you?" },
    ],
  });

  console.log(response.choices[0].message.content);
  ```
</CodeGroup>

***

## API Reference

<Columns cols={2}>
  <Card title="Anthropic API Compatible (Recommended)" icon="book-open" href="/api-reference/text-anthropic-api" cta="View Docs">
    Call MiniMax models via Anthropic SDK, supporting streaming output and Interleaved Thinking
  </Card>

  <Card title="OpenAI API Compatible" icon="book-open" href="/api-reference/text-openai-api" cta="View Docs">
    Call MiniMax models via OpenAI SDK
  </Card>

  <Card title="Text Generation" icon="file-text" href="/api-reference/text-post" cta="View Docs">
    Call text generation API directly via HTTP requests
  </Card>

  <Card title="Using M2.7 in AI Coding Tools" icon="code" href="/guides/text-ai-coding-tools" cta="View Docs">
    Use M2.7 in Claude Code, Cursor, Cline and other tools
  </Card>
</Columns>

***

## Contact Us

If you encounter any issues while using MiniMax models:

* Contact our technical support team through official channels such as email [Model@minimax.io](mailto:Model@minimax.io)
* Submit an Issue on our [Github](https://github.com/MiniMax-AI/MiniMax-M2.7/issues) repository

## Related Links

* [Anthropic SDK Documentation](https://docs.anthropic.com/en/api/client-sdks)
* [OpenAI SDK Documentation](https://platform.openai.com/docs/libraries)
* [MiniMax M2.7](https://www.minimax.io/news/minimax-m27-en)
