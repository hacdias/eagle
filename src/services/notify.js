const Telegram = require('telegraf/telegram')

module.exports = function createNotify ({ chatID, token }) {
  const tg = new Telegram(token)

  const sendError = (e) => {
    const formatted = `An error occurred on the server\n\n${e.stack}`
    send(formatted)
  }

  const send = (msg) => {
    tg.sendMessage(chatID, msg, {
      disable_web_page_preview: true
    })
  }

  return Object.freeze({
    sendError,
    send
  })
}
