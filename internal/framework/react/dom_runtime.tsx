import {
  ReactNode,
  ReactElement,
  Component,
  PropsWithChildren,
  isValidElement,
  Children,
} from "react"
import { renderToStaticMarkup } from "react-dom/server"

// Value meta types
// See: https://developer.mozilla.org/en-US/docs/Web/HTML/Element/meta
// TODO: dedupe with server
function headSelector(node: ReactElement): string | void {
  switch (node.type) {
    case "title":
      return "title"
    case "base":
      return "base"
    case "link":
      if (!node.props || typeof node.props !== "object") {
        return
      }
      if ("href" in node.props) {
        return `link[href="${node.props.href}"]`
      }
    case "meta":
      if (!node.props || typeof node.props !== "object") {
        return
      }
      if ("name" in node.props) {
        return `meta[name="${node.props.name}"]`
      } else if ("http-equiv" in node.props) {
        return `meta[http-equiv="${node.props["http-equiv"]}"]`
      } else if ("charset" in node.props) {
        return `meta[charset]`
      }
  }
}

const headElement = document.createElement("head")
const heads: Record<string, Element> = {}
const head = document.head

function makeHeadElement(node: ReactNode): Element | null {
  headElement.innerHTML = renderToStaticMarkup(node)
  return headElement.firstElementChild
}

export class Head extends Component<PropsWithChildren> {
  componentDidMount(): void {
    this.update(this.props.children)
  }

  componentWillUnmount(): void {
    for (let child of Children.toArray(this.props.children)) {
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

  componentDidUpdate(nextProps: Readonly<{ children?: ReactNode }>): void {
    this.update(nextProps.children)
  }

  private updateTitle(element: Element | null) {
    if (!element) return
    const text = element.textContent || ""
    if (document.title === text) return
    document.title = text
  }

  private update(children?: ReactNode) {
    for (let child of Children.toArray(children)) {
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
        if (!oldElement.isEqualNode(newElement)) {
          oldElement.parentElement.replaceChild(newElement, oldElement)
          heads[selector] = newElement
        }
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
