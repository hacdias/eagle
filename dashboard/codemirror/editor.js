import {EditorState, basicSetup} from "@codemirror/basic-setup"
import {markdown} from "@codemirror/lang-markdown"
import {EditorView, keymap} from "@codemirror/view"
import {defaultTabBinding} from "@codemirror/commands"

const textarea = document.querySelector("#content")
const editor = document.querySelector("#editor")

let view = new EditorView({
  state: EditorState.create({
    doc: textarea.value,
    extensions: [
      basicSetup,
      EditorView.lineWrapping,
      keymap.of([defaultTabBinding]),
      markdown()
    ]
  }),
  parent: editor,
})

textarea.form.addEventListener('submit', () => {
  textarea.value = view.state.doc.toString()
})
textarea.style.display = 'none'