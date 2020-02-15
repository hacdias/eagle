const Telegram = require('telegraf/telegram')
const debug = require('debug')('eagle:notify')

module.exports = function createNotify ({ chatID, token }) {
  const tg = new Telegram(token)

  const sendError = (e) => {
    const formatted = `An error occurred on the server\n\n${e.stack}`
    send(formatted)
  }

  const send = async (msg) => {
    try {
      await tg.sendMessage(chatID, msg, {
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
