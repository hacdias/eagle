import {EditorState, basicSetup} from "@codemirror/basic-setup"
import {markdown} from "@codemirror/lang-markdown"
import {indentWithTab} from "@codemirror/commands"
import {EditorView, keymap} from "@codemirror/view"

const textarea = document.querySelector("#content")
const editor = document.querySelector("#editor")

const langs = {
  'markdown': markdown
}

const extensions = [
  basicSetup,
  EditorView.lineWrapping,
  keymap.of([indentWithTab]),
  markdown()
]

if (langs[textarea.dataset.lang]) {
  extensions.push(langs[textarea.dataset.lang]())
}

let view = new EditorView({
  state: EditorState.create({
    doc: textarea.value,
    extensions
  }),
  parent: editor,
})

textarea.form.addEventListener('submit', () => {
  textarea.value = view.state.doc.toString()
})
textarea.style.display = 'none'
