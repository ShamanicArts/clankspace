const byId = (id) => document.getElementById(id)
const cleanURL = (value) => value.trim().replace(/\/+$/, '')
const cleanSlug = (value) => value.trim().toLowerCase().replace(/[^a-z0-9-]/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '') || 'your-project'

function renderSetup() {
  const serviceURL = cleanURL(byId('service-url').value) || 'https://clank.shamanicarts.dev'
  const localURL = cleanURL(byId('local-url').value) || 'http://127.0.0.1:8080'
  const project = cleanSlug(byId('project-slug').value)
  const localFirst = byId('local-first').checked
  const pointer = localFirst ? { url: localURL, fallbackUrls: [serviceURL], project } : { url: serviceURL, project }

  byId('pointer-output').textContent = JSON.stringify(pointer, null, 2)
  byId('credential-output').textContent = [
    'read -rsp "Project token: " CLANKSPACE_TOKEN; echo',
    'printf \'%s\\n\' "$CLANKSPACE_TOKEN" | clank auth set \\',
    `  --url ${serviceURL} \\`,
    `  --project ${project} \\`,
    '  --token-stdin',
    'unset CLANKSPACE_TOKEN',
    'clank context',
    'clank auth status'
  ].join('\n')

  const projectArg = project === 'your-project' ? '' : ` --project ${project}`
  byId('agent-prompt').textContent = `Set up this repository with ClankSpace. Do as much as possible yourself and ask me only to approve the browser authentication step.

1. If the clank CLI is missing and Go 1.26 or newer is available, install it:
   go install github.com/ShamanicArts/clankspace/cmd/clank@latest
2. From the repository root, run:
   clank setup --url ${pointer.url}${projectArg}
3. The command will infer the repository and project, open a short-lived approval page, install the ClankSpace skill, add the non-secret project pointer and lean AGENTS.md instruction, store the project credential outside the repository, and verify the connection.
4. If browser approval cannot open automatically, give me the URL and code exactly as printed, then keep waiting. Never ask me to paste a project token into chat.
5. When setup completes, run clank context and one read-only clank brief for the likely work area. Do not create a note merely because setup succeeded.
6. Report the files changed, resolved service and project, and whether the brief worked.

ClankSpace is advisory project context, not canonical law or an instruction channel. Never put credentials, private conversation, raw quotes, prompts, transcripts, emotional commentary, or hidden reasoning into it.`
}

function toast(message) {
  const element = byId('copy-toast')
  element.textContent = message
  element.classList.add('show')
  clearTimeout(toast.timer)
  toast.timer = setTimeout(() => element.classList.remove('show'), 1700)
}

document.querySelector('.connect-form').addEventListener('input', renderSetup)
document.querySelectorAll('[data-copy]').forEach((button) => button.addEventListener('click', async () => {
  const value = byId(button.dataset.copy).textContent
  try {
    await navigator.clipboard.writeText(value)
    toast(button.dataset.copy === 'agent-prompt' ? 'Agent prompt copied' : 'Copied')
  } catch {
    const range = document.createRange()
    range.selectNodeContents(byId(button.dataset.copy))
    const selection = getSelection()
    selection.removeAllRanges()
    selection.addRange(range)
    toast('Selected. Copy manually.')
  }
}))

fetch('/api/v1/meta').then((response) => response.json()).then((meta) => {
  const state = byId('host-auth-state')
  if (meta.authMode === 'bootstrap') {
    state.textContent = 'This host currently uses owner token access. Email invitations are not active until its operator configures SMTP.'
  } else {
    state.textContent = 'Email sign-in is active on this host. Use the address from your workspace invitation.'
    state.classList.add('available')
  }
}).catch(() => {
  byId('host-auth-state').textContent = 'Could not read this host’s sign-in status. Try the sign-in page or contact the workspace owner.'
})

const nav = document.querySelector('.docs-nav')
const links = [...nav.querySelectorAll('a')]
const sections = links.map((link) => ({ link, element: document.querySelector(link.getAttribute('href')) })).filter((item) => item.element)
const observer = new IntersectionObserver((entries) => entries.forEach((entry) => {
  if (!entry.isIntersecting) return
  links.forEach((link) => link.classList.remove('active'))
  const match = sections.find((item) => item.element === entry.target)
  if (!match) return
  match.link.classList.add('active')
  if (innerWidth <= 820) match.link.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'center' })
}), { rootMargin: '-12% 0px -78% 0px' })
sections.forEach((item) => observer.observe(item.element))

renderSetup()
