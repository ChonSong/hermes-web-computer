import App from "./App.svelte"

console.log("Svelte 5 test mount starting...")
console.log("App:", typeof App)

const target = document.getElementById("app")
console.log("Target:", target)

if (target) {
  try {
    const app = new App({ target })
    console.log("App mounted successfully:", app)
  } catch (e: any) {
    console.error("Mount failed:", e)
    target.innerHTML = `<div style="color: red; padding: 20px;">Mount failed: ${e.message}</div>`
  }
} else {
  console.error("Target #app not found")
}
