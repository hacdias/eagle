const Telegraf = require('telegraf')

module.exports = function createTelegram ({ token, chatID, git, hugo }) {
  const actions = {
    echo: ({ reply }) => reply('echo'),
    push: async ({ reply }) => {
      const { stdout } = await git.push()
      reply(`Pushed!\n\n\`\`\`\n${stdout}\n\`\`\``, {
        parse_mode: 'Markdown'
      })
    },
    pull: async ({ reply }) => {
      const { stdout } = await git.pull()
      reply(`Pulled!\n\n\`\`\`\n${stdout}\n\`\`\``, {
        parse_mode: 'Markdown'
      })
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
