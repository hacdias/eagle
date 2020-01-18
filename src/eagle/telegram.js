const Telegram = require('telegraf/telegram')

module.exports = function createTelegram ({ token, chatID }) {
  const bot = new Telegram(token)

  const sendError = (e) => {
    const formatted = `An error occurred on the server\n\n${e.stack}`
    bot.sendMessage(chatID, formatted)
  }

  return Object.freeze({
    sendError
  })
}
