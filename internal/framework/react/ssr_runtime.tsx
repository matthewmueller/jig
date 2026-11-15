import React from "react";
import {
  ComponentType,
  createContext,
  JSX,
  Fragment,
  ReactNode,
  ReactElement,
  Component,
  ContextType,
  Children,
  PropsWithChildren,
  isValidElement,
} from "react";
import { renderToString } from "react-dom/server";

type Context = {
  styles: Record<string, string>;
  heads: Record<string, JSX.Element>;
};

const Context = createContext<Context>({
  styles: {},
  heads: {},
});

type Views = Record<string, ComponentType<any>>;

// A layout with its props
type Layout = {
  View: ComponentType<any>;
  props: object;
};

type LayoutProps = {
  head: JSX.Element;
  children: React.ReactNode;
};

function defaultLayout(props: LayoutProps) {
  return (
    <html>
      <head>{props.head}</head>
      <body>{props.children}</body>
    </html>
  );
}

type ErrorProps = {
  error: Error;
};

function defaultError(props: ErrorProps) {
  return <div>An error occurred: {props.error.message}</div>;
}

// Wraps pages that are rendered without a layout
const noLayout: Layout = { View: defaultLayout, props: {} };

function renderToHtml(element: ReactNode): string | Error {
  try {
    return renderToString(element);
  } catch (err) {
    // TODO: make error if not already
    return err as Error;
  }
}

// The head tags and styles collected while rendering the root. Pages that
// hydrate pass their clientRoute.
function headOf(context: Context, clientRoute?: string): JSX.Element {
  const heads: JSX.Element[] = [];
  if (clientRoute) {
    heads.push(
      <script key={clientRoute} defer type="module" src={clientRoute} />,
    );
  }
  for (let key in context.heads) {
    heads.push(context.heads[key]);
  }
  for (let id in context.styles) {
    heads.push(
      <style
        key={id}
        id={id}
        dangerouslySetInnerHTML={{ __html: context.styles[id] }}
      />,
    );
  }
  return <Fragment>{heads}</Fragment>;
}

// Render the layout around the root. The head and children win over props of
// the same name.
function renderDocument(
  layout: Layout,
  head: JSX.Element,
  root: ReactNode,
): string | Error {
  const { View, props } = layout;
  const html = renderToHtml(
    <View {...props} head={head}>
      {root}
    </View>,
  );
  if (html instanceof Error) {
    return html;
  }
  return `<!doctype html>` + html;
}

function renderError(
  layout: Layout,
  ErrorView: ComponentType<any>,
  context: Context,
  error: Error,
): string {
  const contentHtml = renderToHtml(
    <Context.Provider value={context}>
      <ErrorView error={error} />
    </Context.Provider>,
  );
  if (contentHtml instanceof Error) {
    // Try again with the default error view
    return renderError(layout, defaultError, context, contentHtml);
  }
  const html = renderDocument(
    layout,
    headOf(context),
    <div id="root" dangerouslySetInnerHTML={{ __html: contentHtml }} />,
  );
  if (html instanceof Error) {
    // Try again with the default layout
    return renderError(noLayout, ErrorView, context, html);
  }
  return html;
}

// Render a stack of views, outermost first, with the JSON props of each. The
// last view is the page, which the browser hydrates with its props. A layout
// may come before it, which wraps the root and only renders on the server.
export function renderHtml(
  views: Views,
  errors: Views,
  names: string[],
  props: string[],
): string {
  const last = names.length - 1;
  const layout =
    last > 0
      ? { View: views[names[0]], props: JSON.parse(props[0]) }
      : noLayout;
  const PageView = views[names[last]];
  const ErrorView = errors[names[last]] || defaultError;
  const pageProps: unknown = JSON.parse(props[last]);
  const context: Context = {
    styles: {},
    heads: {},
  };
  if (typeof pageProps !== "object") {
    return renderError(
      layout,
      ErrorView,
      context,
      new Error("react: props must be an object"),
    );
  }
  const contentHtml = renderToHtml(
    <Context.Provider value={context}>
      <PageView {...pageProps} />
    </Context.Provider>,
  );
  if (contentHtml instanceof Error) {
    return renderError(layout, ErrorView, context, contentHtml);
  }
  const html = renderDocument(
    layout,
    headOf(context, "/" + names[last] + ".js"),
    <Fragment>
      <div id="root" dangerouslySetInnerHTML={{ __html: contentHtml }} />
      {/* Embedded as Go encoded it, which escapes "<" for the script tag */}
      <script
        type="application/json"
        id="react-props"
        dangerouslySetInnerHTML={{ __html: props[last] }}
      ></script>
    </Fragment>,
  );
  if (html instanceof Error) {
    return renderError(layout, ErrorView, context, html);
  }
  return html;
}

export class Head extends Component<PropsWithChildren> {
  static contextType = Context;
  declare context: ContextType<typeof Context>;

  // Value meta types
  // See: https://developer.mozilla.org/en-US/docs/Web/HTML/Element/meta
  // TODO: dedupe with client
  private headSelector(node: ReactElement): string | void {
    switch (node.type) {
      case "title":
        return "title";
      case "base":
        return "base";
      case "link":
        if (!node.props || typeof node.props !== "object") {
          return;
        }
        if ("href" in node.props) {
          return `link[href="${node.props.href}"]`;
        }
      case "meta":
        if (!node.props || typeof node.props !== "object") {
          return;
        }
        if ("name" in node.props) {
          return `meta[name="${node.props.name}"]`;
        } else if ("http-equiv" in node.props) {
          return `meta[http-equiv="${node.props["http-equiv"]}"]`;
        } else if ("charSet" in node.props) {
          return `meta[charset]`;
        }
    }
  }

  render() {
    Children.forEach(this.props.children, (child) => {
      if (!isValidElement(child)) {
        return;
      }
      const id = this.headSelector(child);
      if (!id) {
        return;
      }
      this.context.heads[id] = child;
    });
    return null;
  }
}

export class Style extends Component<PropsWithChildren<{ id: string }>> {
  static contextType = Context;
  declare context: ContextType<typeof Context>;

  render() {
    const style = Children.toArray(this.props.children).join("\n");
    this.context.styles[this.props.id] = style;
    return null;
  }
}
