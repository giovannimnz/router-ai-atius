import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + '\n'
}

const newKeys = {
  en: {
    'Device authorization could not be started. Start again.':
      'Device authorization could not be started. Start again.',
    'Device authorization expired. Start again.':
      'Device authorization expired. Start again.',
    'Restart device authorization': 'Restart device authorization',
  },
  zh: {
    'Device authorization could not be started. Start again.':
      '无法启动设备授权。请重新开始。',
    'Device authorization expired. Start again.': '设备授权已过期。请重新开始。',
    'Restart device authorization': '重新启动设备授权',
  },
  fr: {
    'Device authorization could not be started. Start again.':
      'Impossible de démarrer l’autorisation de l’appareil. Recommencez.',
    'Device authorization expired. Start again.':
      'L’autorisation de l’appareil a expiré. Recommencez.',
    'Restart device authorization': 'Redémarrer l’autorisation de l’appareil',
  },
  ja: {
    'Device authorization could not be started. Start again.':
      'デバイス認可を開始できませんでした。もう一度開始してください。',
    'Device authorization expired. Start again.':
      'デバイス認可の有効期限が切れました。もう一度開始してください。',
    'Restart device authorization': 'デバイス認可を再開',
  },
  ru: {
    'Device authorization could not be started. Start again.':
      'Не удалось запустить авторизацию устройства. Начните снова.',
    'Device authorization expired. Start again.':
      'Срок действия авторизации устройства истёк. Начните снова.',
    'Restart device authorization': 'Перезапустить авторизацию устройства',
  },
  vi: {
    'Device authorization could not be started. Start again.':
      'Không thể bắt đầu ủy quyền thiết bị. Hãy bắt đầu lại.',
    'Device authorization expired. Start again.':
      'Ủy quyền thiết bị đã hết hạn. Hãy bắt đầu lại.',
    'Restart device authorization': 'Khởi động lại ủy quyền thiết bị',
  },
  pt: {
    'Device authorization could not be started. Start again.':
      'Não foi possível iniciar a autorização do dispositivo. Inicie novamente.',
    'Device authorization expired. Start again.':
      'A autorização do dispositivo expirou. Inicie novamente.',
    'Restart device authorization': 'Reiniciar autorização do dispositivo',
    'Back to Dashboard': 'Voltar ao dashboard',
    'Cost = model price × that one ratio. Nothing else from the group settings enters the formula.':
      'Custo = preço do modelo × aquele único multiplicador. Nada mais das configurações do grupo entra na fórmula.',
    'Failed to copy': 'Falha ao copiar',
    'Failed to copy channel': 'Falha ao copiar canal',
    'Failed to copy keys': 'Falha ao copiar chaves',
    'Failed to copy model names': 'Falha ao copiar nomes de modelos',
    'Failed to copy to clipboard': 'Falha ao copiar para a área de transferência',
    'Failed to update settings': 'Falha ao atualizar as configurações',
    'Filter Dashboard Models': 'Filtrar modelos do dashboard',
    'Loading content settings...': 'Carregando configurações de conteúdo...',
    'Loading maintenance settings...': 'Carregando configurações de manutenção...',
    'Loading settings...': 'Carregando configurações...',
    'No additional type-specific settings for this channel type.':
      'Nenhuma configuração adicional específica está disponível para este tipo de canal.',
    'No models match your current filters.':
      'Nenhum modelo corresponde aos filtros atuais.',
    'No models to copy': 'Nenhum modelo para copiar',
    'No records found. Try adjusting your filters.':
      'Nenhum registro encontrado. Tente ajustar os filtros.',
    'No results for "{{query}}". Try adjusting your search or filters.':
      'Nenhum resultado para "{{query}}". Tente ajustar a busca ou os filtros.',
    'No users available. Try adjusting your search or filters.':
      'Nenhum usuário disponível. Tente ajustar a busca ou os filtros.',
    'Please enable io.net model deployment service and configure an API key in System Settings.':
      'Ative o serviço de deploy de modelos do io.net e configure uma chave de API em Configurações do sistema.',
    'Return to dashboard': 'Voltar ao dashboard',
    'Set filters to customize your dashboard statistics and charts.':
      'Defina filtros para personalizar as estatísticas e os gráficos do dashboard.',
    'Set filters to narrow down your log search results.':
      'Defina filtros para refinar os resultados da busca de logs.',
    'Start a conversation to see messages here':
      'Inicie uma conversa para ver as mensagens aqui',
    'Use sidebar shortcut': 'Use o atalho na barra lateral',
  },
}

async function main() {
  let totalApplied = 0

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

    let count = 0
    for (const [key, value] of Object.entries(trans)) {
      if (json.translation[key] !== value) {
        json.translation[key] = value
        count++
      }
    }

    if (count > 0) {
      json.translation = Object.fromEntries(
        Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
      )
      await fs.writeFile(filePath, stableStringify(json), 'utf8')
    }

    console.log(`${locale}: ${count} translations applied`)
    totalApplied += count
  }

  console.log(`Total: ${totalApplied} translations applied`)
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
