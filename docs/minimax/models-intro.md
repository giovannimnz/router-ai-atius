> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/guides/models-intro",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# Models

> Overview of MiniMax AI models and their capabilities

<CardGroup cols={3}>
  <Card title="MiniMax M2.7" href="https://www.minimax.io/news/minimax-m27-en" img="https://file.cdn.minimax.io/public/cf016263-4211-45b1-8295-23e39812c201.png">
    Beginning the journey of recursive self-improvement.
  </Card>

  <Card title="Music2.6" href="https://platform.minimax.io/docs/api-reference/music-generation" img="https://file.cdn.minimax.io/public/4dd6ffb0-256f-4062-8d9b-06165522b9bf.png">
    Cover Reborn. Bass Redefined.
  </Card>

  <Card title="MiniMax Hailuo 2.3" horizontal="false" href="https://www.minimax.io/news/minimax-hailuo-23" img="https://filecdn.minimax.chat/public/71df4ee8-f064-441e-85aa-bd6cab09111b.png">
    Breathtaking Motion, Lifelike Emotion
  </Card>

  <Card title="MiniMax Speech 2.8" href="https://www.minimax.io/news/minimax-speech-28" img="https://file.cdn.minimax.io/public/21410750-5213-4ec5-9d65-513178f8a9b6.png">
    Natural Sound Tags, lifelike voice, pristine audio quality
  </Card>

  <Card title="MiniMax M2-her" href="https://www.minimax.io/news/a-deep-dive-into-the-minimax-m2-her-2" img="https://file.cdn.minimax.io/public/283600ad-d685-4a85-a68d-4cbce24cc6a5.png">
    Multi-Character Roleplay, Immersive Long-horizon Interaction
  </Card>
</CardGroup>

## Models Overview

### Text

| **Models**                                                                                  | **Description**                                                                     | **Features**                                                                                           |
| :------------------------------------------------------------------------------------------ | :---------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------- |
| [MiniMax-M2.7](https://platform.minimax.io/docs/api-reference/text-anthropic-api)           | Beginning the journey of recursive self-improvement                                 | • Top real-world engineering <br />• Professional office delivery <br /> •  Character-rich interaction |
| [MiniMax-M2.7-highspeed](https://platform.minimax.io/docs/api-reference/text-anthropic-api) | Same performance as M2.7<br /> • Significantly faster inference                     | • Polyglot code mastery<br />• Precision code refactoring <br /> • Low latency                         |
| [MiniMax-M2.5](https://platform.minimax.io/docs/api-reference/text-anthropic-api)           | • Optimized for code generation and refactoring                                     | • Peak Performance. Ultimate Value. Master the Complex.                                                |
| [MiniMax-M2.5-highspeed](https://platform.minimax.io/docs/api-reference/text-anthropic-api) | • Same performance as M2.5<br /> • Significantly faster inference                   | • Polyglot code mastery<br />• Precision code refactoring <br /> • Low latency                         |
| [M2-her](https://platform.minimax.io/docs/api-reference/text-chat)                          | • Text dialogue model<br />• Designed for role-playing and multi-turn conversations | • Character customization<br />• Emotional expression<br />• Multi-turn dialogue                       |

<Accordion title="Legacy Models">
  | **Models**                                                                                  | **Description**                                                                                                | **Features**                                                                                        |
  | :------------------------------------------------------------------------------------------ | :------------------------------------------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------- |
  | [MiniMax-M2.1](https://platform.minimax.io/docs/api-reference/text-anthropic-api)           | • 230B total parameters with 10B activated per inference<br /> • Optimized for code generation and refactoring | • Polyglot code mastery<br />• Precision code refactoring <br /> • Enhanced reasoning               |
  | [MiniMax-M2.1-highspeed](https://platform.minimax.io/docs/api-reference/text-anthropic-api) | • Same performance as M2.1<br /> • Significantly faster inference                                              | • Polyglot code mastery<br />• Precision code refactoring <br /> • Low latency                      |
  | [MiniMax-M2](https://platform.minimax.io/docs/api-reference/text-anthropic-api)             | • Context Length: 200k tokens<br />• Maximum Output: 128k tokens (including CoT)                               | • Agentic capabilities<br />• Function calling<br />• Advanced reasoning<br />• Real-time streaming |
</Accordion>

### Audio

| **Models**                                                                         | **Description**                                                        | **Features**                                                                                           |
| :--------------------------------------------------------------------------------- | :--------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------- |
| [speech-2.8-hd](https://platform.minimax.io/docs/api-reference/speech-t2a-http)    | • Ultra-realistic quality featuring sound tags                         | • 40 languages supported<br />• 7 emotions supported<br />• specified languages and dialects supported |
| [speech-2.8-turbo](https://platform.minimax.io/docs/api-reference/speech-t2a-http) | • Seamless speed meets natural flow                                    | • 40 languages supported<br />• 7 emotions supported<br />• specified languages and dialects supported |
| [speech-2.6-hd](https://platform.minimax.io/docs/api-reference/speech-t2a-http)    | • Ultimate Similarity<br />• Ultra-High Quality                        | • 40 languages supported<br />• 7 emotions supported<br />• specified languages and dialects supported |
| [speech-2.6-turbo](https://platform.minimax.io/docs/api-reference/speech-t2a-http) | • Ultimate Value<br />• Low latency                                    | • 40 languages supported<br />• 7 emotions supported<br />• specified languages and dialects supported |
| [speech-02-hd](https://platform.minimax.io/docs/api-reference/speech-t2a-http)     | • Stronger replication similarity<br />• High quality voice generation | • 24 languages supported<br />• 7 emotions supported<br />• specified languages and dialects supported |
| [speech-02-turbo](https://platform.minimax.io/docs/api-reference/speech-t2a-http)  | • Superior rhythm and stability<br />• Low latency                     | • 24 languages supported<br />• 7 emotions supported<br />• specified languages and dialects supported |

### Video

| **Models**                                                                                    | **Description**                                                                                   | **Res.& Dur.**                                     | **FPS**         |
| :-------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------ | :------------------------------------------------- | :-------------- |
| [MiniMax Hailuo 2.3](https://platform.minimax.io/docs/api-reference/video-generation-t2v)     | • Text to Video & Image to Video<br />• SOTA instruction following<br />• Extreme physics mastery | • 1080p 6s<br />• 768p 6s, 10s<br />               | 24 fps          |
| [MiniMax Hailuo 2.3Fast](https://platform.minimax.io/docs/api-reference/video-generation-i2v) | •  Image to Video<br />• Extreme physics mastery<br />• Value and Efficiency                      | • 1080p 6s<br />• 768p 6s, 10s<br />               | 24 fps          |
| [MiniMax Hailuo 02](https://platform.minimax.io/docs/api-reference/video-generation-t2v)      | • Text to Video & Image to Video<br />• SOTA instruction following<br />• Extreme physics mastery | • 1080p 6s<br />• 768p 6s, 10s<br />• 512p 6s, 10s | 24 fps          |

### Music

| **Models**                                                                     | **Description**                                                                       | **Features**                                                                                                                   |
| :----------------------------------------------------------------------------- | :------------------------------------------------------------------------------------ | :----------------------------------------------------------------------------------------------------------------------------- |
| [Music-2.6](https://platform.minimax.io/docs/api-reference/music-generation)   | • Cover Reborn. Bass Redefined.                                                       | • Cover Reborn. Bass Redefined.                                                                                                |
| [Music-Cover](https://platform.minimax.io/docs/api-reference/music-generation) | • Generate cover versions from reference audio                                        | • One-step cover generation<br />• Two-step cover with lyrics modification<br />• Style transfer<br />• Auto lyrics extraction |
| [Music-2.0](https://platform.minimax.io/docs/api-reference/music-generation)   | • Text to Music<br />• Enhanced musicality <br />• Natural vocals and smooth melodies | • Human-like performance<br />• Riche emotional expression<br />• Enhanced tone control<br />                                  |

## Recommended Reading

<CardGroup cols={2}>
  <Card title="Quick start" icon="book-open" href="/guides/quickstart" arrow="true" cta="Click here">
    Refer to the Quick Start Guide to explore and experience the model’s capabilities
  </Card>

  <Card title="Compatible Anthropic API (Recommended)" icon="book-open" href="/api-reference/text-anthropic-api" arrow="true" cta="Click here">
    Use Anthropic SDK with MiniMax models
  </Card>
</CardGroup>
