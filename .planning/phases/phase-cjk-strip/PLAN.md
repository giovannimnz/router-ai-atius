# Phase: CJK Character Strip Post-Filter

**Problema:** MiniMax-M2.7 por vezes gera caracteres chineses/CJK (ex: "重新") no meio de respostas em português. O tokenizer BBPE do MiniMax inclui tokens CJK, e com temperature sampling o modelo ocasionalmente selecciona esses tokens erroneamente durante next-token prediction — especialmente em padrões de repetição ou contexto ambíguo.

**Solução:** Post-filter regex no router — remove caracteres CJK do response antes de devolver ao cliente. 100% eficaz, sem impacto na latência, não afecta nenhum character set além de CJK.

---

## Fase Boundary

Filtro post-response que stripping CJK characters de todos os text content fields antes de o response ser enviado ao cliente. Afecta apenas responses de texto — não afecta imagens, audio, ou outros content types.

---

## Decisions

### D-01: Regex vs Lista de Caracteres
- **Decision:** Usar regex Unicode ranges, não lista de caracteres
- **Rationale:** O regex `[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{3000}-\x{303f}\x{ff00}-\x{ffef}]` cobre todos os caracteres CJK relevantes com uma única passagem. Listas de caracteres seriam incompletas e mais lentas.
- **Ranges:**
  - `\x{4e00}-\x{9fff}` — CJK Unified Ideographs (~20k caracteres)
  - `\x{3400}-\x{4dbf}` — CJK Extension A
  - `\x{3000}-\x{303f}` — CJK Symbols and Punctuation
  - `\x{ff00}-\x{ffef}` — Halfwidth and Fullwidth Forms

### D-02: Per-Channel Setting vs Global Toggle
- **Decision:** Per-channel setting (`StripCJK bool` em `ChannelSettings`)
- **Rationale:** Sigue o padrão existente do codebase ( ThinkingToContent, ForceFormat são per-channel). Permite ativar apenas para MiniMax sem afetar outros canais. Operador pode ativar noutros canais se necessário.
- **Alternative rejected:** Global toggle — menos flexível, exige restart do router para ajustar.

### D-03: Injection Points
- **Decision:** Injeta nos handlers do OpenAI adaptor (OpenaiHandler e sendStreamData)
- **Rationale:** MiniMax usa o OpenAI-compatible relay (via `openai.Adaptor{}` em `DoResponse`). Estes são os pontos mais próximos do output final antes de ser enviado ao cliente. Não requer modificações nos canais nem nos conversion functions.
- **Non-streaming:** Strip do `simpleResponse.Choices[i].Message` antes do switch de formato
- **Streaming:** Strip do `lastStreamResponse.Choices[i].Delta` antes de `helper.ObjectData`

### D-04: Non-Streaming ResponseBody Handling
- **Decision:** Strip do `simpleResponse` struct (in-place) antes do switch de formato
- **Rationale:** Quando `forceFormat=true` ou conversão para Claude/Gemini, o código faz `common.Marshal(simpleResponse)` — se StripCJK modificar o struct antes, o marshal já produz bytes limpos. Para o caso default (forceFormat=false), o `responseBody` original é usado diretamente — o StripCJK no struct não afeta este path porque o código na linha 279 faz `break` (usa o original). **CORREÇÃO:** Para o caso `forceFormat=false`, precisamos de fazer strip dos `responseBody` bytes diretamente.
- **Implementation:** Após strip no struct, para o caso `forceFormat=false` fazer também `responseBody = common.Marshal(simpleResponse)` (reescrever responseBody com o struct já limpo).

---

## Implementation

### T-01: Adicionar StripCJK a common/str.go

**Read first:** `common/str.go`

**Action:** Adicionar no início do ficheiro, após os imports:

```go
var (
    maskURLPattern    = regexp.MustCompile(`(http|https)://[^\s/$.?#].[^\s]*`)
    // ... existing vars ...
    // cjkRegex matches all CJK (Chinese/Japanese/Korean) characters via Unicode ranges.
    cjkRegex = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{3000}-\x{303f}\x{ff00}-\x{ffef}]`)
)
```

Adicionar nova função no final do ficheiro:

```go
// StripCJK removes all CJK (Chinese/Japanese/Korean) characters from a string.
// Uses Unicode ranges: CJK Unified Ideographs, CJK Extension A, CJK Symbols/Punctuation,
// and Halfwidth/Fullwidth Forms. Returns the input string if no CJK characters are found.
func StripCJK(s string) string {
    return cjkRegex.ReplaceAllString(s, "")
}
```

**Verification:**
- `go build ./common/` não dá erros
- Teste manual: `StripCJK("Hello 世界 日本語 테스트")` → `"Hello  "` (caracteres CJK removidos, espaços preservados)

---

### T-02: Adicionar StripCJK a dto.ChannelSettings

**Read first:** `dto/channel_settings.go`

**Action:** Adicionar campo a `ChannelSettings`:

```go
type ChannelSettings struct {
    ForceFormat            bool   `json:"force_format,omitempty"`
    ThinkingToContent      bool   `json:"thinking_to_content,omitempty"`
    Proxy                  string `json:"proxy"`
    PassThroughBodyEnabled bool   `json:"pass_through_body_enabled,omitempty"`
    SystemPrompt           string `json:"system_prompt,omitempty"`
    SystemPromptOverride   bool   `json:"system_prompt_override,omitempty"`
    StripCJK               bool   `json:"strip_cjk,omitempty"` // NEW: remove CJK chars from response text
}
```

**Verification:**
- `go build ./dto/` não dá erros
- JSON marshal/unmarshal de `ChannelSettings` com `StripCJK: true` funciona

---

### T-03: Non-streaming handler — OpenaiHandler

**Read first:** `relay/channel/openai/relay-openai.go` (linhas 195-300)

**Action:** Após o processamento de `simpleResponse.Choices` (após linha 258, antes do `switch info.RelayFormat` na linha 262), adicionar:

```go
// StripCJK: remove CJK characters from all message content fields.
// Applies only when StripCJK is explicitly enabled on this channel setting.
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

**Verification:**
- Compilar sem erros: `go build ./relay/channel/openai/`
- Testar com response JSON contendo CJK — verificar que caracteres são removidos antes de IOCopyBytesGracefully

---

### T-04: Streaming handler — sendStreamData

**Read first:** `relay/channel/openai/relay-openai.go` (linhas 25-104)

**Action:** Antes do `return helper.ObjectData(c, lastStreamResponse)` na linha 103, adicionar:

```go
// StripCJK: remove CJK characters from delta content before sending.
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

**Também nos casos intermédios de think→content:**
- Linha ~68: após `response := lastStreamResponse.Copy()`, antes de `helper.ObjectData(c, response)`:
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
- Linha ~83: após `response.Choices[j].Delta.SetContentString("\n
</think>

\n")`:
  ```go
  if info.ChannelSetting.StripCJK {
      for j := range response.Choices {
          delta := &response.Choices[j].Delta
          if delta.Content != nil && *delta.Content != "" {
              stripped := common.StripCJK(*delta.Content)
              delta.Content = &stripped
          }
      }
  }
  ```

**Verification:**
- Compilar sem erros
- Testar streaming response com CJK no delta — verificar caracteres removidos

---

### T-05: Configurar MiniMax channel com StripCJK=true

**Read first:** Ficheiro de config do canal MiniMax (provider config ou channel settings)

**Action:** No channel config do MiniMax (ou no .env / config.yaml), ativar:

```json
{
  "strip_cjk": true
}
```

O mecanismo exacto depende de como os canais estão configurados neste deploy — verificar em `setting/channel.go` ou `.env` para a estrutura de config do MiniMax.

**Verification:**
- Reiniciar o router
- Fazer pedido de teste para MiniMax-M2.7 com prompt em português
- Verificar que a resposta não contém caracteres CJK

---

## Acceptance Criteria

1. **Funcionalidade:** Caracteres CJK (汉字, ひらがな, katakana, 한자) são removidos de todos os text content fields no response antes de ser enviado ao cliente
2. **Não-afetação:** Caracteres não-CJK (latinos, números, pontuação, emojis) são preservados na totalidade
3. **Latência:** Impacto < 1ms (regex compilado, uma única passagem por string)
4. **Scope:** Apenas canais com `StripCJK: true` são afetados; canais sem a opção continuam idênticos
5. **Build:** `go build ./...` compila sem erros
6. **Teste manual:** Pedido para MiniMax com prompt "Conte-me uma história curta em português" não produz caracteres chineses/japoneses/koreanos na resposta

---

## Canonical References

- `common/str.go` — onde adicionar StripCJK
- `dto/channel_settings.go` — onde adicionar opção per-channel
- `relay/channel/openai/relay-openai.go` — onde injetar nos handlers
- `relay/channel/openai/helper.go` — StreamScannerHandler callback flow

---

## Dependências

Nenhuma — usa apenas bibliotecas padrão Go (regexp, strings).
