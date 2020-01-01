const Telegram = require('telegraf/telegram')

module.exports = class TelegramService {
  constructor ({ token, chatID }) {
    this.chatID = chatID
    this.bot = new Telegram(token)
  }

  sendError (e) {
    const formatted = `An error occurred on the server\n\n${e.stack}`
    this.bot.sendMessage(this.chatID, formatted)
  }
}
