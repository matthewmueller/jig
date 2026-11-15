// Turbolinks-style navigation: same-origin links fetch the next page's HTML
// and swap it in, so the code that's already loaded is reused. Every page
// bundle includes this file, so only the first copy to run listens.
// Ported from elmo's router.
if (!window.__jig_navigate) {
  window.__jig_navigate = true
  history.scrollRestoration = "manual"

  // The rendered page, to tell hash changes apart from page changes
  let page = location.pathname + location.search
  let controller

  // Scripts parsed by DOMParser never run, so re-create them
  function recreate(script) {
    const el = document.createElement("script")
    for (const attr of script.attributes) el.setAttribute(attr.name, attr.value)
    el.textContent = script.textContent
    return el
  }

  async function visit(url, push, scrollY = 0) {
    // Only the latest visit renders
    controller?.abort()
    controller = new AbortController()
    let res, html
    try {
      res = await fetch(url, { signal: controller.signal })
      if (!res.headers.get("content-type")?.startsWith("text/html")) {
        throw new Error("jig: not an html page")
      }
      html = await res.text()
    } catch (err) {
      // Let the browser load what can't be swapped in
      if (err.name !== "AbortError") location.href = url
      return
    }
    const doc = new DOMParser().parseFromString(html, "text/html")

    // Keep this entry's scroll position for the back button, then point the
    // URL at the new page before its scripts run. Redirects change the URL,
    // but the response doesn't keep the hash.
    const next = new URL(res.url)
    next.hash = new URL(url, location.href).hash
    if (push) {
      history.replaceState({ scrollY: window.scrollY }, "")
      history.pushState({}, "", next)
    }
    page = next.pathname + next.search

    // Let frameworks unmount the current page
    document.dispatchEvent(new Event("jig:unload"))

    // Merge the head. Scripts stay, since modules only run once per document.
    const heads = new Set([...doc.head.children].map((el) => el.outerHTML))
    const kept = new Set()
    for (const el of [...document.head.children]) {
      if (el.tagName === "SCRIPT" || heads.has(el.outerHTML)) {
        kept.add(el.outerHTML)
      } else {
        el.remove()
      }
    }
    for (const el of [...doc.head.children]) {
      if (kept.has(el.outerHTML)) continue
      document.head.append(el.tagName === "SCRIPT" ? recreate(el) : el)
    }

    // Swap the body and run its scripts, which mount the page
    document.body.replaceWith(doc.body)
    for (const script of document.body.querySelectorAll("script")) {
      script.replaceWith(recreate(script))
    }

    const target = next.hash && document.getElementById(next.hash.slice(1))
    if (target) target.scrollIntoView()
    else window.scrollTo(0, scrollY)
  }

  addEventListener("click", (e) => {
    if (e.ctrlKey || e.metaKey || e.altKey || e.shiftKey || e.button || e.defaultPrevented) return
    const a = e.target instanceof Element && e.target.closest("a[href]")
    if (!a || a.hasAttribute("download")) return
    const target = a.getAttribute("target")
    if (target && target !== "_self") return
    const url = new URL(a.getAttribute("href"), location.href)
    if (url.origin !== location.origin) return
    // Let the browser scroll to anchors on this page
    if (url.hash && url.pathname + url.search === page) return
    e.preventDefault()
    visit(url.href, true)
  })

  addEventListener("popstate", (e) => {
    // Hash changes on this page don't need a visit
    if (location.pathname + location.search === page) return
    visit(location.href, false, e.state?.scrollY)
  })
}
