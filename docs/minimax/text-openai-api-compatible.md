> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/api-reference/text-openai-api",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# Compatible OpenAI API

> Call MiniMax models using the OpenAI SDK

To meet developers' needs for the OpenAI API ecosystem, our API now supports the OpenAI API format. With simple configuration, you can integrate MiniMax capabilities into the OpenAI API ecosystem.

## Quick Start

### 1. Install OpenAI SDK

<CodeGroup>
  ```bash Python theme={null}
  pip install openai
  ```

  ```bash Node.js theme={null}
  npm install openai
  ```
</CodeGroup>

### 2. Configure Environment Variables

```bash theme={null}
export OPENAI_BASE_URL=https://api.minimax.io/v1
export OPENAI_API_KEY=${YOUR_API_KEY}
```

### 3. Call API

```python Python theme={null}
from openai import OpenAI

client = OpenAI()

response = client.chat.completions.create(
    model="MiniMax-M2.7",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Hi, how are you?"},
    ],
    # Set reasoning_split=True to separate thinking content into reasoning_details field
    extra_body={"reasoning_split": True},
)

print(f"Thinking:\n{response.choices[0].message.reasoning_details[0]['text']}\n")
print(f"Text:\n{response.choices[0].message.content}\n")
```

### 4. Important Note

In multi-turn function call conversations, the complete model response (i.e., the assistant message) must be append to the conversation history to maintain the continuity of the reasoning chain.

* Append the full `response_message` object (including the `tool_calls` field) to the message history
  * For native OpenAI API with `MiniMax-M2.7` `MiniMax-M2.7-highspeed` `MiniMax-M2.5` `MiniMax-M2.5-highspeed` `MiniMax-M2.1` `MiniMax-M2.1-highspeed` `MiniMax-M2` models, the `content` field will contain `<think>` tag content, which must be preserved completely
  * In the Interleaved Thinking compatible format, by enabling the additional parameter (`reasoning_split=True`), the model's thinking content is provided separately via the `reasoning_details` field, which must also be preserved completely

## Supported Models

When using the OpenAI SDK, the following MiniMax models are supported:

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

<Note>
  For more model information, please refer to the standard MiniMax API
  documentation.
</Note>

## Examples

### Streaming Response

```python Python theme={null}
from openai import OpenAI

client = OpenAI()

print("Starting stream response...\n")
print("=" * 60)
print("Thinking Process:")
print("=" * 60)

stream = client.chat.completions.create(
    model="MiniMax-M2.7",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Hi, how are you?"},
    ],
    # Set reasoning_split=True to separate thinking content into reasoning_details field
    extra_body={"reasoning_split": True},
    stream=True,
)

reasoning_buffer = ""
text_buffer = ""

for chunk in stream:
    if (
        hasattr(chunk.choices[0].delta, "reasoning_details")
        and chunk.choices[0].delta.reasoning_details
    ):
        for detail in chunk.choices[0].delta.reasoning_details:
            if "text" in detail:
                reasoning_text = detail["text"]
                new_reasoning = reasoning_text[len(reasoning_buffer) :]
                if new_reasoning:
                    print(new_reasoning, end="", flush=True)
                    reasoning_buffer = reasoning_text

    if chunk.choices[0].delta.content:
        content_text = chunk.choices[0].delta.content
        new_text = content_text[len(text_buffer) :] if text_buffer else content_text
        if new_text:
            print(new_text, end="", flush=True)
            text_buffer = content_text

print("\n" + "=" * 60)
print("Response Content:")
print("=" * 60)
print(f"{text_buffer}\n")
```

### Tool Use & Interleaved Thinking

Learn how to use M2.1 Tool Use and Interleaved Thinking capabilities with OpenAI SDK, please refer to the following documentation.

<Columns cols={1}>
  <Card title="M2.1 Tool Use & Interleaved Thinking" icon="book-open" href="/guides/text-m2-function-call#openai-sdk" arrow="true" cta="Click here">
    Learn how to leverage MiniMax-M2.1 tool calling and interleaved thinking capabilities to enhance performance in complex tasks.
  </Card>
</Columns>

## Important Notes

<Warning>
  1. The `temperature` parameter range is (0.0, 1.0], recommended value: 1.0, values outside this range will return an error

  2. Some OpenAI parameters (such as `presence_penalty`, `frequency_penalty`, `logit_bias`, etc.) will be ignored

  3. Image and audio type inputs are not currently supported

  4. The `n` parameter only supports value 1

  5. The deprecated `function_call` is not supported, please use the `tools` parameter
</Warning>
