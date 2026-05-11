import { mount } from "svelte"
import App from "./App.svelte"
import { connect } from "./stores/ws"
import "./app.css"
import "./styles/glass.css"

const target = document.getElementById("app")
if (target) {
  mount(App, { target })
  console.log("Agent-OS mounted")
  connect()
}
