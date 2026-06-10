import { mount } from "svelte"
import App from "./App.svelte"
import { connect } from "./stores/ws"
import "./app.css"
import "./styles/glass.css"

// Initialize theme store (applies persisted theme via data-theme attribute)
import "./stores/theme.svelte"

const target = document.getElementById("app")
if (target) {
  mount(App, { target })
  console.log("Agent-OS mounted")
  connect()
}
