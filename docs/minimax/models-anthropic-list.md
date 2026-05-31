> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/api-reference/models/anthropic/list-models",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# List Models

> Returns a list of all available models compatible with Anthropic API specification.



## OpenAPI

````yaml /api-reference/models/anthropic/api/list-models.json GET /anthropic/v1/models
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
  /anthropic/v1/models:
    get:
      tags:
        - Models
      summary: List Models
      description: >-
        Returns a list of all available models. This endpoint is compatible with
        Anthropic API specification.
      operationId: anthropicListModels
      parameters:
        - name: limit
          in: query
          required: false
          description: Number of items to return per page
          schema:
            type: integer
        - name: after_id
          in: query
          required: false
          description: Pagination cursor, returns models after this ID
          schema:
            type: string
        - name: before_id
          in: query
          required: false
          description: Pagination cursor, returns models before this ID
          schema:
            type: string
      responses:
        '200':
          description: A list of available models.
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    description: Array of model objects
                    items:
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
                  first_id:
                    type: string
                    description: First model ID in the returned data
                  has_more:
                    type: boolean
                    description: Whether there is more data
                  last_id:
                    type: string
                    description: Last model ID in the returned data
              examples:
                Default:
                  value:
                    data:
                      - id: MiniMax-M2.7
                        created_at: '2026-03-18T02:00:00Z'
                        display_name: MiniMax-M2.7
                        type: model
                      - id: MiniMax-M2.5
                        created_at: '2026-02-13T02:00:00Z'
                        display_name: MiniMax-M2.5
                        type: model
                    first_id: MiniMax-M2.7
                    has_more: false
                    last_id: MiniMax-M2.5
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