import {EditorState, basicSetup} from "@codemirror/basic-setup"
import {markdown} from "@codemirror/lang-markdown"
import {EditorView, keymap} from "@codemirror/view"
import {defaultTabBinding} from "@codemirror/commands"

const textarea = document.querySelector("#content")
const editor = document.querySelector("#editor")

// Sync: https://discuss.codemirror.net/t/codemirror-6-and-textareas/2731/3

let editor = new EditorView({
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
