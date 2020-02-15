const Telegram = require('telegraf/telegram')
const debug = require('debug')('eagle:notify')

module.exports = function createNotify ({ chatID, token }) {
  const tg = new Telegram(token)

  const sendError = (err) => {
    const formatted = `An error occurred:\n\`\`\`\n${err.stack}\n\`\`\``
    send(formatted)
  }

  const send = async (msg) => {
    try {
      await tg.sendMessage(chatID, msg, {
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
