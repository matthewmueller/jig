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

export type Page = {
  LayoutView?: ComponentType<any>;
  ErrorView?: ComponentType<any>;
  EntryView: ComponentType<any>;
  clientRoute?: string;
};

export type PathProps = Record<string, unknown>;

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

function renderToHtml(element: ReactNode): string | Error {
  try {
    return renderToString(element);
  } catch (err) {
    // TODO: make error if not already
    return err as Error;
  }
}

function renderError(page: Page, context: Context, error: Error): string {
  let { LayoutView, ErrorView } = page;
  ErrorView = ErrorView || defaultError;
  let contentHtml = renderToHtml(
    <Context.Provider value={context}>
      <ErrorView error={error} />
    </Context.Provider>,
  );
  if (contentHtml instanceof Error) {
    // Try again with the default error handler. Copy the page since it's
    // shared across renders.
    return renderError({ ...page, ErrorView: defaultError }, context, contentHtml);
  }
  // TODO: we can probably dedupe some of this
  const heads: JSX.Element[] = [];
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
  LayoutView = LayoutView || defaultLayout;
  let html = renderToHtml(
    <LayoutView head={<Fragment>{heads}</Fragment>}>
      <div id="root" dangerouslySetInnerHTML={{ __html: contentHtml }} />
    </LayoutView>,
  );
  if (html instanceof Error) {
    return renderError({ ...page, LayoutView: defaultLayout }, context, html);
  }
  return `<!doctype html>` + html;
}

export function renderHtml(page: Page, propsJSON: string): string {
  let { LayoutView, EntryView, clientRoute } = page;
  const props: unknown = JSON.parse(propsJSON);
  const context: Context = {
    styles: {},
    heads: {},
  };
  if (typeof props !== "object") {
    return renderError(
      page,
      context,
      new Error("react: props must be an object"),
    );
  }
  let contentHtml = renderToHtml(
    <Context.Provider value={context}>
      <EntryView {...props} />
    </Context.Provider>,
  );
  if (contentHtml instanceof Error) {
    return renderError(page, context, contentHtml);
  }
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
  LayoutView = LayoutView || defaultLayout;
  let html = renderToHtml(
    <LayoutView head={<Fragment>{heads}</Fragment>}>
      <div id="root" dangerouslySetInnerHTML={{ __html: contentHtml }} />
      {clientRoute && (
        <script
          key="react-props"
          type="application/json"
          id="react-props"
          dangerouslySetInnerHTML={{ __html: propsJSON }}
        ></script>
      )}
    </LayoutView>,
  );
  // TODO: dedupe with above
  if (html instanceof Error) {
    return renderError(page, context, html);
  }
  return `<!doctype html>` + html;
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
