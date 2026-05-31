> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/api-reference/models/openai/retrieve-model",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# Retrieve Model

> Retrieves details for a specific model, compatible with OpenAI API specification.



## OpenAPI

````yaml /api-reference/models/openai/api/retrieve-model.json GET /v1/models/{model_id}
openapi: 3.1.0
info:
  title: MiniMax Models API
  description: MiniMax models API compatible with OpenAI API specification.
  version: 1.0.0
servers:
  - url: https://api.minimax.io
security:
  - bearerAuth: []
paths:
  /v1/models/{model_id}:
    get:
      tags:
        - Models
      summary: Retrieve Model
      description: Retrieves details for a specific model.
      operationId: retrieveModel
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
                  object:
                    type: string
                    description: Object type, always "model"
                  created:
                    type: integer
                    description: Unix timestamp when the model was created
                  owned_by:
                    type: string
                    description: Organization that owns the model
              examples:
                Default:
                  value:
                    id: MiniMax-M2.7
                    object: model
                    created: 1773799200
                    owned_by: minimax
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