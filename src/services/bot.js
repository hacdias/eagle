const Telegraf = require('telegraf')

module.exports = function createTelegram ({ token, chatID, git, hugo, activitypub }) {
  const actions = {
    echo: ({ reply }) => reply('echo'),
    push: async ({ reply }) => {
      const { stdout } = await git.push()
      reply(`Pushed!\n\`\`\`\n${stdout}\n\`\`\``, {
        parse_mode: 'Markdown'
      })
    },
    pull: async ({ reply }) => {
      const { stdout } = await git.pull()
      reply(`Pulled!\n\`\`\`\n${stdout}\n\`\`\``, {
        parse_mode: 'Markdown'
      })
    },
    build: async ({ reply }, parts) => {
      if (parts[1] && parts[1].trim() === 'clean') {
        await hugo.buildAndClean()
        reply('Built cleaned version!')
        return
      }

      await hugo.build()
      reply('Built!')
    },
    activity: async ({ reply }, parts) => {
      if (!parts[1]) {
        reply('Must provide a post')
        return
      }

      activitypub.postArticle(parts[1].trim())
      reply('Post sent to followers!')
    }
  }

  const bot = new Telegraf(token)

  bot.on('text', async (event) => {
    if (event.update.message.chat.id !== chatID) {
      return
    }

    const parts = event.message.text
      .trim()
      .split(' ', 2)

    const text = parts[0]
      .toLowerCase()

    const fn = actions[text]
    if (fn) {
      try {
        await fn(event, parts)
      } catch (e) {
        event.reply(e)
      }
    }
  })

  bot.launch()
}
