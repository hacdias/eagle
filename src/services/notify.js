const debug = require('debug')('eagle:notify')
const Telegram = require('telegraf/telegram')

module.exports = function createNotify ({ telegramChatId, telegramToken }) {
  const tg = new Telegram(telegramToken)

  const sendError = (err) => {
    const formatted = `An error occurred:\n\`\`\`\n${err.stack}\n\`\`\``
    send(formatted)
  }

  const send = async (msg) => {
    try {
      await tg.sendMessage(telegramChatId, msg, {
        parse_mode: 'Markdown',
        disable_web_page_preview: true
      })
    } catch (e) {
      debug('could not send message: %s', e.stack)
    }
  }

  return Object.freeze({
    sendError,
    send
  })
}
