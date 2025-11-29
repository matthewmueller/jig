import {
  Component,
  ComponentFactory,
  hydrate,
  render,
  ComponentChildren,
  toChildArray,
  VNode,
  isValidElement,
} from "preact"

type Route = {
  path: string
  pattern: RegExp
  slots: string[]
}

type InitialProps = {
  route: string
  path: string
  query: Record<string, string>
}

type Imported = {
  Page: ComponentModule
  route: Route
}

type ComponentModule<Props = {}> = {
  default: ComponentFactory<Props>
  load?: (initial: InitialProps) => Promise<Record<string, unknown>>
}

type Input = {
  Page: ComponentModule
  frames: ComponentModule[]
  Error: {
    // TODO: default error
    default?: ComponentFactory<{ message: string }>
    load?: (initial: InitialProps) => Promise<Record<string, unknown>>
  }
  route: Route
  clientPath: string
  target: HTMLElement
}

const DefaultErrorPage = ({ message }: { message: string }) => (
  <div>
    <Head>
      <title>An unexpected error occurred</title>
    </Head>
    <h1>An unexpected error occurred</h1>
    <pre>{message}</pre>
  </div>
)

export function hydrateDOM(input: Input) {
  // Only hydrate on boot
  const script = document.querySelector(
    'script[src="' + input.clientPath + '"]'
  )
  if (!script || script.hasAttribute("hydrated")) {
    return
  }

  // Hydrate from the static props
  const props = getStaticProps()
  hydrate(<input.Page.default {...props} />, input.target)
  script.setAttribute("hydrated", "true")

  // Listen for history.{push,replace} changes
  listen(input.target, (err, url) => {
    if (!input.Error.default) {
      // Render a fallback error. Should try to match the template in errpage.
      render(<DefaultErrorPage message={err.message} />, input.target)
      return
    }
    // TODO: call load()
    // TODO: spread the error (needs to be turned into a record first though)
    render(<input.Error.default message={err.message} />, input.target)
  })

  // Upon page load run the loader once more. This time with the actual URL
  // values.
  loadInitialPage(input, (err) => {
    // TODO: render error page
    if (err) {
      console.error("error during initial page load", err)
    }
    setReady(true)
  })
}

function loadInitialPage(input: Input, fn: (err: Error | void) => void) {
  if (!input.Page.load) {
    // Do nothing since we hydrated already
    return fn()
  }
  const initial: InitialProps = {
    route: input.route.path,
    path: location.pathname,
    query: parseQuery(location.search),
  }
  const slots = matchSlots(input.route, location.pathname)
  for (let slot in slots) {
    initial.query[slot] = slots[slot]
  }
  input.Page.load(initial)
    .then((props) => {
      render(<input.Page.default {...props} />, input.target)
      fn()
    })
    .catch(fn)
}

function parseQuery(search: string): Record<string, string> {
  const searchParams = new URLSearchParams(search)
  const query: Record<string, string> = {}
  for (let [key, value] of searchParams) {
    query[key] = value
  }
  return query
}

function getStaticProps(): Record<string, unknown> {
  const el = document.getElementById("elmo-props")
  if (!el) return {}
  const content = el.textContent
  if (!content) return {}
  try {
    return JSON.parse(content)
  } catch (err) {
    return {}
  }
}

function setReady(ready: boolean) {
  // @ts-expect-error arbitrarily attaching properties to the window isn't
  // permitted by Typescript, but this is perfectly safe to do. This is used
  // for services like prerender.io and also for our test suite to know
  // when the page is fully loaded.
  window.prerenderReady = ready
}

// Inspired by: http://github.com/lukeed/navaid
function listen(
  target: Element,
  onError: (err: Error, url: URL) => void
): () => void {
  // Use an <a> element to do forgiving url parsing
  const a = document.createElement("a")
  function parseUrl(url: URL | string): URL {
    if (typeof url === "string") {
      a.href = url
      return new URL(a.href)
    }
    return url
  }

  function loadUrl(url: URL, fn: (err: Error | void) => void) {
    const clientPath = toClientPath(url.pathname)
    import(clientPath)
      .then((imported) => renderPage(url, imported, fn))
      .then(() => window.scrollTo(0, 0))
      .catch((err) => {
        if (is404(err)) {
          // If we're unable to resolve a module, this means that we don't have
          // a custom 404 page, so it's not specific to Preact. In this case,
          // we'll fallback to a normal redirect.
          location.href = "/404"
          return
        }
        fn(err)
      })
  }

  function toClientPath(path: string): string {
    const parts = path.replace(/\/$/, "").split("/")
    const last = parts[parts.length - 1]
    if (!last) return "/index.js"
    switch (last) {
      case "edit":
      case "show":
      case "new":
      case "error":
        return path + ".js"
      default:
        return path + "/index.js"
    }
  }

  function renderPage(
    url: URL,
    { Page, route }: Imported,
    fn: (err: Error | void) => void
  ) {
    if (!Page.load) {
      render(<Page.default />, target)
      return fn()
    }
    // These props are for the transitioning page
    const initial: InitialProps = {
      route: route.path,
      path: url.pathname,
      query: parseQuery(url.search),
    }
    const slots = matchSlots(route, url.pathname)
    for (let slot in slots) {
      initial.query[slot] = slots[slot]
    }
    Page.load(initial)
      .then((props) => {
        render(<Page.default {...props} />, target)
        fn()
      })
      .catch((err) => {
        fn(err)
      })
  }

  // override pushState
  const pushState = history.pushState
  history.pushState = function (data, unused, url) {
    if (!url) return
    url = parseUrl(url)
    if (url.href == location.href) return
    setReady(false)
    loadUrl(url, (err) => {
      setReady(true)
      pushState.call(this, data, unused, url)
      // We still set the URL regardless of error because we want the back
      // button to allow users to be able to hit the back button after getting
      // an error.
      if (err) {
        return onError(err, url)
      }
    })
  }

  // override replaceState
  const replaceState = history.replaceState
  history.replaceState = function (data, unused, url) {
    if (!url) return
    url = parseUrl(url)
    if (url.href == location.href) return
    setReady(false)
    loadUrl(url, (err) => {
      setReady(true)
      replaceState.call(this, data, unused, url)
      // TODO: consider pushing instead of replacing if we got an error.
      if (err) {
        return onError(err, url)
      }
    })
  }

  function onPopState() {
    setReady(false)
    const url = parseUrl(document.location.href)
    loadUrl(url, (err) => {
      setReady(true)
      if (err) {
        return onError(err, url)
      }
    })
  }

  // Handle clicks on either an `<a href>` or `<button href>`
  // This behavior can be ignored with `e.stopPropagation()`
  function onClick(e: MouseEvent) {
    if (
      e.ctrlKey ||
      e.metaKey ||
      e.altKey ||
      e.shiftKey ||
      e.button ||
      e.defaultPrevented
    )
      return
    if (!e.target || !(e.target instanceof HTMLElement)) return
    const ancestor = e.target.closest("a") || e.target.closest("button")
    if (!ancestor) return
    const href = ancestor.getAttribute("href")
    if (!href || href == "#") {
      e.preventDefault()
      return
    }
    const target = ancestor.getAttribute("target")
    if (target && target !== "_self") return
    const url = parseUrl(href)
    if (url.host !== location.host) return
    e.preventDefault()
    history.pushState(null, "", a.href)
  }

  addEventListener("popstate", onPopState)
  addEventListener("click", onClick)

  // unlisten
  return () => {
    history.pushState = pushState
    history.replaceState = replaceState
    removeEventListener("popstate", onPopState)
    removeEventListener("click", onClick)
  }
}

function matchSlots(route: Route, path: string): Record<string, string> {
  const slots: Record<string, string> = {}
  path = path == "/" ? path : path.replace(/\/$/, "")
  const m = path.match(route.pattern)
  if (!m) return {}
  for (let i = 1; i < m.length; i++) {
    slots[route.slots[i - 1]] = m[1]
  }
  return slots
}

type StyleProps = {
  id: string
}

export class Style extends Component<StyleProps> {
  private static tags: Record<string, HTMLElement> = {}
  private static refs: Record<string, number> = {}

  componentWillMount(): void {
    Style.tags[this.props.id] =
      document.getElementById(this.props.id) ||
      this.appendStyleTag(this.props.id, this.props.children)
    if (!Style.refs[this.props.id]) {
      Style.refs[this.props.id] = 0
    }
    Style.refs[this.props.id]++
  }

  componentWillUnmount(): void {
    const style = Style.tags[this.props.id]
    if (!style || !style.parentElement) return
    Style.refs[this.props.id]--
    if (Style.refs[this.props.id] <= 0) {
      style.parentElement.removeChild(style)
      delete Style.tags[this.props.id]
      delete Style.refs[this.props.id]
    }
  }

  private appendStyleTag(id: string, children: ComponentChildren): HTMLElement {
    const style = document.createElement("style")
    style.id = id
    const css = toChildArray(children).join("\n")
    style.appendChild(document.createTextNode(css))
    document.head.appendChild(style)
    return style
  }

  render() {
    return null
  }
}

// Value meta types
// See: https://developer.mozilla.org/en-US/docs/Web/HTML/Element/meta
// TODO: dedupe with server
function headSelector(node: VNode): string | void {
  switch (node.type) {
    case "title":
      return "title"
    case "base":
      return "base"
    case "link":
      if ("href" in node.props) {
        return `link[href="${node.props.href}"]`
      }
    case "meta":
      if ("name" in node.props) {
        return `meta[name="${node.props.name}"]`
      } else if ("http-equiv" in node.props) {
        return `meta[http-equiv="${node.props["http-equiv"]}"]`
      } else if ("charset" in node.props) {
        return `meta[charset]`
      }
  }
}

function makeHeadElement(node: VNode): Element | null {
  render(node, headFragment)
  return headFragment.firstElementChild
}

const headFragment = document.createDocumentFragment()
const heads: Record<string, Element> = {}
const head = document.head

export class Head extends Component {
  componentWillMount(): void {
    this.update(this.props.children)
  }

  componentWillUnmount(): void {
    for (let child of toChildArray(this.props.children)) {
      if (!isValidElement(child)) {
        continue
      }
      const selector = headSelector(child)
      if (!selector) {
        continue
      }
      const oldElement = heads[selector]
      if (!oldElement || !oldElement.parentElement) {
        continue
      }
      oldElement.parentElement.removeChild(oldElement)
      delete heads[selector]
    }
  }

  componentWillUpdate(
    nextProps: Readonly<{ children?: ComponentChildren }>
  ): void {
    this.update(nextProps.children)
  }

  private updateTitle(element: Element | null) {
    if (!element) return
    const text = element.textContent || ""
    if (document.title === text) return
    document.title = text
  }

  private update(children?: ComponentChildren) {
    for (let child of toChildArray(children)) {
      if (!isValidElement(child)) {
        continue
      }
      const selector = headSelector(child)
      if (!selector) {
        continue
      }
      const oldElement = heads[selector] || head.querySelector(selector)
      const newElement = makeHeadElement(child)
      // Handle title in a special way
      if (selector === "title") {
        this.updateTitle(newElement)
        continue
      }
      // Replace any existing elements
      if (oldElement && oldElement.parentElement && newElement) {
        heads[selector] = newElement
      } else if (newElement) {
        head.appendChild(newElement)
        heads[selector] = newElement
      }
    }
  }

  render() {
    return null
  }
}

function is404(err: Error): boolean {
  if (err.message.startsWith("Failed to fetch dynamically imported module")) {
    return true
  }
  return true
}
