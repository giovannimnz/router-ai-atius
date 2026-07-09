# Phase v1.7.1: CJK Strip Post-Filter

**Goal:** Implementar post-filter regex no router que remove caracteres CJK (chinês/japonês/koreano) de responses de texto antes de devolver ao cliente — eliminando "重新" e similares de respostas MiniMax em português.

**Mode:** standard

---

## Technical Context

### Arquitectura do Response Flow

O relay do new-api processa responses em dois caminhos:

**Non-streaming** (`OpenaiHandler`, `relay/channel/openai/relay-openai.go:195`):
```
upstream response body (bytes)
  → io.ReadAll → responseBody (bytes)
  → common.Unmarshal → simpleResponse (dto.OpenAITextResponse)
  → simpleResponse.Choices[i].Message.Content (string ou []any)
  → switch RelayFormat
      → common.Marshal(simpleResponse) OU
      → ResponseOpenAI2Claude → common.Marshal OU
      → ResponseOpenAI2Gemini → common.Marshal
  → service.IOCopyBytesGracefully → cliente
```

**Streaming** (`OaiStreamHandler` → `sendStreamData`, `relay/channel/openai/relay-openai.go:106`):
```
SSE chunk (JSON string)
  → common.Unmarshal → ChatCompletionsStreamResponse
  → choice.Delta.GetContentString() / GetReasoningContent()
  → SetContentString() (think-to-content conversion)
  → helper.ObjectData(c, lastStreamResponse) → cliente
```

**Canal MiniMax:** Usa `openai.Adaptor{}` em `DoResponse` (minimax/adaptor.go:168) — logo o filtro em `OpenaiHandler` e `sendStreamData` cobre todo otraffic MiniMax.

---

## Decisions

### D-01: Regex vs Lista de Caracteres
Regex Unicode ranges — uma passagem, cobre todos os CJK. Listas são incompletas e mais lentas.

Ranges: `\x{4e00}-\x{9fff}` (CJK Unified), `\x{3400}-\x{4dbf}` (Extension A), `\x{3000}-\x{303f}` (Symbols), `\x{ff00}-\x{ffef}` (Halfwidth Forms).

### D-02: Per-Channel vs Global Toggle
Per-channel via `ChannelSettings.StripCJK`. Padrão `false` — MiniMax activa explicitamente. Mantém outros canais unaffected. Operador pode ativar noutros canais se necessário.

### D-03: Injection Points
`OpenaiHandler` (non-streaming) e `sendStreamData` (streaming) — os pontos mais próximos do output final antes de `IOCopyBytesGracefully` / `helper.ObjectData`. Não requer mudanças em conversion functions nem em canais.

---

## Tasks

### T-01: StripCJK em common/str.go
**Read first:** `common/str.go`

Adicionar compiled regex como package-level var:
```go
var cjkRegex = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{3000}-\x{303f}\x{ff00}-\x{ffef}]`)
```

Adicionar função:
```go
// StripCJK removes all CJK (Chinese/Japanese/Korean) characters from s.
// Returns s unchanged if no CJK characters are found.
func StripCJK(s string) string {
    return cjkRegex.ReplaceAllString(s, "")
}
```

**Verification:** `go build ./common/` compila sem erros.

---

### T-02: StripCJK em dto.ChannelSettings
**Read first:** `dto/channel_settings.go`

Adicionar campo a `ChannelSettings`:
```go
type ChannelSettings struct {
    // ... existing fields ...
    StripCJK bool `json:"strip_cjk,omitempty"` // NEW
}
```

**Verification:** `go build ./dto/` compila sem erros.

---

### T-03: Non-streaming — OpenaiHandler
**Read first:** `relay/channel/openai/relay-openai.go` linhas 195-300

**Onde:** Após linha 258 (após processamento de `simpleResponse.Choices`, antes do `switch info.RelayFormat` na linha 262).

**Código a adicionar:**
```go
// StripCJK: remove CJK characters from all message content fields.
if info.ChannelSetting.StripCJK {
    for i := range simpleResponse.Choices {
        msg := &simpleResponse.Choices[i].Message
        switch c := msg.Content.(type) {
        case string:
            msg.Content = common.StripCJK(c)
        case []any:
            for j, item := range c {
                if m, ok := item.(map[string]any); ok {
                    if m["type"] == "text" {
                        if text, ok := m["text"].(string); ok {
                            m["text"] = common.StripCJK(text)
                            c[j] = m
                        }
                    }
                }
            }
            msg.Content = c
        }
    }
    // Re-marshal so responseBody is also clean for the forceFormat=false path
    responseBody, _ = common.Marshal(simpleResponse)
}
```

**Nota:** `responseBody, _ = common.Marshal(simpleResponse)` depois do strip garante que tanto o caso `forceFormat=true` (usa Marshal direto) como `forceFormat=false` (usa responseBody) ficam com conteúdo limpo.

**Verification:** `go build ./relay/channel/openai/` compila sem erros.

---

### T-04: Streaming — sendStreamData
**Read first:** `relay/channel/openai/relay-openai.go` linhas 25-104

**Pontos de injecção:**

**4a. Antes do return final** (antes de linha 103, `return helper.ObjectData(c, lastStreamResponse)`):
```go
if info.ChannelSetting.StripCJK && lastStreamResponse.Choices != nil {
    for i := range lastStreamResponse.Choices {
        delta := &lastStreamResponse.Choices[i].Delta
        if delta.Content != nil && *delta.Content != "" {
            stripped := common.StripCJK(*delta.Content)
            delta.Content = &stripped
        }
    }
}
```

**4b. Antes de cada ObjectData intermédio** (linhas ~68 e ~88, nos casos de think-to-content):
```go
if info.ChannelSetting.StripCJK {
    for i := range response.Choices {
        delta := &response.Choices[i].Delta
        if delta.Content != nil && *delta.Content != "" {
            stripped := common.StripCJK(*delta.Content)
            delta.Content = &stripped
        }
    }
}
```
Aplicar antes de cada `helper.ObjectData(c, response)` nas transições de think→content (linhas 68 e 88).

**Verification:** `go build ./relay/channel/openai/` compila sem erros.

---

### T-05: Configurar canal MiniMax
**Dependência:** T-02 (ChannelSettings tem StripCJK)

Activar `StripCJK: true` no channel config do MiniMax. O mecanismo depende de como os canais estão configurados neste deploy — verificar em `setting/channel.go` ou `.env`.

```json
// Exemplo: no channel config do MiniMax (canal 1)
{
  "strip_cjk": true
}
```

**Verification:**
- Reiniciar router
- `curl` de teste para MiniMax-M2.7-hs com prompt em PT-BR
- Inspecionar response — nenhum caracter CJK

---

## Verification

1. **Build:** `go build ./...` compila sem erros
2. **Unit test (manual):**
   ```go
   assert.Equal(t, "Hello 太郎 ", StripCJK("Hello 太郎 世界"))  // espaços preservados
   assert.Equal(t, "Hello ", StripCJK("Hello 日本語"))
   assert.Equal(t, "test", StripCJK("test"))  // sem CJK = inalterado
   ```
3. **Integração:**
   - Request MiniMax-M2.7-hs com prompt em PT-BR
   - Response não contém caracteres CJK
   - Response texto não-CJK (acentos, emojis, pontuação) preservado

---

## Files Modified

| Ficheiro | Change |
|---|---|
| `common/str.go` | +cjkRegex var, +StripCJK() |
| `dto/channel_settings.go` | +StripCJK bool |
| `relay/channel/openai/relay-openai.go` | StripCJK in OpenaiHandler + sendStreamData |

---

## Must-Haves (Goal-Backward)

- [ ] `common.StripCJK()` compile e passe testes com strings CJK
- [ ] `StripCJK` config項 presente em `ChannelSettings`
- [ ] Non-streaming responses MiniMax não contêm caracteres CJK
- [ ] Streaming responses MiniMax não contêm caracteres CJK
- [ ] `go build ./...` compila sem erros
- [ ] Router inicia sem erros com `StripCJK: true` no canal MiniMax
