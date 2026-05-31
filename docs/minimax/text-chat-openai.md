> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/api-reference/text-chat-openai",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# Text Chat (Compatible OpenAI API)

> Use the OpenAI API compatible format to call MiniMax models, supporting role-playing, multi-turn conversations and other dialogue scenarios. Supports rich role settings (system, user_system, group, etc.) and example dialogue learning.



## OpenAPI

````yaml /api-reference/text/api/openapi-chat-openai.json POST /v1/chat/completions
openapi: 3.1.0
info:
  title: MiniMax Text API OpenAI
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
  /v1/chat/completions:
    post:
      tags:
        - Text Generation
      summary: Text Generation OpenAI
      operationId: chatCompletionOpenAI
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
              $ref: '#/components/schemas/ChatCompletionReq'
            examples:
              Request:
                value:
                  model: MiniMax-M2.7
                  messages:
                    - role: system
                      name: MiniMax AI
                    - role: user
                      name: User
                      content: Hello
              Stream:
                value:
                  model: MiniMax-M2.7
                  messages:
                    - role: system
                      name: MiniMax AI
                    - role: user
                      name: User
                      content: Hello
                  stream: true
        required: true
      responses:
        '200':
          description: ''
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ChatCompletionResp'
              examples:
                Request:
                  value:
                    id: 0637a03982880edad2460180345734fe
                    choices:
                      - finish_reason: stop
                        index: 0
                        message:
                          content: >-
                            <think>

                            The user just says "Hello". This is a simple
                            greeting. I should respond with a friendly greeting
                            and offer to help.

                            </think>


                            Hello! How can I help you today?
                          role: assistant
                          name: MiniMax AI
                          audio_content: ''
                    created: 1776839993
                    model: MiniMax-M2.7
                    object: chat.completion
                    usage:
                      total_tokens: 80
                      total_characters: 0
                      prompt_tokens: 42
                      completion_tokens: 38
                      completion_tokens_details:
                        reasoning_tokens: 29
                    input_sensitive: false
                    output_sensitive: false
                    input_sensitive_type: 0
                    output_sensitive_type: 0
                    output_sensitive_int: 0
                    base_resp:
                      status_code: 0
                      status_msg: ''
                Stream:
                  value:
                    - id: 0637a0697354164d5c9d79cd97b388c8
                      choices:
                        - index: 0
                          delta:
                            content: |-
                              <think>
                              The user
                            role: assistant
                            name: MiniMax AI
                            audio_content: ''
                      created: 1776840041
                      model: MiniMax-M2.7
                      object: chat.completion.chunk
                      usage: null
                      input_sensitive: false
                      output_sensitive: false
                      input_sensitive_type: 0
                      output_sensitive_type: 0
                      output_sensitive_int: 0
                    - id: 0637a0697354164d5c9d79cd97b388c8
                      choices:
                        - index: 0
                          delta:
                            content: |2-
                               says "Hello". I should respond politely, greet them, and ask how I can help.
                              </think>

                              Hello! How can I help you today
                            role: assistant
                            name: MiniMax AI
                            audio_content: ''
                      created: 1776840041
                      model: MiniMax-M2.7
                      object: chat.completion.chunk
                      usage: null
                      input_sensitive: false
                      output_sensitive: false
                      input_sensitive_type: 0
                      output_sensitive_type: 0
                      output_sensitive_int: 0
                    - id: 0637a0697354164d5c9d79cd97b388c8
                      choices:
                        - finish_reason: stop
                          index: 0
                          delta:
                            content: '?'
                            role: assistant
                            name: MiniMax AI
                            audio_content: ''
                      created: 1776840041
                      model: MiniMax-M2.7
                      object: chat.completion.chunk
                      usage: null
                      input_sensitive: false
                      output_sensitive: false
                      input_sensitive_type: 0
                      output_sensitive_type: 0
                      output_sensitive_int: 0
            text/event-stream:
              schema:
                $ref: '#/components/schemas/ChatCompletionChunk'
              examples:
                Stream:
                  value:
                    - id: 0637a0697354164d5c9d79cd97b388c8
                      choices:
                        - index: 0
                          delta:
                            content: |-
                              <think>
                              The user
                            role: assistant
                            name: MiniMax AI
                            audio_content: ''
                      created: 1776840041
                      model: MiniMax-M2.7
                      object: chat.completion.chunk
                      usage: null
                      input_sensitive: false
                      output_sensitive: false
                      input_sensitive_type: 0
                      output_sensitive_type: 0
                      output_sensitive_int: 0
                    - id: 0637a0697354164d5c9d79cd97b388c8
                      choices:
                        - index: 0
                          delta:
                            content: |2-
                               says "Hello". I should respond politely, greet them, and ask how I can help.
                              </think>

                              Hello! How can I help you today
                            role: assistant
                            name: MiniMax AI
                            audio_content: ''
                      created: 1776840041
                      model: MiniMax-M2.7
                      object: chat.completion.chunk
                      usage: null
                      input_sensitive: false
                      output_sensitive: false
                      input_sensitive_type: 0
                      output_sensitive_type: 0
                      output_sensitive_int: 0
                    - id: 0637a0697354164d5c9d79cd97b388c8
                      choices:
                        - finish_reason: stop
                          index: 0
                          delta:
                            content: '?'
                            role: assistant
                            name: MiniMax AI
                            audio_content: ''
                      created: 1776840041
                      model: MiniMax-M2.7
                      object: chat.completion.chunk
                      usage: null
                      input_sensitive: false
                      output_sensitive: false
                      input_sensitive_type: 0
                      output_sensitive_type: 0
                      output_sensitive_int: 0
components:
  schemas:
    ChatCompletionReq:
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
        stream:
          type: boolean
          description: >-
            Whether to use streaming output, defaults to `false`. When set to
            `true`, the response will be returned in chunks
          default: false
        max_completion_tokens:
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
            1], default value for `MiniMax-M2.7` model is 1.0. Higher values
            produce more random output; lower values produce more deterministic
            output
          minimum: 0
          exclusiveMinimum: 0
          maximum: 1
          default: 1
        top_p:
          type: number
          format: double
          description: >-
            Sampling strategy, affects output randomness, value range (0, 1],
            default value for `MiniMax-M2.7` model is 0.95
          minimum: 0
          exclusiveMinimum: 0
          maximum: 1
          default: 0.95
        messages:
          type: array
          description: >-
            A list of messages containing the conversation history. For more
            details on message parameters, refer to [Text Chat
            Guide](/guides/text-chat)
          items:
            $ref: '#/components/schemas/Message'
    ChatCompletionResp:
      type: object
      properties:
        id:
          type: string
          description: Unique ID of this response
        choices:
          type: array
          description: List of response choices
          items:
            type: object
            properties:
              finish_reason:
                type: string
                description: >-
                  Reason for stopping generation: `stop` (natural ending),
                  `length` (reached `max_completion_tokens` limit)
                enum:
                  - stop
                  - length
              index:
                type: integer
                description: Index of the choice, starting from 0
              message:
                type: object
                description: Complete reply generated by the model
                required:
                  - content
                  - role
                properties:
                  content:
                    type: string
                    description: Text reply content
                  role:
                    type: string
                    description: Role, fixed as `assistant`
                    enum:
                      - assistant
        created:
          type: integer
          format: int64
          description: Unix timestamp (seconds) when the response was created
        model:
          type: string
          description: Model ID used for this request
        object:
          type: string
          description: >-
            Object type. `chat.completion` for non-streaming,
            `chat.completion.chunk` for streaming
          enum:
            - chat.completion
            - chat.completion.chunk
        usage:
          $ref: '#/components/schemas/Usage'
        input_sensitive:
          type: boolean
          description: >-
            Whether the input content triggered sensitive word detection. If the
            input content is severely inappropriate, the API will return a
            content violation error message with empty reply content
        input_sensitive_type:
          type: integer
          format: int64
          description: >-
            Type of sensitive word triggered by input, returned when
            input_sensitive is true. Values: 1 Severe violation; 2 Pornography;
            3 Advertising; 4 Prohibited; 5 Abuse; 6 Violence/Terrorism; 7 Other
        output_sensitive:
          type: boolean
          description: >-
            Whether the output content triggered sensitive word detection. If
            the output content is severely inappropriate, the API will return a
            content violation error message with empty reply content
        output_sensitive_type:
          type: integer
          format: int64
          description: Type of sensitive word triggered by output
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
    ChatCompletionChunk:
      type: object
      description: ''
      properties:
        id:
          type: string
          description: Unique ID of this response
        choices:
          type: array
          description: List of streaming response choices
          items:
            type: object
            properties:
              index:
                type: integer
                description: Index of the choice, starting from 0
              delta:
                type: object
                description: Incremental content
                properties:
                  role:
                    type: string
                    description: Role, fixed as `assistant`
                    enum:
                      - assistant
                  content:
                    type: string
                    description: Incremental text content
              finish_reason:
                type: string
                nullable: true
                description: >-
                  Reason for stopping generation, null when not finished: `stop`
                  (natural ending), `length` (reached `max_completion_tokens`
                  limit)
                enum:
                  - stop
                  - length
        created:
          type: integer
          format: int64
          description: Unix timestamp (seconds) when the response was created
        model:
          type: string
          description: Model ID used for this request
        object:
          type: string
          description: Object type, fixed as `chat.completion.chunk`
          enum:
            - chat.completion.chunk
        usage:
          $ref: '#/components/schemas/Usage'
          description: Token usage (only returned in the last chunk)
        input_sensitive_type:
          type: integer
          format: int64
          description: Type of sensitive word triggered by input
        output_sensitive:
          type: boolean
          description: Whether the output content triggered sensitive word detection
        output_sensitive_type:
          type: integer
          format: int64
          description: Type of sensitive word triggered by output
    Message:
      type: object
      required:
        - role
        - content
      properties:
        role:
          type: string
          enum:
            - system
            - user
            - assistant
            - user_system
            - group
            - sample_message_user
            - sample_message_ai
          description: |-
            Role of the message sender
            - `system`: Set the model's role and behavior
            - `user`: User input
            - `assistant`: Model's historical reply
            - `user_system`: Set the user's role and persona
            - `group`: Name of the conversation
            - `sample_message_user`: Example user input
            - `sample_message_ai`: Example model output
        name:
          type: string
          description: >-
            Name of the sender. If there are multiple roles of the same type, a
            specific name must be provided to distinguish them
        content:
          type: string
          description: Message content
    Usage:
      type: object
      description: Token usage statistics for this request
      properties:
        total_tokens:
          type: integer
          description: Total number of tokens consumed
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