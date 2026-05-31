> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/api-reference/text-chat-anthropic",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# Text Chat (Compatible Anthropic API)

> Use the Anthropic API compatible format to call MiniMax models, supporting role-playing, multi-turn conversations and other dialogue scenarios. Supports rich role settings (system, user_system, group, etc.) and example dialogue learning.



## OpenAPI

````yaml /api-reference/text/api/openapi-chat-anthropic.json POST /anthropic/v1/messages
openapi: 3.1.0
info:
  title: MiniMax Text API Anthropic
  description: >-
    MiniMax text generation API with support for chat completion and streaming
    output
  license:
    name: MIT
  version: 1.0.0
servers:
  - url: https://api.minimax.io
security:
  - bearerAuth: []
paths:
  /anthropic/v1/messages:
    post:
      tags:
        - Text Generation
      summary: Text Generation Anthropic
      operationId: chatCompletionAnthropic
      parameters:
        - name: Content-Type
          in: header
          required: true
          description: >-
            Media type of the request body, should be set to `application/json`
            to ensure JSON format
          schema:
            type: string
            enum:
              - application/json
            default: application/json
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateMessageReq'
            examples:
              Request:
                value:
                  model: MiniMax-M2.7
                  messages:
                    - role: user
                      content: Hello
              Stream:
                value:
                  model: MiniMax-M2.7
                  stream: true
                  messages:
                    - role: user
                      content: Hello
        required: true
      responses:
        '200':
          description: ''
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CreateMessageResp'
              examples:
                Request:
                  value:
                    id: 06379fa1dfdd9047604b8abc088ea75c
                    type: message
                    role: assistant
                    model: MiniMax-M2.7
                    content:
                      - thinking: >
                          The user says "Hello". This is a simple greeting. We
                          should respond politely, greet them back, maybe ask
                          how we can help.
                        signature: >-
                          1c3a0ae890922669e9815a201f9b645abdaafe8d8b5a65a5e48f90830c6e0750
                        type: thinking
                      - text: Hello! How can I help you today?
                        type: text
                    usage:
                      input_tokens: 39
                      output_tokens: 40
                      cache_creation_input_tokens: 0
                      cache_read_input_tokens: 0
                    stop_reason: end_turn
                    base_resp:
                      status_code: 0
                      status_msg: success
                Stream:
                  value:
                    - type: message_start
                      message:
                        id: 06379fc52f6880f96d6c2102eea03cc9
                        type: message
                        role: assistant
                        content: []
                        model: MiniMax-M2.7
                        stop_reason: null
                        stop_sequence: null
                        usage:
                          input_tokens: 3
                          output_tokens: 0
                          cache_creation_input_tokens: 0
                          cache_read_input_tokens: 0
                        service_tier: standard
                    - type: ping
                    - type: content_block_start
                      index: 0
                      content_block:
                        type: thinking
                        thinking: ''
                    - type: content_block_delta
                      index: 0
                      delta:
                        type: thinking_delta
                        thinking: >+
                          The user says "Hello". The conversation starts. The
                          user presumably wants a greeting and interaction. The
                          system instructions are to be helpful and the
                          developer says anything is allowed. So I should
                          respond with a friendly greeting, maybe ask how I can
                          help. There's no conflict with policies. Just respond.


                    - type: content_block_delta
                      index: 0
                      delta:
                        type: signature_delta
                        signature: >-
                          48f709926bbaa022cc567c6d64805fd0d5306458d2b60e2f815ffcbe13b67593
                    - type: content_block_stop
                      index: 0
                    - type: content_block_start
                      index: 1
                      content_block:
                        type: text
                        text: ''
                    - type: content_block_delta
                      index: 1
                      delta:
                        type: text_delta
                        text: |-


                          Hello! How can I help you today?
                    - type: content_block_stop
                      index: 1
                    - type: message_delta
                      delta:
                        stop_reason: end_turn
                      usage:
                        input_tokens: 7
                        output_tokens: 71
                        cache_creation_input_tokens: 0
                        cache_read_input_tokens: 32
                    - type: message_stop
            text/event-stream:
              schema:
                $ref: '#/components/schemas/StreamEvent'
              examples:
                Stream:
                  value:
                    - type: message_start
                      message:
                        id: 06369732447536c3c6b051a4f612aae3
                        type: message
                        role: assistant
                        content: []
                        model: MiniMax-M2.7
                        stop_reason: null
                        stop_sequence: null
                        usage:
                          input_tokens: 0
                          output_tokens: 0
                        service_tier: standard
                    - type: ping
                    - type: content_block_start
                      index: 0
                      content_block:
                        type: text
                        text: ''
                    - type: content_block_delta
                      index: 0
                      delta:
                        type: text_delta
                        text: Hello
                    - type: content_block_delta
                      index: 0
                      delta:
                        type: text_delta
                        text: >-
                          ! I'm MiniMax-M2.7, nice to meet you! Is there
                          anything
                    - type: content_block_delta
                      index: 0
                      delta:
                        type: text_delta
                        text: ' I can help you with?'
                    - type: content_block_stop
                      index: 0
                    - type: message_delta
                      delta:
                        stop_reason: end_turn
                      usage:
                        input_tokens: 176
                        output_tokens: 17
                    - type: message_stop
components:
  schemas:
    CreateMessageReq:
      type: object
      required:
        - model
        - messages
      properties:
        model:
          type: string
          description: Model ID
          enum:
            - MiniMax-M2.7
            - MiniMax-M2.7-highspeed
            - MiniMax-M2.5
            - MiniMax-M2.1
        system:
          description: Set the role and behavior of the model
          oneOf:
            - type: string
              description: Plain text system prompt
            - type: array
              description: System prompt in content block array format
              items:
                type: object
                properties:
                  type:
                    type: string
                    enum:
                      - text
                    description: Content block type
                  text:
                    type: string
                    description: Text content
                required:
                  - type
                  - text
        messages:
          type: array
          description: A list of messages containing the conversation history
          items:
            $ref: '#/components/schemas/Message'
        stream:
          type: boolean
          description: >-
            Whether to use streaming output, defaults to `false`. When set to
            `true`, the response will be returned in chunks
          default: false
        max_tokens:
          type: integer
          format: int64
          description: >-
            Specifies the upper limit for generated content length (in tokens),
            maximum is 2048. Content exceeding the limit will be truncated. If
            generation stops due to `length`, try increasing this value
          minimum: 1
        temperature:
          type: number
          format: double
          description: >-
            Temperature coefficient, affects output randomness, value range (0,
            1], default value for MiniMax model is 1.0. Higher values produce
            more random output; lower values produce more deterministic output
          minimum: 0
          exclusiveMinimum: 0
          maximum: 1
          default: 1
        top_p:
          type: number
          format: double
          description: >-
            Sampling strategy, affects output randomness, value range (0, 1],
            default value for MiniMax model is 0.95
          minimum: 0
          exclusiveMinimum: 0
          maximum: 1
          default: 0.95
    CreateMessageResp:
      type: object
      properties:
        id:
          type: string
          description: Unique ID of this response
        type:
          type: string
          description: Object type, fixed as `message`
          enum:
            - message
        role:
          type: string
          description: Role, fixed as `assistant`
          enum:
            - assistant
        model:
          type: string
          description: Model ID used for this request
        content:
          type: array
          description: List of response content blocks
          items:
            $ref: '#/components/schemas/ContentBlock'
        stop_reason:
          type: string
          description: |-
            Reason for stopping generation:
            - `end_turn`: Model ended naturally
            - `max_tokens`: Reached `max_tokens` limit
            - `stop_sequence`: Hit a stop sequence
          enum:
            - end_turn
            - max_tokens
            - stop_sequence
        usage:
          $ref: '#/components/schemas/Usage'
        base_resp:
          type: object
          description: Error status code and details
          properties:
            status_code:
              type: integer
              format: int64
              description: >-
                Status code


                - `1000`: Unknown error

                - `1001`: Request timeout

                - `1002`: Rate limit triggered

                - `1004`: Authentication failed

                - `1008`: Insufficient balance

                - `1013`: Internal server error

                - `1027`: Output content error

                - `1039`: Token limit exceeded

                - `2013`: Parameter error


                For more details, see [Error Code
                Reference](/api-reference/errorcode)
            status_msg:
              type: string
              description: Error details
    StreamEvent:
      type: object
      description: ''
      required:
        - type
      properties:
        type:
          type: string
          description: >-
            Event type:

            - `message_start`: Message start, contains complete message metadata

            - `ping`: Heartbeat event

            - `content_block_start`: Content block start

            - `content_block_delta`: Content block incremental update

            - `content_block_stop`: Content block end

            - `message_delta`: Message-level incremental update (e.g.,
            stop_reason)

            - `message_stop`: Message end
          enum:
            - message_start
            - ping
            - content_block_start
            - content_block_delta
            - content_block_stop
            - message_delta
            - message_stop
        message:
          type: object
          description: Message object (returned when `type` is `message_start`)
          properties:
            id:
              type: string
              description: Unique ID of the message
            type:
              type: string
              enum:
                - message
            role:
              type: string
              enum:
                - assistant
            content:
              type: array
              description: Content block list, initially an empty array
              items:
                $ref: '#/components/schemas/ContentBlock'
            model:
              type: string
              description: Model ID
            stop_reason:
              type: string
              nullable: true
              description: Stop reason, null at the start of streaming
            stop_sequence:
              type: string
              nullable: true
              description: Stop sequence, null at the start of streaming
            usage:
              $ref: '#/components/schemas/Usage'
            service_tier:
              type: string
              description: Service tier
        index:
          type: integer
          description: >-
            Index of the content block (returned for `content_block_start`,
            `content_block_delta`, `content_block_stop`)
        content_block:
          $ref: '#/components/schemas/ContentBlock'
          description: Content block object (returned when `type` is `content_block_start`)
        delta:
          type: object
          description: >-
            Incremental update content (returned for `content_block_delta` or
            `message_delta`)
          properties:
            type:
              type: string
              description: Delta type, e.g., `text_delta`
            text:
              type: string
              description: Incremental text content
            stop_reason:
              type: string
              description: Stop reason (returned for `message_delta`)
              enum:
                - end_turn
                - max_tokens
                - stop_sequence
        usage:
          $ref: '#/components/schemas/Usage'
          description: Token usage (returned for `message_delta`)
    Message:
      type: object
      required:
        - role
        - content
      properties:
        role:
          type: string
          enum:
            - user
            - assistant
            - user_system
            - group
            - sample_message_user
            - sample_message_ai
          description: |-
            Role of the message sender
            - `user`: User input
            - `assistant`: Model's historical reply
            - `user_system`: Set the user's role and persona
            - `group`: Name of the conversation
            - `sample_message_user`: Example user input
            - `sample_message_ai`: Example model output
        content:
          description: Message content, supports plain text string or content block array
          oneOf:
            - type: string
              description: Plain text message
            - type: array
              description: Content block array, supports text and thinking content blocks
              items:
                $ref: '#/components/schemas/ContentBlock'
    ContentBlock:
      type: object
      required:
        - type
      properties:
        type:
          type: string
          description: |-
            Content block type:
            - `text`: Text content
            - `thinking`: Model thinking process
          enum:
            - text
            - thinking
        text:
          type: string
          description: Text content (when `type` is `text`)
        thinking:
          type: string
          description: Model's thinking process content (when `type` is `thinking`)
        signature:
          type: string
          description: Signature of the thinking content (when `type` is `thinking`)
    Usage:
      type: object
      description: Token usage for this request
      properties:
        input_tokens:
          type: integer
          description: Number of tokens consumed by input
        output_tokens:
          type: integer
          description: Number of tokens consumed by output
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: |-
        `HTTP: Bearer Auth`
         - Security Scheme Type: http
         - HTTP Authorization Scheme: Bearer API_key, used for account verification, can be viewed in [Account Management > API Keys](https://platform.minimax.io/user-center/basic-information/interface-key)

````