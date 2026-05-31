> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/api-reference/models/anthropic/retrieve-model",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# Retrieve Model

> Retrieves details for a specific model, compatible with Anthropic API specification.



## OpenAPI

````yaml /api-reference/models/anthropic/api/retrieve-model.json GET /anthropic/v1/models/{model_id}
openapi: 3.1.0
info:
  title: MiniMax Models API
  description: MiniMax models API compatible with Anthropic API specification.
  version: 1.0.0
servers:
  - url: https://api.minimax.io
security:
  - bearerAuth: []
paths:
  /anthropic/v1/models/{model_id}:
    get:
      tags:
        - Models
      summary: Retrieve Model
      description: Retrieves details for a specific model.
      operationId: anthropicRetrieveModel
      parameters:
        - name: model_id
          in: path
          required: true
          description: Model identifier
          schema:
            type: string
      responses:
        '200':
          description: Details of the requested model.
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                    description: Model identifier
                  created_at:
                    type: string
                    description: Model creation time (ISO 8601 format)
                  display_name:
                    type: string
                    description: Model display name
                  type:
                    type: string
                    description: Model type, always "model"
              examples:
                Default:
                  value:
                    id: MiniMax-M2.7
                    created_at: '2026-03-18T02:00:00Z'
                    display_name: MiniMax-M2.7
                    type: model
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: >-
        Bearer token authentication. API key can be obtained from Account
        Management > API Keys

````