const Telegraf = require('telegraf')

module.exports = function createTelegram ({ token, chatID, git, hugo }) {
  const actions = {
    echo: ({ reply }) => reply('echo'),
    push: ({ reply }) => {
      git.push()
      reply('Pushed!')
    },
    pull: ({ reply }) => {
      git.pull()
      reply('Pulled!')
    },
    build: ({ reply }) => {
      hugo.build()
      reply('Built!')
    },
    'build clean': ({ reply }) => {
      hugo.buildAndClean()
      reply('Built cleaned version!')
    }
  }

  const bot = new Telegraf(token)

  bot.on('text', async (event) => {
    if (event.update.message.chat.id !== chatID) {
      return
    }

    const text = event.message.text
      .trim()
      .toLowerCase()

    const fn = actions[text]
    if (fn) {
      try {
        await fn(event)
      } catch (e) {
        event.reply(e)
      }
    }
  })

  bot.launch()
}
