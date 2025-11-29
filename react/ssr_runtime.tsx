import { ComponentType, createContext, JSX, Fragment } from "react"
import { renderToString } from "react-dom/server"

type Context = {
  styles: Record<string, string>
  heads: Record<string, JSX.Element>
}

const Context = createContext<Context>({
  styles: {},
  heads: {},
})

export type View = {
  path: string
  Component: ComponentType<any>
}

export type Page = {
  layout?: View
  error?: View
  frames: View[]
  content: View
  clientRoute?: string
  liveRoute?: string
}

export type PathProps = Record<string, unknown>

const defaultLayout: View = {
  path: "default_layout",
  Component: ({
    head,
    script,
    children,
  }: {
    head: JSX.Element
    script: JSX.Element
    children: React.ReactNode
  }) => (
    <html>
      <head>{head}</head>
      <body>
        {children}
        {script}
      </body>
    </html>
  ),
}

export function renderHtml(
  { layout, error, frames, content, clientRoute, liveRoute }: Page,
  pathProps: PathProps
) {
  const context: Context = {
    styles: {},
    heads: {},
  }
  const contentProps = pathProps[content.path] || {}
  const contentHTML = renderToString(
    <Context.Provider value={context}>
      <content.Component {...contentProps} />
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
  const scripts = []
  if (clientRoute) {
    scripts.push(
      <script
        type="text/template"
        id="react-props"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(contentProps) }}
      ></script>,
      <script async defer type="module" src={clientRoute} />
    )
  }
  if (liveRoute) {
    scripts.push(<script async defer type="module" src={liveRoute} />)
  }
  layout = layout || defaultLayout
  const html = renderToString(
    <layout.Component
      head={<Fragment>{heads}</Fragment>}
      script={<Fragment>{scripts}</Fragment>}
    >
      <div id="go-react" dangerouslySetInnerHTML={{ __html: contentHTML }} />
    </layout.Component>
  )
  return `<!doctype html>` + html
}
