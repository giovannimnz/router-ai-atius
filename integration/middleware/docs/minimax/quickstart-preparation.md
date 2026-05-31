> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/guides/quickstart-preparation",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# Prerequisites

> Before using the MiniMax API, you need to complete account registration and obtain an API Key.

<Steps>
  <Step title="Register or Login">
    Access MiniMax API Platform, [register](https://platform.minimax.io/login)  or [login](https://platform.minimax.io/login) .
  </Step>

  <Step title="Create an API Key">
    * **Pay-as-you-go**：Visit [API Keys > Create new secret key](https://platform.minimax.io/user-center/basic-information/interface-key) to get your **API Key**
      <Note>Pay-as-you-go supports all modality models, including Text, Video, Speech, and Image.</Note>
    * **Token Plan**：Visit [API Keys > Create Token Plan Key](https://platform.minimax.io/user-center/payment/token-plan) to get your **API Key**
      <Note>Token Plan supports MiniMax models of all modalities. See [Token Plan Overview](https://platform.minimax.io/docs/token-plan/intro) for details.</Note>

    After generating an API key, we recommend you export it as an environment variable in terminal or save it to a `.env` file.

    ```bash theme={null}
    # Compatible Anthropic API (Recommended)
    export ANTHROPIC_BASE_URL=https://api.minimax.io/anthropic
    export ANTHROPIC_API_KEY=${YOUR_API_KEY}
    ```
  </Step>

  <Step title="Recharge Account">
    Access [Billing/Balance](https://platform.minimax.io/user-center/payment/balance) page to top up if needed.
  </Step>
</Steps>
