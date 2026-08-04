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
  byId('agent-prompt').textContent = window.clankSetupPrompt({ serviceURL: pointer.url, projectArg })
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
