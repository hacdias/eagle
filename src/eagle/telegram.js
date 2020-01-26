const Telegram = require('telegraf/telegram')
const Telegraf = require('telegraf')

module.exports = function createTelegram ({ token, chatID, git, hugo }) {
  const tg = new Telegram(token)
  const bot = new Telegraf(token)

  bot.on('text', async ({ update, reply }) => {
    if (update.message.chat.id !== chatID) {
      return
    }

    const text = update.message.text
      .trim()
      .toLowerCase()

    switch (text) {
      case 'echo':
        return reply('echo')
      case 'push':
        try {
          git.push()
          reply('Pushed!')
        } catch (e) {
          sendError(e)
        }
        break
      case 'pull':
        try {
          git.pull()
          reply('Pulled!')
        } catch (e) {
          sendError(e)
        }
        break
      case 'build':
        try {
          hugo.build()
          reply('Built!')
        } catch (e) {
          sendError(e)
        }
        break
      case 'build clean':
        try {
          hugo.buildAndClean()
          reply('Built cleaned version!')
        } catch (e) {
          sendError(e)
        }
    }
  })

  bot.launch()

  const sendError = (e) => {
    const formatted = `An error occurred on the server\n\n${e.stack}`
    send(formatted)
  }

  const send = (msg) => {
    tg.sendMessage(chatID, msg)
  }

  return Object.freeze({
    sendError,
    send
  })
}
