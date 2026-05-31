> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/guides/pricing-paygo",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# Pay as You Go

> MiniMax Pay as You Go Pricing

## Text

[Recharge Now](https://platform.minimax.io/user-center/payment/balance)

| Model                      | Input            | Output           | Prompt caching Read | Prompt caching Write |
| :------------------------- | :--------------- | :--------------- | :------------------ | :------------------- |
| **MiniMax-M2.7**           | \$0.3 / M tokens | \$1.2 / M tokens | \$0.06 / M tokens   | \$0.375 / M tokens   |
| **MiniMax-M2.7-highspeed** | \$0.6 / M tokens | \$2.4 / M tokens | \$0.06 / M tokens   | \$0.375 / M tokens   |
| **MiniMax-M2.5**           | \$0.3 / M tokens | \$1.2 / M tokens | \$0.03 / M tokens   | \$0.375 / M tokens   |
| **MiniMax-M2.5-highspeed** | \$0.6 / M tokens | \$2.4 / M tokens | \$0.03 / M tokens   | \$0.375 / M tokens   |
| **M2-her**                 | \$0.3 / M tokens | \$1.2 / M tokens | ——                  | ——                   |

<Accordion title="Legacy Models">
  | Model                      | Input            | Output           | Prompt caching Read | Prompt caching Write |
  | :------------------------- | :--------------- | :--------------- | :------------------ | :------------------- |
  | **MiniMax-M2.1**           | \$0.3 / M tokens | \$1.2 / M tokens | \$0.03 / M tokens   | \$0.375 / M tokens   |
  | **MiniMax-M2.1-highspeed** | \$0.6 / M tokens | \$2.4 / M tokens | \$0.03 / M tokens   | \$0.375 / M tokens   |
  | **MiniMax-M2**             | \$0.3 / M tokens | \$1.2 / M tokens | \$0.03 / M tokens   | \$0.375 / M tokens   |
</Accordion>

<Info>
  Note:

  1. The billing item is token count; the token-to-character ratio varies slightly depending on the usage scenario, subject to actual consumption
  2. Billing tokens include both input and output.
  3. Token to character ratio (estimate): approximately 1600 Chinese characters consume 1000 tokens
</Info>

## Audio

[Recharge Now](https://platform.minimax.io/user-center/payment/balance)

| API                     | Model                                                               | Price              |
| :---------------------- | :------------------------------------------------------------------ | :----------------- |
| **T2A**                 | • speech-2.8-turbo <br />• speech-2.6-turbo <br />• speech-02-turbo | \$60/M characters  |
| **T2A**                 | • speech-2.8-hd <br />• speech-2.6-hd <br />• speech-02-hd          | \$100/M characters |
| **Rapid Voice Cloning** | All Models                                                          | \$1.5 per voice    |
| **Voice Design**        | All Models                                                          | \$3 per voice      |

## Video

[Recharge Now](https://platform.minimax.io/user-center/payment/balance)

| Model                                     | Price                      |
| :---------------------------------------- | :------------------------- |
| MiniMax-Hailuo-2.3-Fast                   | \$0.19 per 768P, 6s video  |
| MiniMax-Hailuo-2.3-Fast                   | \$0.32 per 768P, 10s video |
| MiniMax-Hailuo-2.3-Fast                   | \$0.33 per 1080P, 6s video |
| MiniMax-Hailuo-2.3<br />MiniMax-Hailuo-02 | \$0.28 per 768P, 6s video  |
| MiniMax-Hailuo-2.3<br />MiniMax-Hailuo-02 | \$0.56 per 768P, 10s video |
| MiniMax-Hailuo-2.3<br />MiniMax-Hailuo-02 | \$0.49 per 1080P, 6s video |
| MiniMax-Hailuo-02                         | \$0.10 per 512P, 6s video  |
| MiniMax-Hailuo-02                         | \$0.15 per 512P, 10s video |

## Music

[Recharge Now](https://platform.minimax.io/user-center/payment/balance)

| Model                     | Price                                       |
| :------------------------ | :------------------------------------------ |
| Music-2.6                 | \$0.15/up-to-5 minutes music (Limited Free) |
| Music-2.5+<br />Music-2.5 | \$0.15/up-to-5 minutes music                |
| Music-2.0                 | \$0.03/up-to-5 minutes music                |
| Lyrics Generation         | \$0.01/per song (Limited Free)              |

## Image

[Recharge Now](https://platform.minimax.io/user-center/payment/balance)

| Model    | Price              |
| :------- | :----------------- |
| image-01 | \$0.0035 per image |

## MCP

[Recharge Now](https://platform.minimax.io/user-center/payment/balance)

| Model       | Input Price      |
| :---------- | :--------------- |
| **API-vlm** | \$0.06 / request |
