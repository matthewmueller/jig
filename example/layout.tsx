import React from "react";

export type Props = {
  head: React.ReactNode;
  children: React.ReactNode;
};

export default function Layout(props: Props) {
  return (
    <html>
      <head>{props.head}</head>
      <body className="whatever">{props.children}</body>
    </html>
  );
}
