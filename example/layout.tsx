import React from "react"

export type Props = {
  head: React.ReactNode
  children: React.ReactNode
}

export default function Layout(props: Props) {
  return (
    <html>
      <head>{props.head}</head>
      <body className="whatever">
        <nav>
          <a href="/">Home</a> <a href="/about">About</a>{" "}
          <a href="/dashboard">Dashboard</a>
        </nav>
        {props.children}
      </body>
    </html>
  )
}
