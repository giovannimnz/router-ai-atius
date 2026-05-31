> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/guides/quickstart-sdk",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# Integrate via SDK

> Use the Anthropic SDK to quickly integrate with the MiniMax API and start calling the MiniMax-M2.7 model.

<Steps>
  <Step title="Install Anthropic SDK">
    <CodeGroup>
      ```bash Python theme={null}
      pip install anthropic
      ```

      ```bash Node.js theme={null}
      npm install @anthropic-ai/sdk
      ```
    </CodeGroup>
  </Step>

  <Step title="Call API">
    ```python Python theme={null}
    import anthropic

    client = anthropic.Anthropic()

    message = client.messages.create(
        model="MiniMax-M2.7",
        max_tokens=1000,
        system="You are a helpful assistant.",
        messages=[
            {
                "role": "user",
                "content": [
                    {
                        "type": "text",
                        "text": "Hi, how are you?"
                    }
                ]
            }
        ]
    )

    for block in message.content:
        if block.type == "thinking":
            print(f"Thinking:\n{block.thinking}\n")
        elif block.type == "text":
            print(f"Text:\n{block.text}\n")
    ```
  </Step>

  <Step title="Example output">
    ```json theme={null}
    {
      "thinking": "The user is just greeting me casually. I should respond in a friendly, professional manner.",
      "text": "Hi there! I'm doing well, thanks for asking. I'm ready to help you with whatever you need today—whether it's coding, answering questions, brainstorming ideas, or just chatting. What can I do for you?"
    }
    ```
  </Step>
</Steps>

## Next steps

<Columns cols={3}>
  <Card title="Text Generation" icon="book-open" href="/guides/text-generation" cta="Click here">
    Explore MiniMax's latest text models
  </Card>

  <Card title="Image to Video" icon="video" href="/guides/video-generation-i2v-refer" cta="Click here">
    Create image-to-video tasks with Hailuo 2.3
  </Card>

  <Card title="Text to Video" icon="video" href="/guides/video-generation-t2v-refer" cta="Click here">
    Create text-to-video tasks with Hailuo 2.3
  </Card>

  <Card title="Synchronous Text-to-Speech" icon="mic" href="/guides/speech-t2a-websocket" cta="Click here">
    Perform real-time speech synthesis with Speech 2.8
  </Card>

  <Card title="Async Long TTS" icon="mic" href="/guides/speech-t2a-async" cta="Click here">
    Perform asynchronous speech synthesis with Speech 2.8
  </Card>

  <Card title="Voice Clone" icon="mic" href="/guides/speech-voice-clone" cta="Click here">
    Create voice cloning tasks
  </Card>

  <Card title="Music Generation" icon="music" href="/guides/music-generation" cta="Click here">
    Compose music with Music 2.6
  </Card>
</Columns>
