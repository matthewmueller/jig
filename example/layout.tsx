import React from "react"

export type Props = {
  title: string
  children?: React.ReactNode
}

export default function Layout(props: Props) {
  return (
    <html>
      <head>
        <title>{props.title}</title>
      </head>
      <body className="whatever">{props.children}</body>
    </html>
  )
}
