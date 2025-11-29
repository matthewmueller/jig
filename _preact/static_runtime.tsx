import {
  JSX,
  Component,
  ComponentFactory,
  createContext,
  VNode,
  toChildArray,
  ContextType,
  Fragment,
  isValidElement,
} from "preact"
import { render as renderToString } from "preact-render-to-string"

type Context = {
  styles: Record<string, string>
  heads: Record<string, JSX.Element>
}

const Context = createContext<Context>({
  styles: {},
  heads: {},
})

type ErrorProp = {
  message: string
}

type InitialProps = {
  route: string
  path: string
  query: Record<string, string>
  error?: ErrorProp
}

type ComponentModule<Props = {}> = {
  default: ComponentFactory<Props>
  load?: (initial: InitialProps) => Promise<Record<string, unknown>>
}

type Input = {
  Page: ComponentModule
  frames: ComponentModule[]
  Layout?: ComponentModule<LayoutProps>
  Error: ComponentModule
  clientPath: string
  route: string
  path: string
  props: Record<string, unknown>
}

export async function renderHTML({
  Page,
  Layout,
  clientPath,
  route,
  props,
}: Input): Promise<string> {
  const context: Context = {
    styles: {},
    heads: {},
  }
  const initial: InitialProps = {
    route: route,
    path: "",
    query: {},
  }
  if (!Page.default) {
    throw new Error(`page: ${route} doesn't export a default component`)
  }
  const loadedProps = Page.load ? await Page.load(initial) : props
  const pageHTML = renderToString(
    <Context.Provider value={context}>
      <Page.default {...loadedProps} />
    </Context.Provider>
  )
  const heads: JSX.Element[] = []
  for (let key in context.heads) {
    heads.push(context.heads[key])
  }
  for (let id in context.styles) {
    heads.push(
      <style id={id} dangerouslySetInnerHTML={{ __html: context.styles[id] }} />
    )
  }
  const scripts = [
    <script
      type="text/template"
      id="elmo-props"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(loadedProps) }}
    ></script>,
    <script async defer type="module" src={clientPath} />,
  ]
  Layout = Layout || { default: DefaultLayout }
  const html = renderToString(
    <Layout.default
      head={<Fragment>{heads}</Fragment>}
      script={<Fragment>{scripts}</Fragment>}
    >
      <div id="elmo" dangerouslySetInnerHTML={{ __html: pageHTML }} />
    </Layout.default>
  )
  return `<!doctype html>` + html
}

type LayoutProps = {
  head: JSX.Element
  script: JSX.Element
}

export class DefaultLayout extends Component<LayoutProps> {
  render() {
    return (
      <html>
        <head>
          {this.props.head}
          {this.props.script}
        </head>
        <body>{this.props.children}</body>
      </html>
    )
  }
}

export class Style extends Component<{ id: string }> {
  static contextType = Context
  declare context: ContextType<typeof Context>

  render() {
    const style = toChildArray(this.props.children).join("\n")
    this.context.styles[this.props.id] = style
    return null
  }
}

// Value meta types
// See: https://developer.mozilla.org/en-US/docs/Web/HTML/Element/meta
// TODO: dedupe with client
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

export class Head extends Component {
  static contextType = Context
  declare context: ContextType<typeof Context>

  render() {
    for (let child of toChildArray(this.props.children)) {
      if (!isValidElement(child)) {
        continue
      }
      const id = headSelector(child)
      if (!id) {
        continue
      }
      this.context.heads[id] = child
    }
    return null
  }
}
